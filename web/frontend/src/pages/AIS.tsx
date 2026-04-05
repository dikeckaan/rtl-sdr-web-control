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
  ship_type: string;
  last_seen: string;
}

interface AISStatus {
  active_ships: number;
  total_messages: number;
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
              <strong style={{ fontSize: 14 }}>{ship.name || 'Unknown'}</strong>
              <table style={{ marginTop: 8, fontSize: 12, width: '100%' }}>
                <tbody>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>MMSI</td><td style={{ fontFamily: 'monospace' }}>{ship.mmsi}</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Type</td><td>{ship.ship_type || '--'}</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Speed</td><td>{ship.speed?.toFixed(1) ?? '--'} kn</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Course</td><td>{ship.course?.toFixed(0) ?? '--'}&deg;</td></tr>
                  <tr><td style={{ color: '#64748b', padding: '2px 8px 2px 0' }}>Position</td><td>{ship.lat.toFixed(4)}, {ship.lon.toFixed(4)}</td></tr>
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
  const [aisStatus, setAisStatus] = useState<AISStatus>({ active_ships: 0, total_messages: 0, status: 'stopped' });
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    // Fetch initial data
    api<{ ships: Ship[] }>('/ais/ships')
      .then((res) => setShips(res.ships ?? []))
      .catch(() => {});

    api<AISStatus>('/ais/status')
      .then((res) => setAisStatus(res))
      .catch(() => {});

    // WebSocket for real-time ship updates
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
        setAisStatus((prev) => ({
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
            <h2>🚢 AIS Ship Tracking</h2>
            <p>Real-time Automatic Identification System receiver</p>
          </div>
          <span className={`badge badge-${aisStatus.status === 'running' ? 'running' : 'stopped'}`}>
            <span className="badge-dot" />
            {aisStatus.status}
          </span>
        </div>
      </div>

      <div className="stats-bar" style={{ marginBottom: 16 }}>
        <div className="stat-card">
          <div className="stat-value" style={{ color: 'var(--success)' }}>{ships.length}</div>
          <div className="stat-label">Active Ships</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{aisStatus.total_messages}</div>
          <div className="stat-label">Total Messages</div>
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
