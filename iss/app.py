#!/usr/bin/env python3
import os
import time
import math
import json
import subprocess
import threading
from datetime import datetime, timedelta
from pathlib import Path
from flask import Flask, jsonify, send_from_directory, send_file
from flask_socketio import SocketIO

try:
    import ephem
except ImportError:
    ephem = None

app = Flask(__name__, static_folder="/var/www/html", static_url_path="")
app.config["SECRET_KEY"] = "iss-tracker-secret"
socketio = SocketIO(app, cors_allowed_origins="*", async_mode="threading")

CAPTURES_DIR = "/opt/iss/captures"
TLE_FILE = "/tmp/iss_tle.txt"
STATION_LAT = os.environ.get("STATION_LAT", "40.94247")
STATION_LON = os.environ.get("STATION_LON", "29.14824")
STATION_ALT = float(os.environ.get("STATION_ALT", "50"))

recording_active = False
current_recording = None

# Default ISS TLE (updated periodically)
DEFAULT_TLE = [
    "ISS (ZARYA)",
    "1 25544U 98067A   24001.00000000  .00016717  00000-0  10270-3 0  9999",
    "2 25544  51.6400 100.0000 0007000 100.0000 260.0000 15.49000000000000",
]


def update_tle():
    """Download latest ISS TLE from Celestrak."""
    try:
        import urllib.request

        url = "https://celestrak.org/NORAD/elements/gp.php?CATNR=25544&FORMAT=TLE"
        req = urllib.request.Request(url, headers={"User-Agent": "SDR-ISS-Tracker/1.0"})
        response = urllib.request.urlopen(req, timeout=10)
        data = response.read().decode().strip().split("\n")
        if len(data) >= 3:
            with open(TLE_FILE, "w") as f:
                f.write("\n".join(data[:3]))
            return data[:3]
    except Exception as e:
        print(f"TLE update failed: {e}")
    return None


def get_tle():
    """Get current TLE data."""
    if os.path.exists(TLE_FILE):
        try:
            with open(TLE_FILE) as f:
                lines = f.read().strip().split("\n")
            if len(lines) >= 3:
                return lines[:3]
        except Exception:
            pass
    return DEFAULT_TLE


def get_observer():
    """Create an ephem observer for our station."""
    if not ephem:
        return None
    obs = ephem.Observer()
    obs.lat = STATION_LAT
    obs.lon = STATION_LON
    obs.elevation = STATION_ALT
    obs.horizon = "10"  # Min elevation degrees
    return obs


def predict_passes(count=10):
    """Predict upcoming ISS passes."""
    if not ephem:
        return []

    obs = get_observer()
    if not obs:
        return []

    tle = get_tle()
    try:
        iss = ephem.readtle(tle[0], tle[1], tle[2])
    except Exception:
        return []

    passes = []
    obs.date = ephem.now()

    for _ in range(count * 3):  # Try more to get enough visible passes
        try:
            info = obs.next_pass(iss)
            if info[0] is None:
                break

            rise_time = ephem.Date(info[0]).datetime()
            max_alt = math.degrees(info[3])
            set_time = ephem.Date(info[4]).datetime()
            duration = (set_time - rise_time).total_seconds()

            passes.append(
                {
                    "rise": rise_time.isoformat() + "Z",
                    "set": set_time.isoformat() + "Z",
                    "max_alt": round(max_alt, 1),
                    "duration": round(duration),
                }
            )

            obs.date = info[4] + ephem.minute
        except Exception:
            obs.date += ephem.hour
            continue

        if len(passes) >= count:
            break

    return passes


def get_iss_position():
    """Get current ISS lat/lon."""
    if not ephem:
        return None

    tle = get_tle()
    try:
        iss = ephem.readtle(tle[0], tle[1], tle[2])
        iss.compute()
        return {
            "lat": round(math.degrees(iss.sublat), 4),
            "lon": round(math.degrees(iss.sublong), 4),
            "alt": round(iss.elevation / 1000, 1),
        }
    except Exception:
        return None


def record_pass(duration):
    """Start recording an ISS pass."""
    global recording_active, current_recording

    if recording_active:
        return False

    recording_active = True
    filename = f"iss_{datetime.now().strftime('%Y%m%d_%H%M%S')}.wav"
    filepath = os.path.join(CAPTURES_DIR, filename)
    current_recording = filename

    def do_record():
        global recording_active, current_recording
        try:
            subprocess.run(
                ["/opt/iss/start_record.sh", str(duration), filepath],
                timeout=duration + 30,
            )
        except Exception as e:
            print(f"Recording error: {e}")
        finally:
            recording_active = False
            current_recording = None
            socketio.emit("recording_done", {"filename": filename})

    t = threading.Thread(target=do_record, daemon=True)
    t.start()
    return True


def get_captures():
    """List captured recordings."""
    captures = []
    cap_dir = Path(CAPTURES_DIR)
    for f in sorted(cap_dir.glob("*.wav"), reverse=True):
        stat = f.stat()
        captures.append(
            {
                "filename": f.name,
                "size": stat.st_size,
                "date": datetime.fromtimestamp(stat.st_mtime).isoformat(),
            }
        )
    return captures[:50]


@app.route("/")
def index():
    return send_from_directory("/var/www/html", "index.html")


@app.route("/api/passes")
def api_passes():
    return jsonify(
        {
            "passes": predict_passes(10),
            "station": {"lat": float(STATION_LAT), "lon": float(STATION_LON)},
        }
    )


@app.route("/api/position")
def api_position():
    pos = get_iss_position()
    return jsonify({"position": pos})


@app.route("/api/record", methods=["POST"])
def api_record():
    from flask import request

    data = request.get_json() or {}
    duration = min(int(data.get("duration", 300)), 900)  # Max 15 min
    ok = record_pass(duration)
    return jsonify({"recording": ok, "duration": duration})


@app.route("/api/status")
def api_status():
    return jsonify(
        {
            "recording": recording_active,
            "current_file": current_recording,
        }
    )


@app.route("/api/captures")
def api_captures():
    return jsonify({"captures": get_captures()})


@app.route("/api/captures/<filename>")
def api_capture_file(filename):
    filepath = os.path.join(CAPTURES_DIR, filename)
    if os.path.exists(filepath) and ".." not in filename:
        return send_file(filepath, as_attachment=True)
    return jsonify({"error": "not found"}), 404


@app.route("/api/tle/update", methods=["POST"])
def api_update_tle():
    tle = update_tle()
    if tle:
        return jsonify({"status": "ok", "tle": tle})
    return jsonify({"status": "error"}), 500


# Update TLE on startup
threading.Thread(target=update_tle, daemon=True).start()

if __name__ == "__main__":
    socketio.run(app, host="0.0.0.0", port=5000)
