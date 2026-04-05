import os
import signal
from flask import Flask, jsonify, request

app = Flask(__name__)

PRESETS = [
    {"name": "TRT FM", "freq": "93.3M"},
    {"name": "Radyo Eksen", "freq": "96.2M"},
    {"name": "Joy Turk", "freq": "97.4M"},
    {"name": "Kral FM", "freq": "92.0M"},
    {"name": "Power FM", "freq": "100.0M"},
    {"name": "NTV Radyo", "freq": "102.8M"},
    {"name": "Super FM", "freq": "90.8M"},
    {"name": "Radyo D", "freq": "98.8M"},
    {"name": "Slow Turk", "freq": "95.3M"},
    {"name": "TRT Radyo 1", "freq": "95.6M"},
]


def get_current_freq():
    if os.path.exists("/tmp/current_freq"):
        with open("/tmp/current_freq") as f:
            return f.read().strip()
    return os.environ.get("FM_FREQ", "93.0M")


@app.route("/api/freq", methods=["GET"])
def get_freq():
    return jsonify({"freq": get_current_freq(), "presets": PRESETS})


@app.route("/api/freq", methods=["POST"])
def set_freq():
    data = request.get_json()
    freq = data.get("freq", "").strip()
    if not freq:
        return jsonify({"error": "Frekans belirtilmedi"}), 400

    if not freq.endswith("M"):
        freq += "M"

    with open("/tmp/current_freq", "w") as f:
        f.write(freq)

    # Kill only the current rtl_fm by PID — loop restarts it with new freq
    # ffmpeg and icecast stream stay alive, no browser disconnect
    try:
        with open("/tmp/rtl_fm.pid") as f:
            pid = int(f.read().strip())
        os.kill(pid, signal.SIGTERM)
    except Exception:
        pass

    return jsonify({"freq": freq, "status": "ok"})


if __name__ == "__main__":
    app.run(host="127.0.0.1", port=5000)
