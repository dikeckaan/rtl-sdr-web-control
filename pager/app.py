#!/usr/bin/env python3
import os
import time
import subprocess
import threading
from flask import Flask, jsonify, request, send_from_directory
from flask_socketio import SocketIO

app = Flask(__name__, static_folder='/opt/pager/html', static_url_path='')
app.config['SECRET_KEY'] = 'pager-decoder-secret'
socketio = SocketIO(app, cors_allowed_origins='*', async_mode='threading')

LOG_FILE = '/var/log/pager/messages.log'
LIVE_LOG = '/tmp/pager_live.log'
FREQ_FILE = '/tmp/current_freq'
DEFAULT_FREQ = os.environ.get('PAGER_FREQ', '153.350M')


def get_current_freq():
    if os.path.exists(FREQ_FILE):
        with open(FREQ_FILE, 'r') as f:
            return f.read().strip()
    return DEFAULT_FREQ


def read_last_n_messages(n=100):
    messages = []
    if not os.path.exists(LOG_FILE):
        return messages
    try:
        with open(LOG_FILE, 'r', errors='replace') as f:
            lines = f.readlines()
            for line in lines[-n:]:
                line = line.strip()
                if line:
                    messages.append(line)
    except Exception:
        pass
    return messages


def tail_log():
    """Background thread that tails the live log and emits new messages."""
    while not os.path.exists(LIVE_LOG):
        time.sleep(0.5)

    with open(LIVE_LOG, 'r', errors='replace') as f:
        # Seek to end
        f.seek(0, 2)
        while True:
            line = f.readline()
            if line:
                line = line.strip()
                if line:
                    socketio.emit('new_message', {'message': line})
            else:
                time.sleep(0.2)


@app.route('/')
def index():
    return send_from_directory('/opt/pager/html', 'index.html')


@app.route('/api/messages')
def api_messages():
    messages = read_last_n_messages(100)
    return jsonify({'messages': messages, 'count': len(messages)})


@app.route('/api/freq', methods=['GET'])
def api_get_freq():
    return jsonify({'freq': get_current_freq()})


@app.route('/api/freq', methods=['POST'])
def api_set_freq():
    data = request.get_json()
    if not data or 'freq' not in data:
        return jsonify({'error': 'Missing freq parameter'}), 400

    new_freq = data['freq'].strip()
    if not new_freq:
        return jsonify({'error': 'Empty frequency'}), 400

    # Write new frequency
    with open(FREQ_FILE, 'w') as f:
        f.write(new_freq)

    # Kill rtl_fm so supervisor restarts the decoder with the new freq
    try:
        subprocess.run(['pkill', '-f', 'rtl_fm'], timeout=5)
    except Exception:
        pass

    return jsonify({'freq': new_freq, 'status': 'restarting'})


@socketio.on('connect')
def handle_connect():
    messages = read_last_n_messages(50)
    for msg in messages:
        socketio.emit('new_message', {'message': msg})


# Start the log tailer thread
log_thread = threading.Thread(target=tail_log, daemon=True)
log_thread.start()

if __name__ == '__main__':
    socketio.run(app, host='0.0.0.0', port=5000, allow_unsafe_werkzeug=True)
