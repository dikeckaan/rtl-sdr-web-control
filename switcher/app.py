from flask import Flask, render_template, jsonify, request
import subprocess
import os
import time

BASE_DIR = os.environ.get("SDR_BASE_DIR", "/opt/sdr")
ENV_FILE = os.path.join(BASE_DIR, ".env")

app = Flask(__name__)
app.secret_key = os.environ.get("FLASK_SECRET_KEY", "change-me")

PROJECTS = {
    "openwebrx": {
        "name": "OpenWebRX",
        "path": f"{BASE_DIR}/openwebrx/docker-compose.yml",
        "description": "Web tabanli SDR alici - 968 profil ile tam spektrum tarama",
        "icon": "radio",
        "port": 8090,
    },
    "sdrpp": {
        "name": "SDR++",
        "path": f"{BASE_DIR}/sdrpp-server/docker-compose.yml",
        "description": "Profesyonel masaustu SDR yazilimi - noVNC ile uzaktan erisim",
        "icon": "desktop",
        "port": 8090,
    },
    "phantomsdr": {
        "name": "PhantomSDR",
        "path": f"{BASE_DIR}/phantomsdr/docker-compose.yml",
        "description": "Yuksek performansli waterfall gosterimi - coklu kullanici",
        "icon": "waterfall",
        "port": 8090,
    },
    "shinysdr": {
        "name": "ShinySDR",
        "path": f"{BASE_DIR}/shinysdr/docker-compose.yml",
        "description": "Gelismis sinyal analizi ve demodulasyon araclari",
        "icon": "signal",
        "port": 8090,
    },
    "ultrafeeder": {
        "name": "Ultrafeeder ADS-B",
        "path": f"{BASE_DIR}/ultrafeeder/docker-compose.yml",
        "description": "Ucak takip - FlightAware, Flightradar24 ve MLAT destegi",
        "icon": "plane",
        "port": 8090,
    },
}


def get_status():
    status = {}
    for key, proj in PROJECTS.items():
        try:
            result = subprocess.run(
                ["docker", "compose", "-f", proj["path"], "ps", "--format", "{{.Names}}"],
                capture_output=True, text=True, timeout=10,
            )
            names = result.stdout.strip()
            status[key] = bool(names)
        except Exception:
            status[key] = False
    return status


def compose_action(project, action):
    proj = PROJECTS[project]
    cmd = ["docker", "compose", "--env-file", ENV_FILE, "-f", proj["path"]] + action.split()
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    return result.returncode == 0, result.stderr


@app.route("/")
def index():
    return render_template("index.html")


@app.route("/api/status")
def api_status():
    status = get_status()
    active = None
    projects = []
    for key, proj in PROJECTS.items():
        running = status.get(key, False)
        if running:
            active = key
        projects.append({
            "id": key,
            "name": proj["name"],
            "description": proj["description"],
            "icon": proj["icon"],
            "port": proj["port"],
            "running": running,
        })
    return jsonify({"projects": projects, "active": active})


@app.route("/api/start/<project>", methods=["POST"])
def api_start(project):
    if project not in PROJECTS:
        return jsonify({"ok": False, "error": "Gecersiz proje"}), 400

    # Stop all others first
    status = get_status()
    for key in PROJECTS:
        if key != project and status.get(key):
            ok, err = compose_action(key, "down")
            if not ok:
                return jsonify({"ok": False, "error": f"{PROJECTS[key]['name']} durdurulamadi"}), 500

    ok, err = compose_action(project, "up -d")
    if ok:
        return jsonify({"ok": True, "message": f"{PROJECTS[project]['name']} baslatildi"})
    return jsonify({"ok": False, "error": err}), 500


@app.route("/api/stop/<project>", methods=["POST"])
def api_stop(project):
    if project not in PROJECTS:
        return jsonify({"ok": False, "error": "Gecersiz proje"}), 400

    ok, err = compose_action(project, "down")
    if ok:
        return jsonify({"ok": True, "message": f"{PROJECTS[project]['name']} durduruldu"})
    return jsonify({"ok": False, "error": err}), 500


@app.route("/api/restart/<project>", methods=["POST"])
def api_restart(project):
    if project not in PROJECTS:
        return jsonify({"ok": False, "error": "Gecersiz proje"}), 400

    ok, err = compose_action(project, "down")
    if not ok:
        return jsonify({"ok": False, "error": err}), 500

    time.sleep(1)

    ok, err = compose_action(project, "up -d")
    if ok:
        return jsonify({"ok": True, "message": f"{PROJECTS[project]['name']} yeniden baslatildi"})
    return jsonify({"ok": False, "error": err}), 500


@app.route("/api/stop-all", methods=["POST"])
def api_stop_all():
    errors = []
    for key in PROJECTS:
        ok, err = compose_action(key, "down")
        if not ok:
            errors.append(f"{PROJECTS[key]['name']}: {err}")

    if errors:
        return jsonify({"ok": False, "error": "; ".join(errors)}), 500
    return jsonify({"ok": True, "message": "Tum projeler durduruldu"})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8091)
