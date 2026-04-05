#!/usr/bin/env python3
import os
import subprocess
from flask import Flask, request, jsonify

app = Flask(__name__)

PRESETS = [
    {"freq": "145.500M", "label": "2m Cagri", "band": "2m", "mode": "fm"},
    {"freq": "145.200M", "label": "2m Simplex", "band": "2m", "mode": "fm"},
    {"freq": "145.825M", "label": "ISS APRS", "band": "2m", "mode": "fm"},
    {"freq": "144.800M", "label": "APRS", "band": "2m", "mode": "fm"},
    {"freq": "433.500M", "label": "70cm Cagri", "band": "70cm", "mode": "fm"},
    {"freq": "433.400M", "label": "70cm Simplex", "band": "70cm", "mode": "fm"},
    {"freq": "28.500M", "label": "10m", "band": "10m", "mode": "usb"},
]

MODES = ["fm", "am", "usb", "lsb"]


def get_current_freq():
    try:
        with open("/tmp/current_freq", "r") as f:
            return f.read().strip()
    except FileNotFoundError:
        return os.environ.get("HAM_FREQ", "145.500M")


def get_current_mode():
    try:
        with open("/tmp/current_mode", "r") as f:
            return f.read().strip()
    except FileNotFoundError:
        return os.environ.get("HAM_MODE", "fm")


def restart_rtl_fm():
    try:
        subprocess.run(
            ["supervisorctl", "restart", "hamradio"],
            capture_output=True, timeout=15
        )
    except Exception:
        pass


@app.route("/api/tune", methods=["GET"])
def get_tune():
    return jsonify({
        "freq": get_current_freq(),
        "mode": get_current_mode(),
        "presets": PRESETS,
        "modes": MODES,
    })


@app.route("/api/tune", methods=["POST"])
def set_tune():
    data = request.get_json()
    if not data:
        return jsonify({"error": "No data provided"}), 400

    freq = data.get("freq", get_current_freq())
    mode = data.get("mode", get_current_mode())

    if mode not in MODES:
        return jsonify({"error": f"Invalid mode: {mode}. Must be one of {MODES}"}), 400

    with open("/tmp/current_freq", "w") as f:
        f.write(freq)

    with open("/tmp/current_mode", "w") as f:
        f.write(mode)

    restart_rtl_fm()

    return jsonify({
        "freq": freq,
        "mode": mode,
        "status": "tuning",
    })


if __name__ == "__main__":
    # Initialize files if they don't exist
    if not os.path.exists("/tmp/current_freq"):
        with open("/tmp/current_freq", "w") as f:
            f.write(os.environ.get("HAM_FREQ", "145.500M"))
    if not os.path.exists("/tmp/current_mode"):
        with open("/tmp/current_mode", "w") as f:
            f.write(os.environ.get("HAM_MODE", "fm"))

    app.run(host="0.0.0.0", port=5000)
