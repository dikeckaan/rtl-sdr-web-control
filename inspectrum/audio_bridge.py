#!/usr/bin/env python3
"""
PulseAudio -> WebSocket audio bridge.
Captures audio from PulseAudio monitor, streams as PCM16 over WebSocket.
Works through Cloudflare tunnels (single HTTP port, /audio path).
"""

import asyncio
import subprocess
import websockets

PULSE_RATE = 48000
PULSE_CHANNELS = 2
CHUNK_FRAMES = 2400  # 50ms chunks at 48kHz
CHUNK_BYTES = CHUNK_FRAMES * PULSE_CHANNELS * 2  # int16 = 2 bytes

clients = set()


async def audio_producer():
    """Read from PulseAudio monitor and broadcast to all WebSocket clients."""
    while True:
        try:
            proc = await asyncio.create_subprocess_exec(
                "parecord",
                "--rate", str(PULSE_RATE),
                "--channels", str(PULSE_CHANNELS),
                "--format", "s16le",
                "--raw",
                "--device", "virtual_sink.monitor",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.DEVNULL,
            )

            while True:
                data = await proc.stdout.read(CHUNK_BYTES)
                if not data:
                    break

                dead = set()
                for ws in clients.copy():
                    try:
                        await ws.send(data)
                    except Exception:
                        dead.add(ws)
                clients.difference_update(dead)

        except Exception as e:
            print(f"audio_producer error: {e}", flush=True)
            await asyncio.sleep(2)


async def ws_handler(websocket):
    """Handle incoming WebSocket connections."""
    clients.add(websocket)
    print(f"Audio client connected ({len(clients)} total)", flush=True)
    try:
        async for _ in websocket:
            pass  # client doesn't send anything
    except Exception:
        pass
    finally:
        clients.discard(websocket)
        print(f"Audio client disconnected ({len(clients)} total)", flush=True)


async def main():
    # Wait for PulseAudio to be ready
    for _ in range(30):
        result = subprocess.run(
            ["pactl", "info"], capture_output=True, text=True
        )
        if result.returncode == 0:
            break
        await asyncio.sleep(1)

    asyncio.create_task(audio_producer())

    async with websockets.serve(ws_handler, "0.0.0.0", 8092):
        print("Audio bridge listening on ws://0.0.0.0:8092", flush=True)
        await asyncio.Future()  # run forever


if __name__ == "__main__":
    asyncio.run(main())
