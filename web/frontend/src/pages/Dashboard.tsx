import { useEffect, useState, useCallback } from 'react';
import { api, post } from '../api/client';
import type { ServiceInfo, ServicesResponse, SystemStatus } from '../api/client';

export default function Dashboard() {
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
      setServices(svcRes.services);
      setActiveService(svcRes.active_service);
      setStatus(sysRes);
    } catch (err) {
      console.error('Failed to fetch dashboard data:', err);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const handleAction = async (id: string, action: 'start' | 'stop') => {
    setLoading((prev) => ({ ...prev, [id]: true }));
    try {
      await post(`/services/${id}/${action}`);
      await fetchData();
    } catch (err) {
      console.error(`Failed to ${action} service ${id}:`, err);
    } finally {
      setLoading((prev) => ({ ...prev, [id]: false }));
    }
  };

  const totalRunning = services.filter((s) => s.status === 'running').length;

  return (
    <div>
      <div className="page-header">
        <h2>Dashboard</h2>
        <p>System overview and service management</p>
      </div>

      <div className="stats-bar">
        <div className="stat-card">
          <div className="stat-value" style={{ color: 'var(--primary)' }}>
            {activeService || 'None'}
          </div>
          <div className="stat-label">Active Service</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">
            {totalRunning} / {services.length}
          </div>
          <div className="stat-label">Services Running</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">
            {status ? `${status.memory_mb} MB` : '--'}
          </div>
          <div className="stat-label">Memory Usage</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{status?.ws_clients ?? '--'}</div>
          <div className="stat-label">WS Clients</div>
        </div>
      </div>

      <div className="section">
        <div className="section-title">Services</div>
        <div className="grid-auto">
          {services.map((svc) => (
            <div className="card" key={svc.id}>
              <div className="card-header">
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{ fontSize: 24 }}>{svc.icon}</span>
                  <div>
                    <div className="card-title">{svc.name}</div>
                    <div className="text-sm text-muted">{svc.description}</div>
                  </div>
                </div>
              </div>

              <div className="flex-between">
                <span className={`badge badge-${svc.status}`}>
                  <span className="badge-dot" />
                  {svc.status}
                </span>

                <div className="flex-row gap-8">
                  {svc.status === 'stopped' || svc.status === 'error' ? (
                    <button
                      className="btn btn-success btn-sm"
                      disabled={loading[svc.id]}
                      onClick={() => handleAction(svc.id, 'start')}
                    >
                      {loading[svc.id] ? '...' : 'Start'}
                    </button>
                  ) : svc.status === 'running' ? (
                    <button
                      className="btn btn-danger btn-sm"
                      disabled={loading[svc.id]}
                      onClick={() => handleAction(svc.id, 'stop')}
                    >
                      {loading[svc.id] ? '...' : 'Stop'}
                    </button>
                  ) : (
                    <button className="btn btn-outline btn-sm" disabled>
                      {svc.status}...
                    </button>
                  )}
                </div>
              </div>

              <div style={{ marginTop: 12 }}>
                <span
                  className="badge"
                  style={{
                    background: svc.category === 'docker' ? 'rgba(59,130,246,0.12)' : 'rgba(168,85,247,0.12)',
                    color: svc.category === 'docker' ? 'var(--primary)' : '#a855f7',
                  }}
                >
                  {svc.category}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
