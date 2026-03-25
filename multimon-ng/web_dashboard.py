#!/usr/bin/env python3
"""multimon-ng Web Dashboard - Decode pager, DTMF, FSK, FLEX, POCSAG, Morse and more"""

from flask import Flask, Response, jsonify, render_template_string
import subprocess
import threading
import time
from collections import deque

app = Flask(__name__)
messages = deque(maxlen=1000)
lock = threading.Lock()

HTML = """<!DOCTYPE html>
<html><head>
<meta charset="UTF-8">
<title>multimon-ng Decoder</title>
<style>
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: 'Courier New', monospace; background: #0a0e17; color: #e2e8f0; }
.header { padding: 20px; text-align: center; border-bottom: 1px solid #1e293b; }
.header h1 { color: #f59e0b; font-size: 1.4em; }
.header p { color: #64748b; font-size: 0.85em; margin-top: 4px; }
.controls { display: flex; gap: 10px; justify-content: center; padding: 15px; flex-wrap: wrap; }
.controls select, .controls button { background: #151822; border: 1px solid #252a3a; color: #e2e8f0;
  padding: 8px 16px; border-radius: 6px; font-family: inherit; cursor: pointer; }
.controls button { background: #22c55e; color: #000; font-weight: bold; border: none; }
.controls button.stop { background: #ef4444; color: #fff; }
.stats { display: flex; gap: 15px; justify-content: center; padding: 10px; }
.stat { background: #151822; padding: 8px 16px; border-radius: 8px; font-size: 0.8em;
  border: 1px solid #252a3a; }
.stat b { color: #3b82f6; }
#log { padding: 10px 20px; font-size: 0.85em; }
.msg { background: #151822; margin: 4px 0; padding: 10px 14px; border-radius: 6px;
  border-left: 3px solid #f59e0b; animation: fadeIn 0.3s; }
.msg .proto { color: #22c55e; font-weight: bold; }
.msg .time { color: #64748b; font-size: 0.8em; }
.msg .content { color: #e2e8f0; }
@keyframes fadeIn { from { opacity: 0; } }
.empty { text-align: center; color: #475569; padding: 60px; }
</style>
</head><body>
<div class="header">
  <h1>📟 multimon-ng Decoder</h1>
  <p>Pager, DTMF, POCSAG, FLEX, Morse ve daha fazlasi</p>
</div>
<div class="controls">
  <select id="freq">
    <option value="148.8125M">148.8125 MHz (Pager)</option>
    <option value="152.840M">152.840 MHz (Pager)</option>
    <option value="462.5625M">462.5625 MHz (FRS/GMRS)</option>
    <option value="446.0M">446.0 MHz (PMR446)</option>
    <option value="433.92M">433.92 MHz (ISM)</option>
  </select>
  <select id="mode">
    <option value="-a POCSAG512 -a POCSAG1200 -a POCSAG2400">POCSAG (Pager)</option>
    <option value="-a FLEX">FLEX (Pager)</option>
    <option value="-a DTMF">DTMF Tones</option>
    <option value="-a MORSE_CW">Morse CW</option>
    <option value="-a AFSK1200">AFSK 1200 (APRS)</option>
    <option value="-a ALL">Hepsi (ALL)</option>
  </select>
  <button onclick="startDecode()" id="startBtn">Dinlemeyi Baslat</button>
  <button onclick="stopDecode()" class="stop" id="stopBtn" disabled>Durdur</button>
</div>
<div class="stats">
  <div class="stat">Toplam: <b id="total">0</b></div>
  <div class="stat">Durum: <b id="status">Bekliyor</b></div>
</div>
<div id="log"><div class="empty">Dinleme baslatilmadi. Frekans ve mod seçip "Dinlemeyi Baslat" butonuna basin.</div></div>
<script>
let evtSource = null;
let total = 0;

function startDecode() {
  const freq = document.getElementById('freq').value;
  const mode = document.getElementById('mode').value;
  stopDecode();
  fetch('/api/start?freq=' + freq + '&mode=' + encodeURIComponent(mode), {method: 'POST'})
    .then(r => r.json()).then(d => {
      if (d.ok) {
        document.getElementById('status').textContent = 'Dinleniyor...';
        document.getElementById('status').style.color = '#22c55e';
        document.getElementById('startBtn').disabled = true;
        document.getElementById('stopBtn').disabled = false;
        document.getElementById('log').innerHTML = '';
        evtSource = new EventSource('/stream');
        evtSource.onmessage = e => {
          total++;
          document.getElementById('total').textContent = total;
          const div = document.createElement('div');
          div.className = 'msg';
          const data = JSON.parse(e.data);
          div.innerHTML = '<span class="time">' + (data.time||'') + '</span> '
            + '<span class="proto">[' + (data.protocol||'?') + ']</span> '
            + '<span class="content">' + (data.message||'') + '</span>';
          document.getElementById('log').prepend(div);
        };
      }
    });
}

function stopDecode() {
  if (evtSource) { evtSource.close(); evtSource = null; }
  fetch('/api/stop', {method: 'POST'});
  document.getElementById('status').textContent = 'Durduruldu';
  document.getElementById('status').style.color = '#ef4444';
  document.getElementById('startBtn').disabled = false;
  document.getElementById('stopBtn').disabled = true;
}
</script>
</body></html>"""

proc = None
proc_lock = threading.Lock()

def reader_thread(process):
    """Read multimon-ng output"""
    try:
        for line in process.stdout:
            line = line.strip()
            if line and not line.startswith('multimon-ng'):
                parts = line.split(':', 1)
                protocol = parts[0].strip() if len(parts) > 1 else 'UNKNOWN'
                message = parts[1].strip() if len(parts) > 1 else line
                with lock:
                    messages.append({
                        'time': time.strftime('%H:%M:%S'),
                        'protocol': protocol,
                        'message': message
                    })
    except Exception:
        pass

@app.route('/')
def index():
    return render_template_string(HTML)

@app.route('/api/start', methods=['POST'])
def start():
    global proc
    from flask import request
    freq = request.args.get('freq', '148.8125M')
    mode = request.args.get('mode', '-a POCSAG512 -a POCSAG1200 -a POCSAG2400')

    with proc_lock:
        if proc:
            proc.kill()
        # rtl_fm piped to multimon-ng
        cmd = f"rtl_fm -f {freq} -s 22050 -g 42 | multimon-ng {mode} -t raw -"
        proc = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
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
                current_len = len(messages)
                if current_len > last_len:
                    for item in list(messages)[last_len:]:
                        yield f"data: {json.dumps(item)}\n\n"
                    last_len = current_len
            time.sleep(0.3)
    import json
    return Response(generate(), mimetype='text/event-stream')

@app.route('/api/messages')
def api_messages():
    with lock:
        return jsonify(list(messages))

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8090)
