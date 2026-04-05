import { useEffect, useState, useRef } from 'react';
import { api, post, connectWS } from '../api/client';
import type { WSMessage } from '../api/client';
import { MapContainer, TileLayer, Marker, Popup, Polyline, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

interface ISSPosition {
  lat: number;
  lon: number;
  altitude: number;
  velocity: number;
  timestamp: string;
}

interface Pass {
  rise_time: string;
  set_time: string;
  max_elevation: number;
  duration_seconds: number;
  direction: string;
}

interface Capture {
  id: string;
  filename: string;
  timestamp: string;
  duration: number;
  size_mb: number;
  download_url: string;
}

interface ISSStatus {
  position: ISSPosition;
  passes: Pass[];
  captures: Capture[];
  recording: boolean;
  station: { lat: number; lon: number };
}

const issIcon = L.divIcon({
  className: '',
  html: `<div style="font-size:28px; filter: drop-shadow(0 2px 4px rgba(0,0,0,0.5));">🛰️</div>`,
  iconSize: [28, 28],
  iconAnchor: [14, 14],
});

const stationIcon = L.divIcon({
  className: '',
  html: `<div style="font-size:20px; filter: drop-shadow(0 1px 3px rgba(0,0,0,0.5));">📡</div>`,
  iconSize: [20, 20],
  iconAnchor: [10, 10],
});

function ISSMarker({ position }: { position: ISSPosition }) {
  const map = useMap();

  useEffect(() => {
    if (position.lat !== 0 || position.lon !== 0) {
      map.setView([position.lat, position.lon], map.getZoom(), { animate: true });
    }
  }, [position, map]);

  return (
    <Marker position={[position.lat, position.lon]} icon={issIcon}>
      <Popup>
        <div style={{ color: '#1e293b' }}>
          <strong>ISS</strong>
          <div style={{ fontSize: 12, marginTop: 4 }}>
            <div>Lat: {position.lat.toFixed(4)}</div>
            <div>Lon: {position.lon.toFixed(4)}</div>
            <div>Alt: {position.altitude?.toFixed(1)} km</div>
            <div>Speed: {position.velocity?.toFixed(0)} km/h</div>
          </div>
        </div>
      </Popup>
    </Marker>
  );
}

export default function ISS() {
  const [position, setPosition] = useState<ISSPosition>({ lat: 0, lon: 0, altitude: 408, velocity: 27600, timestamp: '' });
  const [passes, setPasses] = useState<Pass[]>([]);
  const [captures, setCaptures] = useState<Capture[]>([]);
  const [recording, setRecording] = useState(false);
  const [recDuration, setRecDuration] = useState('600');
  const [stationPos, setStationPos] = useState({ lat: 41.0, lon: 29.0 });
  const [countdown, setCountdown] = useState('');
  const [trail, setTrail] = useState<[number, number][]>([]);
  const wsRef = useRef<WebSocket | null>(null);

  const fetchData = async () => {
    try {
      const res = await api<ISSStatus>('/iss/status');
      if (res.position) setPosition(res.position);
      if (res.passes) setPasses(res.passes);
      if (res.captures) setCaptures(res.captures);
      setRecording(res.recording ?? false);
      if (res.station) setStationPos(res.station);
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    fetchData();

    wsRef.current = connectWS(['iss.position'], (msg: WSMessage) => {
      if (msg.topic === 'iss.position') {
        const pos = msg.data as ISSPosition;
        setPosition(pos);
        setTrail((prev) => [...prev.slice(-100), [pos.lat, pos.lon]]);
      }
    });

    return () => {
      wsRef.current?.close();
    };
  }, []);

  // Countdown timer for next pass
  useEffect(() => {
    if (passes.length === 0) return;

    const timer = setInterval(() => {
      const nextRise = new Date(passes[0].rise_time).getTime();
      const now = Date.now();
      const diff = nextRise - now;
      if (diff <= 0) {
        setCountdown('NOW!');
      } else {
        const h = Math.floor(diff / 3600000);
        const m = Math.floor((diff % 3600000) / 60000);
        const s = Math.floor((diff % 60000) / 1000);
        setCountdown(`${h}h ${m}m ${s}s`);
      }
    }, 1000);

    return () => clearInterval(timer);
  }, [passes]);

  const startRecording = async () => {
    try {
      await post('/iss/record/start', { duration: parseInt(recDuration, 10) });
      setRecording(true);
    } catch {
      /* ignore */
    }
  };

  const stopRecording = async () => {
    try {
      await post('/iss/record/stop');
      setRecording(false);
      await fetchData();
    } catch {
      /* ignore */
    }
  };

  const formatDateTime = (ts: string) => {
    try {
      return new Date(ts).toLocaleString('tr-TR', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
      });
    } catch {
      return ts;
    }
  };

  return (
    <div>
      <div className="page-header">
        <h2>🛰️ ISS Tracker</h2>
        <p>Track the International Space Station and record passes</p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 360px', gap: 16, marginBottom: 24 }}>
        {/* Map */}
        <div className="map-container" style={{ height: 420 }}>
          <MapContainer
            center={[position.lat || 41, position.lon || 29]}
            zoom={3}
            style={{ height: '100%', width: '100%' }}
          >
            <TileLayer
              attribution='&copy; CARTO'
              url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
            />
            <ISSMarker position={position} />
            <Marker position={[stationPos.lat, stationPos.lon]} icon={stationIcon}>
              <Popup>
                <div style={{ color: '#1e293b' }}>
                  <strong>Ground Station</strong>
                  <div style={{ fontSize: 12 }}>{stationPos.lat.toFixed(4)}, {stationPos.lon.toFixed(4)}</div>
                </div>
              </Popup>
            </Marker>
            {trail.length > 1 && (
              <Polyline positions={trail} pathOptions={{ color: '#3b82f6', weight: 2, opacity: 0.6 }} />
            )}
          </MapContainer>
        </div>

        {/* Sidebar Info */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* ISS Info */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>ISS Position</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <div>
                <div className="text-sm text-muted">Latitude</div>
                <div className="font-mono">{position.lat.toFixed(4)}</div>
              </div>
              <div>
                <div className="text-sm text-muted">Longitude</div>
                <div className="font-mono">{position.lon.toFixed(4)}</div>
              </div>
              <div>
                <div className="text-sm text-muted">Altitude</div>
                <div className="font-mono">{position.altitude?.toFixed(1)} km</div>
              </div>
              <div>
                <div className="text-sm text-muted">Speed</div>
                <div className="font-mono">{position.velocity?.toFixed(0)} km/h</div>
              </div>
            </div>
          </div>

          {/* Next Pass */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Next Pass</div>
            {passes.length > 0 ? (
              <>
                <div
                  className="font-mono"
                  style={{
                    fontSize: 28,
                    fontWeight: 700,
                    color: countdown === 'NOW!' ? 'var(--success)' : 'var(--primary)',
                    textAlign: 'center',
                    padding: '8px 0',
                  }}
                >
                  {countdown}
                </div>
                <div className="text-sm text-muted text-center">
                  Max El: {passes[0].max_elevation}&deg; | {passes[0].direction}
                </div>
              </>
            ) : (
              <div className="text-muted text-center">No pass data available</div>
            )}
          </div>

          {/* Recording Controls */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Recording</div>
            {recording ? (
              <div>
                <div className="flex-row mb-8" style={{ justifyContent: 'center' }}>
                  <span className="badge badge-running" style={{ fontSize: 14 }}>
                    <span className="badge-dot" />
                    Recording...
                  </span>
                </div>
                <button className="btn btn-danger" style={{ width: '100%' }} onClick={stopRecording}>
                  Stop Recording
                </button>
              </div>
            ) : (
              <div>
                <div className="form-group">
                  <label className="form-label">Duration (seconds)</label>
                  <input
                    className="form-input"
                    type="number"
                    value={recDuration}
                    onChange={(e) => setRecDuration(e.target.value)}
                  />
                </div>
                <button className="btn btn-success" style={{ width: '100%' }} onClick={startRecording}>
                  Start Recording
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Pass Predictions Table */}
      <div className="section">
        <div className="section-title">Pass Predictions (Next 10)</div>
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Rise Time</th>
                <th>Set Time</th>
                <th>Max Elevation</th>
                <th>Duration</th>
                <th>Direction</th>
              </tr>
            </thead>
            <tbody>
              {passes.length === 0 ? (
                <tr>
                  <td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: 24 }}>
                    No pass predictions available
                  </td>
                </tr>
              ) : (
                passes.slice(0, 10).map((p, i) => (
                  <tr key={i}>
                    <td className="font-mono">{formatDateTime(p.rise_time)}</td>
                    <td className="font-mono">{formatDateTime(p.set_time)}</td>
                    <td>
                      <span
                        style={{
                          color: p.max_elevation >= 45 ? 'var(--success)' : p.max_elevation >= 20 ? 'var(--warning)' : 'var(--text-muted)',
                          fontWeight: 600,
                        }}
                      >
                        {p.max_elevation}&deg;
                      </span>
                    </td>
                    <td>{Math.floor(p.duration_seconds / 60)}m {p.duration_seconds % 60}s</td>
                    <td>{p.direction}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Captures */}
      <div className="section">
        <div className="section-title">Captures</div>
        {captures.length === 0 ? (
          <div className="card text-center text-muted" style={{ padding: 32 }}>
            No captures yet. Record an ISS pass to get started.
          </div>
        ) : (
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Filename</th>
                  <th>Duration</th>
                  <th>Size</th>
                  <th>Download</th>
                </tr>
              </thead>
              <tbody>
                {captures.map((c) => (
                  <tr key={c.id}>
                    <td className="font-mono">{formatDateTime(c.timestamp)}</td>
                    <td>{c.filename}</td>
                    <td>{Math.floor(c.duration / 60)}m {c.duration % 60}s</td>
                    <td>{c.size_mb?.toFixed(1)} MB</td>
                    <td>
                      <a
                        href={c.download_url}
                        className="btn btn-outline btn-sm"
                        download
                      >
                        Download
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
