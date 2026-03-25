#!/usr/bin/env python3
"""Inject audio streaming UI into noVNC page."""

HTML_PATH = "/usr/share/novnc/vnc.html"

with open(HTML_PATH, "r") as f:
    content = f.read()

if "sdr-audio-control" in content:
    import re
    content = re.sub(
        r'<!-- SDR\+\+ Audio Stream -->.*?</script>\s*</body>',
        '</body>',
        content,
        flags=re.DOTALL,
    )

AUDIO_WIDGET = """
<!-- SDR++ Audio Stream -->
<div id="sdr-audio-control" style="position:fixed;bottom:10px;right:10px;z-index:9999;
  background:rgba(22,33,62,0.95);color:#eee;padding:12px 16px;border-radius:8px;
  font-family:sans-serif;font-size:13px;border:1px solid #4ecca3;box-shadow:0 2px 12px rgba(0,0,0,0.4);">
  <div style="display:flex;align-items:center;gap:10px;">
    <span style="color:#4ecca3;font-weight:bold;">SDR Audio</span>
    <button id="sdr-audio-btn" style="padding:5px 14px;border:none;border-radius:4px;
      background:#4ecca3;color:#1a1a2e;font-weight:bold;cursor:pointer;">Play</button>
    <input id="sdr-vol" type="range" min="0" max="100" value="80"
      style="width:80px;accent-color:#4ecca3;">
    <span id="sdr-audio-status" style="color:#888;font-size:11px;">Stopped</span>
  </div>
</div>
<script>
(function(){
  var btn = document.getElementById("sdr-audio-btn");
  var vol = document.getElementById("sdr-vol");
  var status = document.getElementById("sdr-audio-status");
  var audioCtx, ws, gainNode, playing = false;
  var nextTime = 0;

  btn.onclick = function(){
    if(!playing) startAudio(); else stopAudio();
  };

  vol.oninput = function(){
    if(gainNode) gainNode.gain.value = this.value / 100;
  };

  function startAudio(){
    audioCtx = new (window.AudioContext || window.webkitAudioContext)({sampleRate:48000});
    gainNode = audioCtx.createGain();
    gainNode.gain.value = vol.value / 100;
    gainNode.connect(audioCtx.destination);
    nextTime = 0;

    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/audio");
    ws.binaryType = "arraybuffer";

    ws.onopen = function(){
      playing = true;
      btn.textContent = "Stop";
      btn.style.background = "#e94560";
      btn.style.color = "#fff";
      status.textContent = "Connected";
      status.style.color = "#4ecca3";
    };

    ws.onmessage = function(e){
      if(!e.data || e.data.byteLength < 4) return;

      var raw = new Int16Array(e.data);
      var frames = raw.length / 2;
      if(frames < 1) return;

      var buf = audioCtx.createBuffer(2, frames, 48000);
      var L = buf.getChannelData(0);
      var R = buf.getChannelData(1);
      for(var i = 0; i < frames; i++){
        L[i] = raw[i*2] / 32768;
        R[i] = raw[i*2+1] / 32768;
      }

      var src = audioCtx.createBufferSource();
      src.buffer = buf;
      src.connect(gainNode);

      var now = audioCtx.currentTime;
      if(nextTime < now) nextTime = now + 0.05;
      src.start(nextTime);
      nextTime += buf.duration;
    };

    ws.onerror = function(){
      status.textContent = "Error";
      status.style.color = "#e94560";
    };

    ws.onclose = function(){
      if(playing) stopAudio();
    };
  }

  function stopAudio(){
    playing = false;
    if(ws){ try{ws.close();}catch(e){} ws=null; }
    if(audioCtx){ try{audioCtx.close();}catch(e){} audioCtx=null; }
    btn.textContent = "Play";
    btn.style.background = "#4ecca3";
    btn.style.color = "#1a1a2e";
    status.textContent = "Stopped";
    status.style.color = "#888";
  }
})();
</script>
</body>
"""

# Force autoconnect + resize on load
AUTOCONNECT_SCRIPT = """
<script>
(function(){
  // Force autoconnect params
  if(window.location.search.indexOf('autoconnect') === -1){
    var sep = window.location.search ? '&' : '?';
    window.location.search += sep + 'autoconnect=true&resize=scale&path=websockify';
  }
})();
</script>
"""

content = content.replace("</head>", AUTOCONNECT_SCRIPT + "</head>")
content = content.replace("</body>", AUDIO_WIDGET)

with open(HTML_PATH, "w") as f:
    f.write(content)

print("Audio widget + autoconnect injected into noVNC")
