#!/usr/bin/env python3
import os
import re
import json
import subprocess
import threading
import time
import math
from flask import Flask, jsonify, request, send_from_directory

app = Flask(__name__, static_folder="/var/www/html", static_url_path="")

RESULTS_FILE = "/tmp/gsm_results.json"
scanning = False
last_scan_time = None

# Turkish MCC/MNC operator mapping
OPERATORS = {
    "286-01": "Turkcell",
    "286-02": "Vodafone TR",
    "286-03": "Turk Telekom",
    "286-04": "Turkcell (Anadolu)",
}

# GSM900 downlink: 935-960 MHz (ARFCN 0-124)
# DCS1800 downlink: 1805-1880 MHz (ARFCN 512-885)
def freq_to_arfcn(freq_mhz):
    """Convert frequency (MHz) to ARFCN."""
    if 935.0 <= freq_mhz <= 960.0:
        return round((freq_mhz - 935.0) / 0.2)
    elif 1805.0 <= freq_mhz <= 1880.0:
        return round((freq_mhz - 1805.0) / 0.2) + 512
    return None

def arfcn_to_freq(arfcn):
    """Convert ARFCN to downlink frequency (MHz)."""
    if 0 <= arfcn <= 124:
        return 935.0 + arfcn * 0.2
    elif 512 <= arfcn <= 885:
        return 1805.0 + (arfcn - 512) * 0.2
    return None


def parse_rtl_power_output(raw_text, threshold=-10.0):
    """Parse rtl_power CSV output and find GSM channels above threshold."""
    channels = {}  # arfcn -> best power

    for line in raw_text.strip().split("\n"):
        line = line.strip()
        if not line or not line[0].isdigit():
            continue

        parts = line.split(",")
        if len(parts) < 7:
            continue

        try:
            # Format: date, time, freq_low, freq_high, bin_size, samples, db1, db2, ...
            freq_low = float(parts[2].strip())
            freq_high = float(parts[3].strip())
            bin_size = float(parts[4].strip())
            db_values = [float(x.strip()) for x in parts[6:]]

            for i, db in enumerate(db_values):
                freq_hz = freq_low + i * bin_size
                freq_mhz = freq_hz / 1e6
                arfcn = freq_to_arfcn(freq_mhz)

                if arfcn is not None and db > threshold:
                    if arfcn not in channels or db > channels[arfcn]["power"]:
                        band = "GSM900" if freq_mhz < 1000 else "DCS1800"
                        channels[arfcn] = {
                            "arfcn": arfcn,
                            "freq": round(freq_mhz, 1),
                            "band": band,
                            "power": round(db, 1),
                            "operator": "Bilinmeyen",
                        }
        except (ValueError, IndexError):
            continue

    # Merge adjacent bins into single channels (GSM channel = 200kHz)
    # Group by ARFCN and keep strongest
    result = list(channels.values())
    result.sort(key=lambda x: x["power"], reverse=True)
    return result


def run_scan(band="ALL"):
    """Run rtl_power scan in background."""
    global scanning, last_scan_time

    if scanning:
        return False

    scanning = True

    def do_scan():
        global scanning, last_scan_time
        try:
            results_all = []

            if band in ("ALL", "GSM900"):
                # GSM900 downlink: 935-960 MHz
                cmd = [
                    "rtl_power",
                    "-f", "935M:960M:200k",
                    "-g", "49",
                    "-i", "20",
                    "-1",
                    "-p", "0",
                ]
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
                raw = result.stdout + result.stderr
                parsed = parse_rtl_power_output(raw, threshold=-10.0)
                results_all.extend(parsed)

            if band in ("ALL", "DCS1800"):
                # DCS1800 downlink: 1805-1880 MHz
                cmd = [
                    "rtl_power",
                    "-f", "1805M:1880M:200k",
                    "-g", "49",
                    "-i", "20",
                    "-1",
                    "-p", "0",
                ]
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
                raw = result.stdout + result.stderr
                parsed = parse_rtl_power_output(raw, threshold=-5.0)
                results_all.extend(parsed)

            # Sort by power
            results_all.sort(key=lambda x: x["power"], reverse=True)

            # Try grgsm_scanner for cell info (optional, best effort)
            try_grgsm_decode(results_all)

            scan_data = {
                "results": results_all,
                "band": band,
                "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S"),
                "count": len(results_all),
            }

            with open(RESULTS_FILE, "w") as f:
                json.dump(scan_data, f)

            last_scan_time = time.time()
        except Exception as e:
            print(f"Scan error: {e}")
            # Save error state
            scan_data = {
                "results": [],
                "band": band,
                "timestamp": time.strftime("%Y-%m-%dT%H:%M:%S"),
                "count": 0,
                "error": str(e),
            }
            with open(RESULTS_FILE, "w") as f:
                json.dump(scan_data, f)
        finally:
            scanning = False

    t = threading.Thread(target=do_scan, daemon=True)
    t.start()
    return True


def try_grgsm_decode(results):
    """Try to decode cell info using grgsm_livemon for top channels. Best effort."""
    # This is optional - if grgsm works, we get MCC/MNC/LAC/CID
    # If not, we still have freq/power data from rtl_power
    for ch in results[:5]:  # Only try top 5 strongest
        try:
            freq_hz = int(ch["freq"] * 1e6)
            cmd = [
                "timeout", "10",
                "grgsm_livemon_headless",
                "-f", str(freq_hz),
                "-g", "49",
            ]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=15)
            raw = result.stdout + result.stderr

            # Try to parse system info from output
            mcc_match = re.search(r"MCC:\s*(\d+)", raw)
            mnc_match = re.search(r"MNC:\s*(\d+)", raw)
            lac_match = re.search(r"LAC:\s*(\d+)", raw)
            cid_match = re.search(r"(?:Cell ID|CID):\s*(\d+)", raw)

            if mcc_match:
                ch["mcc"] = mcc_match.group(1)
            if mnc_match:
                ch["mnc"] = mnc_match.group(1)
            if lac_match:
                ch["lac"] = int(lac_match.group(1))
            if cid_match:
                ch["cid"] = int(cid_match.group(1))

            if ch.get("mcc") and ch.get("mnc"):
                key = f"{ch['mcc']}-{ch['mnc'].zfill(2)}"
                ch["operator"] = OPERATORS.get(key, f"MCC:{ch['mcc']} MNC:{ch['mnc']}")
        except Exception:
            pass  # Best effort - rtl_power data is still valuable


def get_results():
    """Load last scan results."""
    if os.path.exists(RESULTS_FILE):
        try:
            with open(RESULTS_FILE) as f:
                return json.load(f)
        except Exception:
            pass
    return {"results": [], "count": 0}


@app.route("/")
def index():
    return send_from_directory("/var/www/html", "index.html")


@app.route("/api/scan", methods=["POST"])
def api_scan():
    data = request.get_json() or {}
    band = data.get("band", "ALL")
    if band not in ("ALL", "GSM900", "DCS1800"):
        band = "ALL"
    ok = run_scan(band)
    return jsonify({"started": ok, "band": band})


@app.route("/api/results")
def api_results():
    data = get_results()
    data["scanning"] = scanning
    return jsonify(data)


@app.route("/api/status")
def api_status():
    return jsonify({"scanning": scanning, "last_scan": last_scan_time})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
