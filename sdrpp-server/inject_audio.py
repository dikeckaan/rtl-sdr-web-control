import sys

HTML_PATH = "/usr/share/novnc/vnc.html"

with open(HTML_PATH, "r") as f:
    content = f.read()

if "SDR++ Audio Stream" not in content:
    js_code = """
    <!-- SDR++ Audio Stream -->
    <div id="sdr-audio-control" style="position:fixed; bottom:10px; right:10px; z-index:9999; background:rgba(0,0,0,0.7); color:white; padding:10px; border-radius:5px;">
        <label>SDR Audio: </label>
        <button id="sdr-audio-btn" style="padding: 5px;">Listen</button>
        <span id="sdr-audio-status">Stopped</span>
    </div>
    <script>
        document.getElementById("sdr-audio-btn").addEventListener("click", function() {
            var btn = this;
            if (btn.innerText === "Listen") {
                startAudio();
                btn.innerText = "Stop";
                document.getElementById("sdr-audio-status").innerText = "Playing...";
            } else {
                stopAudio();
                btn.innerText = "Listen";
                document.getElementById("sdr-audio-status").innerText = "Stopped";
            }
        });

        var audioCtx;
        var ws;
        var isPlaying = false;

        function startAudio() {
            if (isPlaying) return;
            
            audioCtx = new (window.AudioContext || window.webkitAudioContext)({sampleRate: 48000});
            
            // Connect to websockify on Nginx proxied /audio path
            var host = window.location.host;
            var protocol = window.location.protocol === "https:" ? "wss://" : "ws://";
            ws = new WebSocket(protocol + host + "/audio", ["binary"]);
            ws.binaryType = "arraybuffer";
            
            var startTime = 0;
            
            ws.onmessage = function(event) {
                var rawData = new Int16Array(event.data);
                if (rawData.length === 0) return;
                
                var buf = audioCtx.createBuffer(2, rawData.length / 2, 48000);
                var left = buf.getChannelData(0);
                var right = buf.getChannelData(1);
                
                for (var i = 0; i < rawData.length / 2; i++) {
                    left[i] = rawData[i*2] / 32768.0;
                    right[i] = rawData[(i*2)+1] / 32768.0;
                }
                
                var source = audioCtx.createBufferSource();
                source.buffer = buf;
                source.connect(audioCtx.destination);
                
                var currTime = audioCtx.currentTime;
                if (startTime < currTime) {
                    startTime = currTime + 0.05;
                }
                
                source.start(startTime);
                startTime += buf.duration;
            };
            
            ws.onerror = function() {
                document.getElementById("sdr-audio-status").innerText = "Error!";
            };
            ws.onclose = function() {
                document.getElementById("sdr-audio-status").innerText = "Disconnected";
                stopAudio();
            };
            
            isPlaying = true;
        }

        function stopAudio() {
            if (ws) ws.close();
            if (audioCtx) audioCtx.close();
            isPlaying = false;
        }
    </script>
    </body>
    """
    
    content = content.replace("</body>", js_code)
    
    with open(HTML_PATH, "w") as f:
        f.write(content)
