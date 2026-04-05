from flask import Flask, render_template, redirect, url_for, flash
import subprocess
import os

BASE_DIR = os.environ.get("SDR_BASE_DIR", "/opt/sdr")

app = Flask(__name__)
app.secret_key = os.environ.get("FLASK_SECRET_KEY", "change-me")

PROJECTS = {
    "openwebrx": {
        "name": "OpenWebRX",
        "path": f"{BASE_DIR}/openwebrx/docker-compose.yml",
        "description": "SDR Radyo Alicisi",
        "port": 8090,
    },
    "ultrafeeder": {
        "name": "Ultrafeeder ADS-B",
        "path": f"{BASE_DIR}/ultrafeeder/docker-compose.yml",
        "description": "Ucak Takip (Ultrafeeder + PiAware + FR24)",
        "port": 8090,
    },
    "phantomsdr": {
        "name": "PhantomSDR",
        "path": f"{BASE_DIR}/phantomsdr/docker-compose.yml",
        "description": "Web Tabanli SDR Waterfall",
        "port": 8090,
    },
    "shinysdr": {
        "name": "ShinySDR",
        "path": f"{BASE_DIR}/shinysdr/docker-compose.yml",
        "description": "Gelismis SDR Alici Arayuzu",
        "port": 8090,
    },
    "sdrpp": {
        "name": "SDR++ (noVNC)",
        "path": f"{BASE_DIR}/sdrpp-server/docker-compose.yml",
        "description": "SDR++ Masaustu - Tarayicidan Erisim",
        "port": 8090,
    },
    "satdump": {
        "name": "SatDump",
        "path": f"{BASE_DIR}/satdump/docker-compose.yml",
        "description": "NOAA/Meteor Uydu Goruntu Yakalama",
        "port": 8090,
    },
    "airband": {
        "name": "Airband Dinleme",
        "path": f"{BASE_DIR}/airband/docker-compose.yml",
        "description": "Havacilik Frekans Dinleme (118-137 MHz)",
        "port": 8090,
    },
    "fmradio": {
        "name": "FM Radyo",
        "path": f"{BASE_DIR}/fmradio/docker-compose.yml",
        "description": "FM Radyo Dinleme",
        "port": 8090,
    },
    "hamradio": {
        "name": "Amator Radyo",
        "path": f"{BASE_DIR}/hamradio/docker-compose.yml",
        "description": "Amator Radyo Dinleme (VHF/UHF)",
        "port": 8090,
    },
    "pager": {
        "name": "Pager/POCSAG",
        "path": f"{BASE_DIR}/pager/docker-compose.yml",
        "description": "POCSAG/FLEX Cagri Cihazi Cozucu",
        "port": 8090,
    },
    "ais": {
        "name": "AIS Gemi Takip",
        "path": f"{BASE_DIR}/ais/docker-compose.yml",
        "description": "Deniz Trafigi Izleme (AIS)",
        "port": 8090,
    },
    "iss": {
        "name": "ISS/Meteor",
        "path": f"{BASE_DIR}/iss/docker-compose.yml",
        "description": "ISS SSTV ve Meteor Scatter Izleme",
        "port": 8090,
    },
    "gsm": {
        "name": "GSM Tarayici",
        "path": f"{BASE_DIR}/gsm/docker-compose.yml",
        "description": "GSM Baz Istasyonu Tarama",
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


ENV_FILE = os.path.join(BASE_DIR, ".env")


def compose_action(project, action):
    proj = PROJECTS[project]
    cmd = ["docker", "compose", "--env-file", ENV_FILE, "-f", proj["path"]] + action.split()
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    return result.returncode == 0, result.stderr


@app.route("/")
def index():
    status = get_status()
    return render_template("index.html", projects=PROJECTS, status=status)


@app.route("/start/<project>", methods=["POST"])
def start(project):
    if project not in PROJECTS:
        flash("Gecersiz proje", "error")
        return redirect(url_for("index"))

    # Diger projeyi durdur
    for key in PROJECTS:
        if key != project:
            status = get_status()
            if status.get(key):
                ok, err = compose_action(key, "down")
                if not ok:
                    flash(f"{PROJECTS[key]['name']} durdurulamadi: {err}", "error")
                    return redirect(url_for("index"))

    # Secilen projeyi baslat
    ok, err = compose_action(project, "up -d")
    if ok:
        flash(f"{PROJECTS[project]['name']} baslatildi", "success")
    else:
        flash(f"Hata: {err}", "error")

    return redirect(url_for("index"))


@app.route("/stop/<project>", methods=["POST"])
def stop(project):
    if project not in PROJECTS:
        flash("Gecersiz proje", "error")
        return redirect(url_for("index"))

    ok, err = compose_action(project, "down")
    if ok:
        flash(f"{PROJECTS[project]['name']} durduruldu", "success")
    else:
        flash(f"Hata: {err}", "error")

    return redirect(url_for("index"))


@app.route("/restart/<project>", methods=["POST"])
def restart(project):
    if project not in PROJECTS:
        flash("Gecersiz proje", "error")
        return redirect(url_for("index"))

    ok, err = compose_action(project, "restart")
    if ok:
        flash(f"{PROJECTS[project]['name']} yeniden baslatildi", "success")
    else:
        flash(f"Hata: {err}", "error")

    return redirect(url_for("index"))


@app.route("/stop-all", methods=["POST"])
def stop_all():
    errors = []
    for key in PROJECTS:
        ok, err = compose_action(key, "down")
        if not ok:
            errors.append(f"{PROJECTS[key]['name']}: {err}")

    if errors:
        flash("Hata: " + "; ".join(errors), "error")
    else:
        flash("Tum projeler durduruldu", "success")

    return redirect(url_for("index"))


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8091)
