import { useEffect, useState } from 'react';
import { api } from '../api/client';

interface Channel {
  index: number;
  name: string;
  freq: number;
  modulation: string;
}

export default function Airband() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [status, setStatus] = useState('stopped');

  const fetchChannels = async () => {
    try {
      const res = await api<{ channels: Channel[]; status: string }>('/airband/channels');
      setChannels(res.channels || []);
      setStatus(res.status);
    } catch { /* */ }
  };

  useEffect(() => {
    fetchChannels();
    const interval = setInterval(fetchChannels, 5000);
    return () => clearInterval(interval);
  }, []);

  const isRunning = status === 'running';

  return (
    <div>
      <div className="page-header">
        <div className="flex-between">
          <div>
            <h2>✈️ Havacılık Bandı</h2>
            <p>İstanbul havalimanı frekansları</p>
          </div>
          <span className={`badge badge-${isRunning ? 'running' : 'stopped'}`}>
            <span className="badge-dot" />
            {isRunning ? 'Dinleniyor' : 'Durdu'}
          </span>
        </div>
      </div>

      <div className="grid-2">
        {channels.map((ch) => (
          <div className="card" key={ch.index}>
            <div className="card-header">
              <div>
                <div className="card-title">✈️ {ch.name}</div>
                <div className="font-mono text-sm text-muted" style={{ marginTop: 2 }}>
                  {ch.freq.toFixed(3)} MHz · {ch.modulation.toUpperCase()}
                </div>
              </div>
            </div>

            <div className="audio-player">
              <audio
                controls
                src={isRunning ? `/api/v1/airband/stream/${ch.index}` : undefined}
                style={{ width: '100%' }}
              />
            </div>
            {!isRunning && (
              <div style={{ color: 'var(--text-muted)', fontSize: 12, marginTop: 8, textAlign: 'center' }}>
                Servisi başlatın
              </div>
            )}
          </div>
        ))}

        {channels.length === 0 && (
          <div className="card" style={{ gridColumn: '1/-1', textAlign: 'center', padding: 40 }}>
            <div style={{ fontSize: 48, marginBottom: 12 }}>✈️</div>
            <div className="text-muted">Kanal bilgisi yükleniyor...</div>
          </div>
        )}
      </div>
    </div>
  );
}
