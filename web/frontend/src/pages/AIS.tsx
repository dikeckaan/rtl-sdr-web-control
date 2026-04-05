import { useEffect, useState, useRef } from 'react';
import { api, connectWS } from '../api/client';
import type { WSMessage } from '../api/client';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

interface Ship {
  mmsi: string;
  name: string;
  lat: number;
  lon: number;
  speed: number;
  course: number;
  heading: number;
  last_seen: string;
}

interface AISStats {
  total_messages: number;
  active_ships: number;
  status: string;
}

function createShipIcon(heading: number): L.DivIcon {
  return L.divIcon({
    className: '',
    html: `<div style="
      transform: rotate(${heading}deg);
      width: 24px; height: 24px;
      display: flex; align-items: center; justify-content: center;
      font-size: 20px; filter: drop-shadow(0 1px 3px rgba(0,0,0,0.6));
    ">🔺</div>`,
    iconSize: [24, 24],
    iconAnchor: [12, 12],
  });
}

function ShipMarkers({ ships }: { ships: Ship[] }) {
  const map = useMap();

  useEffect(() => {
    if (ships.length > 0) {
      const bounds = L.latLngBounds(ships.map((s) => [s.lat, s.lon]));
      if (bounds.isValid()) {
        map.fitBounds(bounds, { padding: [50, 50], maxZoom: 14 });
      }
    }
  }, []); // only on first load

  return (
    <>
      {ships.map((ship) => (
        <Marker
          key={ship.mmsi}
          position={[ship.lat, ship.lon]}
          icon={createShipIcon(ship.heading || ship.course || 0)}
        >
          <Popup>
            <div style={{ color: '#1e293b', minWidth: 200 }}>
              <strong style={{ fontSize: 14 }}>{ship.name || 'Bilinmiyor'}</strong>
              <table style={{ marginTop: 8, fontSize: 12, width: '100%' }}>
                <tbody>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>MMSI</td><td style={{ fontFamily: 'monospace' }}>{ship.mmsi}</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Hiz</td><td>{ship.speed?.toFixed(1) ?? '--'} kn</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Rota</td><td>{ship.course?.toFixed(0) ?? '--'}&deg;</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Konum</td><td>{ship.lat.toFixed(4)}, {ship.lon.toFixed(4)}</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Son Gorulme</td><td>{ship.last_seen ? new Date(ship.last_seen).toLocaleString('tr-TR') : '--'}</td></tr>
                </tbody>
              </table>
            </div>
          </Popup>
        </Marker>
      ))}
    </>
  );
}

export default function AIS() {
  const [ships, setShips] = useState<Ship[]>([]);
  const [stats, setStats] = useState<AISStats>({ active_ships: 0, total_messages: 0, status: 'stopped' });
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    // Gemi listesini ve istatistikleri cek
    api<{ ships: Ship[]; stats: { total_messages: number; active_ships: number } }>('/ais/ships')
      .then((res) => {
        setShips(res.ships ?? []);
        if (res.stats) {
          setStats((prev) => ({ ...prev, total_messages: res.stats.total_messages, active_ships: res.stats.active_ships }));
        }
      })
      .catch(() => {});

    api<AISStats>('/ais/stats')
      .then((res) => setStats(res))
      .catch(() => {});

    // Gercek zamanli gemi guncellemeleri icin WebSocket
    wsRef.current = connectWS(['ais.ships'], (msg: WSMessage) => {
      if (msg.topic === 'ais.ships') {
        const updated = msg.data as Ship;
        setShips((prev) => {
          const idx = prev.findIndex((s) => s.mmsi === updated.mmsi);
          if (idx >= 0) {
            const copy = [...prev];
            copy[idx] = updated;
            return copy;
          }
          return [...prev, updated];
        });
        setStats((prev) => ({
          ...prev,
          total_messages: prev.total_messages + 1,
        }));
      }
    });

    return () => {
      wsRef.current?.close();
    };
  }, []);

  return (
    <div>
      <div className="page-header">
        <div className="flex-between">
          <div>
            <h2>AIS Gemi Takibi</h2>
            <p>Gercek zamanli Otomatik Tanimlama Sistemi alicisi</p>
          </div>
          <span className={`badge badge-${stats.status === 'running' ? 'running' : 'stopped'}`}>
            <span className="badge-dot" />
            {stats.status === 'running' ? 'Aktif' : 'Durduruldu'}
          </span>
        </div>
      </div>

      <div className="stats-bar" style={{ marginBottom: 16 }}>
        <div className="stat-card">
          <div className="stat-value" style={{ color: 'var(--success)' }}>{ships.length}</div>
          <div className="stat-label">Aktif Gemiler</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats.total_messages}</div>
          <div className="stat-label">Toplam Mesaj</div>
        </div>
      </div>

      <div className="map-container" style={{ height: 'calc(100vh - 280px)', minHeight: 400 }}>
        <MapContainer
          center={[40.94, 29.14]}
          zoom={11}
          style={{ height: '100%', width: '100%' }}
        >
          <TileLayer
            attribution='&copy; <a href="https://carto.com/">CARTO</a>'
            url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
          />
          <ShipMarkers ships={ships} />
        </MapContainer>
      </div>
    </div>
  );
}
