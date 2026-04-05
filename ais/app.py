#!/usr/bin/env python3
import socket
import threading
import time
import json
from flask import Flask, jsonify, send_from_directory
from flask_socketio import SocketIO

app = Flask(__name__, static_folder="/var/www/html", static_url_path="")
app.config["SECRET_KEY"] = "ais-tracker-secret"
socketio = SocketIO(app, cors_allowed_origins="*", async_mode="threading")

ships = {}
stats = {"messages": 0, "ships": 0}


def parse_nmea_ais(line):
    """Parse NMEA AIS sentences using pyais."""
    try:
        from pyais import decode

        line = line.strip()
        if not line or not line.startswith("!"):
            return None

        msg = decode(line)
        decoded = msg.asdict()

        result = {
            "mmsi": str(decoded.get("mmsi", "")),
            "type": decoded.get("msg_type", 0),
        }

        if "lat" in decoded and "lon" in decoded:
            lat = decoded["lat"]
            lon = decoded["lon"]
            if lat and lon and abs(lat) <= 90 and abs(lon) <= 180:
                result["lat"] = round(lat, 6)
                result["lon"] = round(lon, 6)

        if "speed" in decoded and decoded["speed"] is not None:
            result["speed"] = round(decoded["speed"], 1)
        if "course" in decoded and decoded["course"] is not None:
            result["course"] = round(decoded["course"], 1)
        if "heading" in decoded and decoded["heading"] is not None:
            result["heading"] = decoded["heading"]
        if "shipname" in decoded and decoded["shipname"]:
            result["name"] = decoded["shipname"].strip()
        if "destination" in decoded and decoded["destination"]:
            result["destination"] = decoded["destination"].strip()
        if "ship_type" in decoded:
            result["ship_type"] = decoded["ship_type"]

        return result
    except Exception:
        return None


def listen_ais():
    """Listen to rtl_ais NMEA output on stdout via a pipe."""
    import subprocess
    import sys

    # rtl_ais outputs NMEA sentences to stdout when run with -n flag
    # We read from a TCP connection or from a file
    # Actually rtl_ais -n outputs to stdout, but supervisor captures it
    # So we'll read from a named pipe or TCP
    # Let's use the fact that rtl_ais can output to a file/pipe

    # Wait for rtl_ais to start
    time.sleep(5)

    while True:
        try:
            # rtl_ais outputs NMEA to stdout, we'll read /tmp/ais_output
            with open("/tmp/ais_fifo", "r") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    parsed = parse_nmea_ais(line)
                    if parsed and parsed.get("mmsi"):
                        mmsi = parsed["mmsi"]
                        parsed["last_seen"] = time.time()
                        stats["messages"] += 1

                        if mmsi not in ships:
                            stats["ships"] += 1

                        if mmsi in ships:
                            ships[mmsi].update(parsed)
                        else:
                            ships[mmsi] = parsed

                        socketio.emit("ship_update", ships[mmsi])
        except FileNotFoundError:
            time.sleep(1)
        except Exception:
            time.sleep(1)


@app.route("/")
def index():
    return send_from_directory("/var/www/html", "index.html")


@app.route("/api/ships")
def api_ships():
    # Remove stale ships (not seen in 10 minutes)
    cutoff = time.time() - 600
    active = {k: v for k, v in ships.items() if v.get("last_seen", 0) > cutoff}
    return jsonify({"ships": list(active.values()), "stats": stats})


@app.route("/api/stats")
def api_stats():
    return jsonify(stats)


@socketio.on("connect")
def handle_connect():
    # Send all current ships
    cutoff = time.time() - 600
    for mmsi, ship in ships.items():
        if ship.get("last_seen", 0) > cutoff:
            socketio.emit("ship_update", ship)


# Start AIS listener thread
ais_thread = threading.Thread(target=listen_ais, daemon=True)
ais_thread.start()

if __name__ == "__main__":
    socketio.run(app, host="0.0.0.0", port=5000)
