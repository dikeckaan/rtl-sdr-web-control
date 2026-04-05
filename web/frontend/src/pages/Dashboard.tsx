import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, post } from '../api/client';
import type { ServiceInfo, ServicesResponse, SystemStatus } from '../api/client';

const ICON_MAP: Record<string, string> = {
  radio: '📻',
  antenna: '📡',
  plane: '✈️',
  'message-square': '📟',
  ship: '🚢',
  satellite: '🛰️',
  signal: '📶',
  container: '🐳',
};

const SERVICE_ROUTES: Record<string, string> = {
  fmradio: '/fm',
  hamradio: '/ham',
  airband: '/airband',
  pager: '/pager',
  ais: '/ais',
  iss: '/iss',
  gsm: '/gsm',
};

const STATUS_TR: Record<string, string> = {
  running: 'Çalışıyor',
  stopped: 'Durdu',
  starting: 'Başlıyor',
  stopping: 'Duruyor',
  error: 'Hata',
};

export default function Dashboard() {
  const navigate = useNavigate();
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [activeService, setActiveService] = useState('');
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState<Record<string, boolean>>({});

  const fetchData = useCallback(async () => {
    try {
      const [svcRes, sysRes] = await Promise.all([
        api<ServicesResponse>('/services'),
        api<SystemStatus>('/system/status'),
      ]);
      setServices(svcRes.services || []);
      setActiveService(svcRes.active_service || '');
      setStatus(sysRes);
    } catch (err) {
      console.error('Veri alınamadı:', err);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 8000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleAction = async (id: string, action: 'start' | 'stop') => {
    setLoading((prev) => ({ ...prev, [id]: true }));
    try {
      await post(`/services/${id}/${action}`);
      await fetchData();
    } catch (err) {
      console.error(`${action} hatası:`, err);
    } finally {
      setLoading((prev) => ({ ...prev, [id]: false }));
    }
  };

  const handleStopAll = async () => {
    await post('/services/stop-all');
    await fetchData();
  };

  const totalRunning = services.filter((s) => s.status === 'running').length;

  // Stable sort: native first, then docker, alphabetical within groups
  const sorted = [...services].sort((a, b) => {
    if (a.category !== b.category) return a.category === 'native' ? -1 : 1;
    return a.name.localeCompare(b.name, 'tr');
  });

  return (
    <div>
      <div className="page-header">
        <div className="flex-between">
          <div>
            <h2>Kontrol Paneli</h2>
            <p>Sistem durumu ve servis yönetimi</p>
          </div>
          <button className="btn btn-danger btn-sm" onClick={handleStopAll}>
            Tümünü Durdur
          </button>
        </div>
      </div>

      <div className="stats-bar">
        <div className="stat-card">
          <div className="stat-value" style={{ color: 'var(--primary)', fontSize: 18 }}>
            {activeService ? (services.find(s => s.id === activeService)?.name || activeService) : 'Yok'}
          </div>
          <div className="stat-label">Aktif Servis</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">
            <span style={{ color: 'var(--success)' }}>{totalRunning}</span>
            <span style={{ color: 'var(--text-muted)', fontSize: 16 }}> / {services.length}</span>
          </div>
          <div className="stat-label">Çalışan Servis</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{status ? `${status.memory_mb} MB` : '--'}</div>
          <div className="stat-label">Bellek</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{status?.station?.name || '--'}</div>
          <div className="stat-label">İstasyon</div>
        </div>
      </div>

      <div className="section">
        <div className="section-title">Servisler</div>
        <div className="grid-auto">
          {sorted.map((svc) => {
            const icon = ICON_MAP[svc.icon] || '📡';
            const route = SERVICE_ROUTES[svc.id];
            const isRunning = svc.status === 'running';

            return (
              <div
                className="card"
                key={svc.id}
                style={{
                  borderLeft: isRunning ? '3px solid var(--success)' : undefined,
                  cursor: route && isRunning ? 'pointer' : undefined,
                }}
                onClick={() => {
                  if (route && isRunning) navigate(route);
                }}
              >
                <div className="card-header">
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <span style={{ fontSize: 28 }}>{icon}</span>
                    <div>
                      <div className="card-title">{svc.name}</div>
                      <div className="text-sm text-muted">{svc.description}</div>
                    </div>
                  </div>
                </div>

                <div className="flex-between" onClick={(e) => e.stopPropagation()}>
                  <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                    <span className={`badge badge-${svc.status}`}>
                      <span className="badge-dot" />
                      {STATUS_TR[svc.status] || svc.status}
                    </span>
                    <span
                      className="badge"
                      style={{
                        background: svc.category === 'docker' ? 'rgba(59,130,246,0.1)' : 'rgba(168,85,247,0.1)',
                        color: svc.category === 'docker' ? '#60a5fa' : '#c084fc',
                        fontSize: 10,
                      }}
                    >
                      {svc.category === 'docker' ? 'Docker' : 'Native'}
                    </span>
                  </div>

                  <div className="flex-row gap-8">
                    {(svc.status === 'stopped' || svc.status === 'error') && (
                      <button
                        className="btn btn-success btn-sm"
                        disabled={loading[svc.id]}
                        onClick={() => handleAction(svc.id, 'start')}
                      >
                        {loading[svc.id] ? '...' : 'Başlat'}
                      </button>
                    )}
                    {svc.status === 'running' && (
                      <>
                        <button
                          className="btn btn-danger btn-sm"
                          disabled={loading[svc.id]}
                          onClick={() => handleAction(svc.id, 'stop')}
                        >
                          {loading[svc.id] ? '...' : 'Durdur'}
                        </button>
                        {svc.category === 'docker' && (
                          <a
                            className="btn btn-primary btn-sm"
                            href={`http://${window.location.hostname}:8090`}
                            target="_blank"
                            rel="noreferrer"
                            onClick={(e) => e.stopPropagation()}
                          >
                            Aç ↗
                          </a>
                        )}
                      </>
                    )}
                    {(svc.status === 'starting' || svc.status === 'stopping') && (
                      <button className="btn btn-outline btn-sm" disabled>
                        {STATUS_TR[svc.status]}...
                      </button>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
