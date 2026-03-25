from flask import Flask, render_template, jsonify, request
import subprocess
import os
import time

BASE_DIR = os.environ.get("SDR_BASE_DIR", "/opt/sdr")
ENV_FILE = os.path.join(BASE_DIR, ".env")

app = Flask(__name__)
app.secret_key = os.environ.get("FLASK_SECRET_KEY", "change-me")

CATEGORIES = {
    "receivers": {
        "name": "Genel SDR Alicilar",
        "icon": "📡",
        "description": "FM, AM, SSB ve diger modlari dinleme",
    },
    "digital": {
        "name": "Dijital Sinyal Cozme",
        "icon": "📻",
        "description": "Blok tabanli sinyal isleme ve analiz",
    },
    "decoders": {
        "name": "Dijital Mod Cozuculer",
        "icon": "📶",
        "description": "Pager, DTMF, DMR, P25, APRS cozumleme",
    },
    "tracking": {
        "name": "Ucak & Arac Takip",
        "icon": "✈️",
        "description": "ADS-B ucak takip ve radar",
    },
    "satellite": {
        "name": "Uydu & Hava Durumu",
        "icon": "🛰️",
        "description": "NOAA uydu goruntu alma",
    },
    "security": {
        "name": "Guvenlik & Arastirma",
        "icon": "🛡️",
        "description": "RF analiz ve protokol cozumleme",
    },
}

PROJECTS = {
    # --- Genel SDR Alicilar ---
    "openwebrx": {
        "name": "OpenWebRX",
        "path": f"{BASE_DIR}/openwebrx/docker-compose.yml",
        "description": "Web tabanli SDR alici - 968 profil ile tam spektrum tarama",
        "icon": "radio",
        "port": 8090,
        "category": "receivers",
        "badge": "Web UI",
    },
    "sdrpp": {
        "name": "SDR++",
        "path": f"{BASE_DIR}/sdrpp-server/docker-compose.yml",
        "description": "Profesyonel masaustu SDR yazilimi - noVNC ile uzaktan erisim + ses",
        "icon": "desktop",
        "port": 8090,
        "category": "receivers",
        "badge": "noVNC",
    },
    "phantomsdr": {
        "name": "PhantomSDR",
        "path": f"{BASE_DIR}/phantomsdr/docker-compose.yml",
        "description": "Yuksek performansli waterfall gosterimi - coklu kullanici destegi",
        "icon": "waterfall",
        "port": 8090,
        "category": "receivers",
        "badge": "Web UI",
    },
    "shinysdr": {
        "name": "ShinySDR",
        "path": f"{BASE_DIR}/shinysdr/docker-compose.yml",
        "description": "Gelismis sinyal analizi ve demodulasyon araclari",
        "icon": "signal",
        "port": 8090,
        "category": "receivers",
        "badge": "Web UI",
    },
    "gqrx": {
        "name": "GQRX",
        "path": f"{BASE_DIR}/gqrx/docker-compose.yml",
        "description": "Linux'un en populer SDR alicisi - spektrum + waterfall + demodulasyon",
        "icon": "gqrx",
        "port": 8090,
        "category": "receivers",
        "badge": "noVNC",
    },
    "cubicsdr": {
        "name": "CubicSDR",
        "path": f"{BASE_DIR}/cubicsdr/docker-compose.yml",
        "description": "Modern ve kullanici dostu SDR alici - SoapySDR destegi",
        "icon": "cubic",
        "port": 8090,
        "category": "receivers",
        "badge": "noVNC",
    },
    "sdrangel": {
        "name": "SDRAngel",
        "path": f"{BASE_DIR}/sdrangel/docker-compose.yml",
        "description": "Her sey dahil SDR paketi - TX/RX, analiz, demod, spektrum",
        "icon": "angel",
        "port": 8090,
        "category": "receivers",
        "badge": "noVNC",
    },
    # --- Dijital Sinyal Cozme ---
    "gnuradio": {
        "name": "GNU Radio",
        "path": f"{BASE_DIR}/gnuradio/docker-compose.yml",
        "description": "Blok tabanli sinyal isleme - kendi alici/vericini tasarla",
        "icon": "gnuradio",
        "port": 8090,
        "category": "digital",
        "badge": "noVNC",
    },
    "inspectrum": {
        "name": "Inspectrum",
        "path": f"{BASE_DIR}/inspectrum/docker-compose.yml",
        "description": "Kayitli sinyalleri detayli analiz - bit seviyesinde inceleme",
        "icon": "inspect",
        "port": 8090,
        "category": "digital",
        "badge": "noVNC",
    },
    # --- Dijital Mod Cozuculer ---
    "multimon_ng": {
        "name": "multimon-ng",
        "path": f"{BASE_DIR}/multimon-ng/docker-compose.yml",
        "description": "Pager, APRS, DTMF, FSK, Morse, POCSAG, FLEX cozuculeme",
        "icon": "decoder",
        "port": 8090,
        "category": "decoders",
        "badge": "Web UI",
    },
    "dsd": {
        "name": "DSD",
        "path": f"{BASE_DIR}/dsd/docker-compose.yml",
        "description": "DMR, P25, NXDN, D-STAR dijital ses cozucu",
        "icon": "voice",
        "port": 8090,
        "category": "decoders",
        "badge": "Web UI",
    },
    "rtl433": {
        "name": "rtl_433",
        "path": f"{BASE_DIR}/rtl433/docker-compose.yml",
        "description": "433 MHz ISM cihazlari izleme - termometre, sensor, anahtar",
        "icon": "sensor",
        "port": 8090,
        "category": "decoders",
        "badge": "Web UI",
    },
    "sdrtrunk": {
        "name": "SDRTrunk",
        "path": f"{BASE_DIR}/sdrtrunk/docker-compose.yml",
        "description": "P25, DMR dijital telsiz dinleme - cok kanalli takip",
        "icon": "trunk",
        "port": 8090,
        "category": "decoders",
        "badge": "noVNC",
    },
    "op25": {
        "name": "OP25",
        "path": f"{BASE_DIR}/op25/docker-compose.yml",
        "description": "Profesyonel P25 telsiz sistemi alicisi - web arayuzu",
        "icon": "op25",
        "port": 8090,
        "category": "decoders",
        "badge": "Web UI",
    },
    # --- Ucak Takip ---
    "ultrafeeder": {
        "name": "Ultrafeeder ADS-B",
        "path": f"{BASE_DIR}/ultrafeeder/docker-compose.yml",
        "description": "Ucak takip - FlightAware, Flightradar24 ve MLAT destegi",
        "icon": "plane",
        "port": 8090,
        "category": "tracking",
        "badge": "Web UI",
    },
    # --- Uydu ---
    "wximgtools": {
        "name": "NOAA Uydu Alici",
        "path": f"{BASE_DIR}/wximgtools/docker-compose.yml",
        "description": "NOAA hava uydularindan APT sinyal kaydi ve goruntu cozumleme",
        "icon": "satellite",
        "port": 8090,
        "category": "satellite",
        "badge": "Web UI",
    },
    # --- Guvenlik ---
    "urh": {
        "name": "Universal Radio Hacker",
        "path": f"{BASE_DIR}/urh/docker-compose.yml",
        "description": "Kablosuz cihaz analizi - RF capture, replay, protokol cozumleme",
        "icon": "hacker",
        "port": 8090,
        "category": "security",
        "badge": "noVNC",
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
            "category": proj["category"],
            "badge": proj.get("badge", ""),
        })

    categories = []
    for cat_id, cat_info in CATEGORIES.items():
        cat_projects = [p for p in projects if p["category"] == cat_id]
        if cat_projects:
            categories.append({
                "id": cat_id,
                "name": cat_info["name"],
                "icon": cat_info["icon"],
                "description": cat_info["description"],
                "projects": cat_projects,
            })

    return jsonify({"categories": categories, "active": active, "total": len(PROJECTS)})


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
