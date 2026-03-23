from flask import Flask, render_template, redirect, url_for, flash
import subprocess

app = Flask(__name__)
app.secret_key = "sdr-switcher-key"

PROJECTS = {
    "openwebrx": {
        "name": "OpenWebRX",
        "path": "/home/kaandikec/sdr/openwebrx/docker-compose.yml",
        "description": "SDR Radyo Alicisi",
        "port": 8090,
    },
    "ultrafeeder": {
        "name": "Ultrafeeder ADS-B",
        "path": "/home/kaandikec/sdr/ultrafeeder/docker-compose.yml",
        "description": "Ucak Takip (Ultrafeeder + PiAware + FR24)",
        "port": 8090,
    },
    "phantomsdr": {
        "name": "PhantomSDR",
        "path": "/home/kaandikec/sdr/phantomsdr/docker-compose.yml",
        "description": "Web Tabanli SDR Waterfall",
        "port": 8090,
    },
    "shinysdr": {
        "name": "ShinySDR",
        "path": "/home/kaandikec/sdr/shinysdr/docker-compose.yml",
        "description": "Gelismis SDR Alici Arayuzu",
        "port": 8090,
    },
    "sdrpp": {
        "name": "SDR++ (noVNC)",
        "path": "/home/kaandikec/sdr/sdrpp-server/docker-compose.yml",
        "description": "SDR++ Masaustu - Tarayicidan Erisim",
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
    cmd = ["docker", "compose", "-f", proj["path"]] + action.split()
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
