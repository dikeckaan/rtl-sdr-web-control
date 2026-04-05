import { useEffect, useState, useRef } from 'react';
import { api, post, connectWS } from '../api/client';
import type { WSMessage } from '../api/client';
import { MapContainer, TileLayer, Marker, Popup, Polyline, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

interface ISSPosition {
  lat: number;
  lon: number;
  alt: number;
  timestamp: string;
}

interface Pass {
  rise_time: string;
  max_time: string;
  set_time: string;
  max_alt: number;
  duration_sec: number;
  visible: boolean;
}

interface Capture {
  filename: string;
  size: number;
  mod_time: string;
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
            <div>Enlem: {position.lat.toFixed(4)}</div>
            <div>Boylam: {position.lon.toFixed(4)}</div>
            <div>Yukseklik: {position.alt?.toFixed(1)} km</div>
          </div>
        </div>
      </Popup>
    </Marker>
  );
}

export default function ISS() {
  const [position, setPosition] = useState<ISSPosition>({ lat: 0, lon: 0, alt: 408, timestamp: '' });
  const [passes, setPasses] = useState<Pass[]>([]);
  const [captures, setCaptures] = useState<Capture[]>([]);
  const [recording, setRecording] = useState(false);
  const [recDuration, setRecDuration] = useState('300');
  const [stationPos, setStationPos] = useState({ lat: 41.0, lon: 29.0, alt: 0 });
  const [countdown, setCountdown] = useState('');
  const [trail, setTrail] = useState<[number, number][]>([]);
  const wsRef = useRef<WebSocket | null>(null);

  const fetchData = async () => {
    try {
      const [posRes, passRes, statusRes, capRes] = await Promise.all([
        api<ISSPosition>('/iss/position').catch(() => null),
        api<{ passes: Pass[]; station: { lat: number; lon: number; alt: number } }>('/iss/passes').catch(() => null),
        api<{ status: string; recording: boolean; file: string }>('/iss/status').catch(() => null),
        api<{ captures: Capture[] }>('/iss/captures').catch(() => null),
      ]);
      if (posRes) setPosition(posRes);
      if (passRes) {
        setPasses(passRes.passes ?? []);
        if (passRes.station) setStationPos(passRes.station);
      }
      if (statusRes) setRecording(statusRes.recording ?? false);
      if (capRes) setCaptures(capRes.captures ?? []);
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

  // Sonraki gecis icin geri sayim
  useEffect(() => {
    if (passes.length === 0) return;

    const timer = setInterval(() => {
      const nextRise = new Date(passes[0].rise_time).getTime();
      const now = Date.now();
      const diff = nextRise - now;
      if (diff <= 0) {
        setCountdown('SIMDI!');
      } else {
        const h = Math.floor(diff / 3600000);
        const m = Math.floor((diff % 3600000) / 60000);
        const s = Math.floor((diff % 60000) / 1000);
        setCountdown(`${h}sa ${m}dk ${s}sn`);
      }
    }, 1000);

    return () => clearInterval(timer);
  }, [passes]);

  const startRecording = async () => {
    try {
      await post('/iss/record', { duration: parseInt(recDuration, 10) });
      setRecording(true);
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

  const formatSize = (bytes: number) => {
    if (!bytes) return '--';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div>
      <div className="page-header">
        <h2>ISS Takip</h2>
        <p>Uluslararasi Uzay Istasyonu takibi ve gecis kaydi</p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 360px', gap: 16, marginBottom: 24 }}>
        {/* Harita */}
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
                  <strong>Yer Istasyonu</strong>
                  <div style={{ fontSize: 12 }}>{stationPos.lat.toFixed(4)}, {stationPos.lon.toFixed(4)}</div>
                </div>
              </Popup>
            </Marker>
            {trail.length > 1 && (
              <Polyline positions={trail} pathOptions={{ color: '#3b82f6', weight: 2, opacity: 0.6 }} />
            )}
          </MapContainer>
        </div>

        {/* Yan Panel Bilgileri */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* ISS Konumu */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>ISS Konumu</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <div>
                <div className="text-sm text-muted">Enlem</div>
                <div className="font-mono">{position.lat.toFixed(4)}</div>
              </div>
              <div>
                <div className="text-sm text-muted">Boylam</div>
                <div className="font-mono">{position.lon.toFixed(4)}</div>
              </div>
              <div>
                <div className="text-sm text-muted">Yukseklik</div>
                <div className="font-mono">{position.alt?.toFixed(1)} km</div>
              </div>
            </div>
          </div>

          {/* Sonraki Gecis */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Sonraki Gecis</div>
            {passes.length > 0 ? (
              <>
                <div
                  className="font-mono"
                  style={{
                    fontSize: 28,
                    fontWeight: 700,
                    color: countdown === 'SIMDI!' ? 'var(--success)' : 'var(--primary)',
                    textAlign: 'center',
                    padding: '8px 0',
                  }}
                >
                  {countdown}
                </div>
                <div className="text-sm text-muted text-center">
                  Maks Yukseklik: {passes[0].max_alt}&deg; | {passes[0].visible ? 'Gorunur' : 'Gorunmez'}
                </div>
              </>
            ) : (
              <div className="text-muted text-center">Gecis verisi mevcut degil</div>
            )}
          </div>

          {/* Kayit Kontrolleri */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Kayit</div>
            {recording ? (
              <div>
                <div className="flex-row mb-8" style={{ justifyContent: 'center' }}>
                  <span className="badge badge-running" style={{ fontSize: 14 }}>
                    <span className="badge-dot" />
                    Kayit yapiliyor...
                  </span>
                </div>
              </div>
            ) : (
              <div>
                <div className="form-group">
                  <label className="form-label">Sure (saniye)</label>
                  <input
                    className="form-input"
                    type="number"
                    value={recDuration}
                    onChange={(e) => setRecDuration(e.target.value)}
                  />
                </div>
                <button className="btn btn-success" style={{ width: '100%' }} onClick={startRecording}>
                  Kaydi Baslat
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Gecis Tahminleri Tablosu */}
      <div className="section">
        <div className="section-title">Gecis Tahminleri (Sonraki 10)</div>
        <div className="table-container">
          <table>
            <thead>
              <tr>
                <th>Yukselme Zamani</th>
                <th>Maks Zaman</th>
                <th>Batis Zamani</th>
                <th>Maks Yukseklik</th>
                <th>Sure</th>
                <th>Gorunurluk</th>
              </tr>
            </thead>
            <tbody>
              {passes.length === 0 ? (
                <tr>
                  <td colSpan={6} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: 24 }}>
                    Gecis tahmini mevcut degil
                  </td>
                </tr>
              ) : (
                passes.slice(0, 10).map((p, i) => (
                  <tr key={i}>
                    <td className="font-mono">{formatDateTime(p.rise_time)}</td>
                    <td className="font-mono">{formatDateTime(p.max_time)}</td>
                    <td className="font-mono">{formatDateTime(p.set_time)}</td>
                    <td>
                      <span
                        style={{
                          color: p.max_alt >= 45 ? 'var(--success)' : p.max_alt >= 20 ? 'var(--warning)' : 'var(--text-muted)',
                          fontWeight: 600,
                        }}
                      >
                        {p.max_alt}&deg;
                      </span>
                    </td>
                    <td>{Math.floor(p.duration_sec / 60)}dk {p.duration_sec % 60}sn</td>
                    <td>{p.visible ? 'Evet' : 'Hayir'}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Kayitlar */}
      <div className="section">
        <div className="section-title">Kayitlar</div>
        {captures.length === 0 ? (
          <div className="card text-center text-muted" style={{ padding: 32 }}>
            Henuz kayit yok. Baslamak icin bir ISS gecisi kaydedin.
          </div>
        ) : (
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th>Dosya Adi</th>
                  <th>Boyut</th>
                  <th>Tarih</th>
                  <th>Indir</th>
                </tr>
              </thead>
              <tbody>
                {captures.map((c, i) => (
                  <tr key={i}>
                    <td className="font-mono">{c.filename}</td>
                    <td>{formatSize(c.size)}</td>
                    <td className="font-mono">{formatDateTime(c.mod_time)}</td>
                    <td>
                      <a
                        href={`/api/v1/iss/captures/${encodeURIComponent(c.filename)}`}
                        className="btn btn-outline btn-sm"
                        download
                      >
                        Indir
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
