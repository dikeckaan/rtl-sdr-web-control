#!/usr/bin/env python3
"""rtl_433 Web Dashboard - Real-time 433MHz device monitor"""

from flask import Flask, Response, jsonify, render_template_string
import subprocess
import json
import threading
import time
from collections import deque

app = Flask(__name__)
events = deque(maxlen=500)
lock = threading.Lock()

HTML = """<!DOCTYPE html>
<html><head>
<meta charset="UTF-8">
<title>rtl_433 Monitor</title>
<style>
* { margin:0; padding:0; box-sizing:border-box; }
body { font-family: 'Courier New', monospace; background: #0a0e17; color: #e2e8f0; }
.header { padding: 20px; text-align: center; border-bottom: 1px solid #1e293b; }
.header h1 { color: #22c55e; font-size: 1.4em; }
.header p { color: #64748b; font-size: 0.85em; margin-top: 4px; }
.stats { display: flex; gap: 20px; justify-content: center; padding: 15px; }
.stat { background: #151822; padding: 10px 20px; border-radius: 8px; text-align: center; border: 1px solid #252a3a; }
.stat-num { font-size: 1.5em; color: #3b82f6; font-weight: bold; }
.stat-label { font-size: 0.7em; color: #64748b; }
#events { padding: 10px 20px; }
.event { background: #151822; margin: 6px 0; padding: 12px 16px; border-radius: 8px;
  border-left: 3px solid #3b82f6; font-size: 0.85em; animation: fadeIn 0.3s; }
.event .time { color: #64748b; }
.event .model { color: #22c55e; font-weight: bold; }
.event .data { color: #94a3b8; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(-10px); } }
.empty { text-align: center; color: #475569; padding: 60px; font-size: 1.1em; }
</style>
</head><body>
<div class="header">
  <h1>📡 rtl_433 Monitor</h1>
  <p>433 MHz Cihaz Izleme - Gerçek Zamanli</p>
</div>
<div class="stats">
  <div class="stat"><div class="stat-num" id="total">0</div><div class="stat-label">Toplam Sinyal</div></div>
  <div class="stat"><div class="stat-num" id="devices">0</div><div class="stat-label">Benzersiz Cihaz</div></div>
  <div class="stat"><div class="stat-num" id="rate">0</div><div class="stat-label">Sinyal/dk</div></div>
</div>
<div id="events"><div class="empty">Sinyal bekleniyor... 433 MHz cihazlar dinleniyor</div></div>
<script>
const seen = new Set();
let total = 0;
const startTime = Date.now();

function addEvent(data) {
  total++;
  if (data.model) seen.add(data.model + (data.id || ''));
  document.getElementById('total').textContent = total;
  document.getElementById('devices').textContent = seen.size;
  const elapsed = (Date.now() - startTime) / 60000;
  document.getElementById('rate').textContent = elapsed > 0 ? (total / elapsed).toFixed(1) : '0';

  const container = document.getElementById('events');
  if (container.querySelector('.empty')) container.innerHTML = '';
  const div = document.createElement('div');
  div.className = 'event';
  const time = data.time || new Date().toLocaleTimeString();
  const model = data.model || 'Unknown';
  const rest = Object.entries(data).filter(([k]) => !['time','model'].includes(k))
    .map(([k,v]) => `${k}: ${v}`).join(' | ');
  div.innerHTML = `<span class="time">${time}</span> <span class="model">${model}</span> <span class="data">${rest}</span>`;
  container.prepend(div);
  if (container.children.length > 200) container.lastChild.remove();
}

// SSE stream
const evtSource = new EventSource('/stream');
evtSource.onmessage = e => { try { addEvent(JSON.parse(e.data)); } catch(ex) {} };

// Also poll history
fetch('/api/events').then(r=>r.json()).then(data => data.forEach(addEvent));
</script>
</body></html>"""

def rtl433_reader():
    """Read rtl_433 JSON output in background"""
    while True:
        try:
            proc = subprocess.Popen(
                ['rtl_433', '-F', 'json', '-M', 'time:utc'],
                stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True
            )
            for line in proc.stdout:
                line = line.strip()
                if line:
                    try:
                        data = json.loads(line)
                        with lock:
                            events.append(data)
                    except json.JSONDecodeError:
                        pass
        except Exception:
            time.sleep(5)

@app.route('/')
def index():
    return render_template_string(HTML)

@app.route('/api/events')
def api_events():
    with lock:
        return jsonify(list(events))

@app.route('/stream')
def stream():
    def generate():
        last_len = 0
        while True:
            with lock:
                current_len = len(events)
                if current_len > last_len:
                    for item in list(events)[last_len:]:
                        yield f"data: {json.dumps(item)}\n\n"
                    last_len = current_len
            time.sleep(0.5)
    return Response(generate(), mimetype='text/event-stream')

if __name__ == '__main__':
    t = threading.Thread(target=rtl433_reader, daemon=True)
    t.start()
    app.run(host='0.0.0.0', port=8090)
