#!/usr/bin/env python3
"""Weather Satellite Image Decoder - NOAA APT receiver and decoder"""

from flask import Flask, Response, jsonify, render_template_string, send_from_directory, request as flask_request
import subprocess
import threading
import time
import json
import os
import glob as globmod
from collections import deque

app = Flask(__name__)
status_log = deque(maxlen=200)
lock = threading.Lock()
proc = None

# NOAA satellite frequencies
NOAA_SATS = {
    'NOAA-15': '137.620M',
    'NOAA-18': '137.9125M',
    'NOAA-19': '137.100M',
}

HTML = """<!DOCTYPE html>
<html><head>
<meta charset="UTF-8">
<title>NOAA Uydu Goruntu Alici</title>
<style>
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: -apple-system, sans-serif; background: #0a0e17; color: #e2e8f0; }
.header { padding: 20px; text-align: center; border-bottom: 1px solid #1e293b; }
.header h1 { color: #06b6d4; font-size: 1.4em; }
.controls { display: flex; gap: 10px; justify-content: center; padding: 20px; flex-wrap: wrap; }
.controls select, .controls input { background: #151822; border: 1px solid #252a3a; color: #e2e8f0;
  padding: 8px 14px; border-radius: 6px; }
.controls button { padding: 8px 20px; border: none; border-radius: 6px; font-weight: bold;
  cursor: pointer; }
.btn-rec { background: #ef4444; color: #fff; }
.btn-stop { background: #64748b; color: #fff; }
.btn-decode { background: #06b6d4; color: #000; }
.content { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; padding: 20px; }
@media (max-width: 768px) { .content { grid-template-columns: 1fr; } }
.panel { background: #151822; border: 1px solid #252a3a; border-radius: 10px; padding: 16px; }
.panel h3 { color: #06b6d4; margin-bottom: 10px; font-size: 0.95em; }
.log { font-family: monospace; font-size: 0.8em; color: #94a3b8; max-height: 300px; overflow-y: auto; }
.log div { padding: 2px 0; border-bottom: 1px solid #1a1f2e; }
.images { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
.images img { width: 100%; border-radius: 6px; cursor: pointer; border: 1px solid #252a3a; }
.status { text-align: center; padding: 10px; }
.status .badge { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 0.8em; }
.badge.idle { background: #1e293b; color: #64748b; }
.badge.recording { background: rgba(239,68,68,0.2); color: #ef4444; }
</style>
</head><body>
<div class="header">
  <h1>🛰️ NOAA Uydu Goruntu Alici</h1>
  <p style="color:#64748b;font-size:0.85em;margin-top:4px">APT sinyal kaydi ve goruntu cozumleme</p>
</div>
<div class="status">
  <span class="badge idle" id="statusBadge">Hazir</span>
</div>
<div class="controls">
  <select id="sat">
    <option value="NOAA-15">NOAA-15 (137.620 MHz)</option>
    <option value="NOAA-18">NOAA-18 (137.9125 MHz)</option>
    <option value="NOAA-19">NOAA-19 (137.100 MHz)</option>
  </select>
  <input type="number" id="duration" value="900" min="60" max="1200" style="width:100px">
  <span style="color:#64748b;line-height:36px">saniye</span>
  <button class="btn-rec" onclick="startRec()">Kayit Baslat</button>
  <button class="btn-stop" onclick="stopRec()">Durdur</button>
</div>
<div class="content">
  <div class="panel">
    <h3>Kayit Loglari</h3>
    <div class="log" id="log"></div>
  </div>
  <div class="panel">
    <h3>Alinan Goruntuler</h3>
    <div class="images" id="images"></div>
  </div>
</div>
<script>
function startRec() {
  const sat = document.getElementById('sat').value;
  const dur = document.getElementById('duration').value;
  fetch('/api/record?sat='+sat+'&duration='+dur, {method:'POST'})
    .then(r=>r.json()).then(d => {
      if(d.ok) {
        document.getElementById('statusBadge').className = 'badge recording';
        document.getElementById('statusBadge').textContent = 'Kayit yapiliyor...';
      }
    });
}
function stopRec() {
  fetch('/api/stop', {method:'POST'});
  document.getElementById('statusBadge').className = 'badge idle';
  document.getElementById('statusBadge').textContent = 'Hazir';
}
function refreshImages() {
  fetch('/api/images').then(r=>r.json()).then(imgs => {
    document.getElementById('images').innerHTML = imgs.map(i =>
      '<a href="/images/'+i+'" target="_blank"><img src="/images/'+i+'"></a>'
    ).join('');
  });
}
function refreshLog() {
  fetch('/api/log').then(r=>r.json()).then(logs => {
    document.getElementById('log').innerHTML = logs.map(l => '<div>'+l+'</div>').join('');
  });
}
setInterval(refreshImages, 10000);
setInterval(refreshLog, 3000);
refreshImages();
refreshLog();
</script>
</body></html>"""

@app.route('/')
def index():
    return render_template_string(HTML)

@app.route('/api/record', methods=['POST'])
def record():
    global proc
    sat = flask_request.args.get('sat', 'NOAA-19')
    duration = flask_request.args.get('duration', '900')
    freq = NOAA_SATS.get(sat, '137.100M')
    fname = f"/opt/recordings/{sat}_{time.strftime('%Y%m%d_%H%M%S')}.wav"

    if proc:
        proc.kill()

    cmd = f"timeout {duration} rtl_fm -f {freq} -s 48000 -g 42 -p 0 -E deemp -F 9 - | sox -t raw -r 48000 -es -b16 -c1 - {fname} rate 11025"
    with lock:
        status_log.append(f"[{time.strftime('%H:%M:%S')}] Kayit baslatildi: {sat} @ {freq}")
    proc = subprocess.Popen(cmd, shell=True, stderr=subprocess.PIPE, text=True)

    def monitor():
        global proc
        proc.wait()
        with lock:
            status_log.append(f"[{time.strftime('%H:%M:%S')}] Kayit tamamlandi: {fname}")
        # Auto-decode with sox (basic spectral image)
        try:
            img_name = fname.replace('.wav', '.png')
            subprocess.run(['sox', fname, '-n', 'spectrogram', '-o', img_name, '-x', '1024', '-y', '512'], timeout=60)
            with lock:
                status_log.append(f"[{time.strftime('%H:%M:%S')}] Spektrogram olusturuldu")
        except Exception as e:
            with lock:
                status_log.append(f"[{time.strftime('%H:%M:%S')}] Decode hatasi: {e}")

    threading.Thread(target=monitor, daemon=True).start()
    return jsonify({'ok': True})

@app.route('/api/stop', methods=['POST'])
def stop():
    global proc
    if proc:
        proc.kill()
        proc = None
    return jsonify({'ok': True})

@app.route('/api/log')
def api_log():
    with lock:
        return jsonify(list(status_log))

@app.route('/api/images')
def api_images():
    images = sorted(globmod.glob('/opt/images/*.png') + globmod.glob('/opt/recordings/*.png'), reverse=True)
    return jsonify([os.path.basename(f) for f in images[:20]])

@app.route('/images/<path:filename>')
def serve_image(filename):
    for d in ['/opt/images', '/opt/recordings']:
        if os.path.exists(os.path.join(d, filename)):
            return send_from_directory(d, filename)
    return 'Not found', 404

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8090)
