#!/usr/bin/env python3
"""DSD Web Dashboard - Digital voice decoder (DMR, P25, NXDN, D-STAR)"""

from flask import Flask, Response, jsonify, render_template_string, request as flask_request
import subprocess
import threading
import time
import json
from collections import deque

app = Flask(__name__)
messages = deque(maxlen=500)
lock = threading.Lock()
proc = None
proc_lock = threading.Lock()

HTML = """<!DOCTYPE html>
<html><head>
<meta charset="UTF-8">
<title>DSD - Digital Speech Decoder</title>
<style>
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: 'Courier New', monospace; background: #0a0e17; color: #e2e8f0; }
.header { padding: 20px; text-align: center; border-bottom: 1px solid #1e293b; }
.header h1 { color: #a855f7; font-size: 1.4em; }
.header p { color: #64748b; font-size: 0.85em; margin-top: 4px; }
.controls { display: flex; gap: 10px; justify-content: center; padding: 15px; flex-wrap: wrap; }
.controls select, .controls input { background: #151822; border: 1px solid #252a3a; color: #e2e8f0;
  padding: 8px 14px; border-radius: 6px; font-family: inherit; }
.controls button { padding: 8px 20px; border: none; border-radius: 6px; font-weight: bold;
  cursor: pointer; font-family: inherit; }
.btn-start { background: #22c55e; color: #000; }
.btn-stop { background: #ef4444; color: #fff; }
.stats { display: flex; gap: 15px; justify-content: center; padding: 10px; }
.stat { background: #151822; padding: 8px 16px; border-radius: 8px; font-size: 0.8em;
  border: 1px solid #252a3a; }
.stat b { color: #a855f7; }
#log { padding: 10px 20px; font-size: 0.85em; }
.msg { background: #151822; margin: 4px 0; padding: 10px 14px; border-radius: 6px;
  border-left: 3px solid #a855f7; }
.msg .proto { color: #22c55e; font-weight: bold; }
.msg .time { color: #64748b; }
.empty { text-align: center; color: #475569; padding: 60px; }
</style>
</head><body>
<div class="header">
  <h1>🔊 DSD - Digital Speech Decoder</h1>
  <p>DMR, P25, NXDN, D-STAR, ProVoice dijital ses cozucu</p>
</div>
<div class="controls">
  <input type="number" id="freq" value="460.0" step="0.0125" min="24" max="1766" placeholder="Frekans (MHz)">
  <span style="color:#64748b;line-height:36px">MHz</span>
  <select id="mode">
    <option value="-fp">Otomatik (ProVoice dahil)</option>
    <option value="-f1">P25 Phase 1</option>
    <option value="-fd">DMR/MOTOTRBO</option>
    <option value="-fi">NXDN48 (IDAS)</option>
    <option value="-fn">NXDN96</option>
    <option value="-fr">D-STAR</option>
  </select>
  <button class="btn-start" onclick="startDecode()">Baslat</button>
  <button class="btn-stop" onclick="stopDecode()">Durdur</button>
</div>
<div class="stats">
  <div class="stat">Frames: <b id="frames">0</b></div>
  <div class="stat">Durum: <b id="status">Bekliyor</b></div>
</div>
<div id="log"><div class="empty">Dijital ses cozuculemesi baslatilmadi.</div></div>
<script>
let evtSource = null, frames = 0;
function startDecode() {
  const freq = document.getElementById('freq').value;
  const mode = document.getElementById('mode').value;
  fetch('/api/start?freq='+freq+'M&mode='+encodeURIComponent(mode), {method:'POST'})
    .then(r=>r.json()).then(d => {
      if(d.ok) {
        document.getElementById('status').textContent = 'Dinleniyor...';
        document.getElementById('log').innerHTML = '';
        if(evtSource) evtSource.close();
        evtSource = new EventSource('/stream');
        evtSource.onmessage = e => {
          frames++;
          document.getElementById('frames').textContent = frames;
          const div = document.createElement('div');
          div.className = 'msg';
          const data = JSON.parse(e.data);
          div.innerHTML = '<span class="time">'+data.time+'</span> <span class="proto">['+data.type+']</span> '+data.message;
          document.getElementById('log').prepend(div);
        };
      }
    });
}
function stopDecode() {
  if(evtSource) { evtSource.close(); evtSource = null; }
  fetch('/api/stop', {method:'POST'});
  document.getElementById('status').textContent = 'Durduruldu';
}
</script>
</body></html>"""

def reader_thread(process):
    try:
        for line in process.stderr:
            line = line.strip()
            if line:
                with lock:
                    messages.append({
                        'time': time.strftime('%H:%M:%S'),
                        'type': 'DSD',
                        'message': line
                    })
    except Exception:
        pass

@app.route('/')
def index():
    return render_template_string(HTML)

@app.route('/api/start', methods=['POST'])
def start():
    global proc
    freq = flask_request.args.get('freq', '460.0M')
    mode = flask_request.args.get('mode', '-fp')
    with proc_lock:
        if proc:
            proc.kill()
        cmd = f"rtl_fm -f {freq} -s 48000 -g 42 | dsd {mode} -i - -o /dev/null"
        proc = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        t = threading.Thread(target=reader_thread, args=(proc,), daemon=True)
        t.start()
    return jsonify({'ok': True})

@app.route('/api/stop', methods=['POST'])
def stop():
    global proc
    with proc_lock:
        if proc:
            proc.kill()
            proc = None
    return jsonify({'ok': True})

@app.route('/stream')
def stream():
    def generate():
        last_len = 0
        while True:
            with lock:
                cl = len(messages)
                if cl > last_len:
                    for item in list(messages)[last_len:]:
                        yield f"data: {json.dumps(item)}\n\n"
                    last_len = cl
            time.sleep(0.3)
    return Response(generate(), mimetype='text/event-stream')

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8090)
