import { useEffect, useState } from 'react';
import { api } from '../api/client';

interface Channel {
  id: number;
  name: string;
  frequency: number;
  active: boolean;
  signal_strength?: number;
}

interface AirbandStatus {
  channels: Channel[];
  status: string;
}

const DEFAULT_CHANNELS: Channel[] = [
  { id: 1, name: 'Istanbul Tower', frequency: 118.100, active: false },
  { id: 2, name: 'Istanbul Approach', frequency: 120.700, active: false },
  { id: 3, name: 'Istanbul Ground', frequency: 121.900, active: false },
  { id: 4, name: 'Istanbul ATIS', frequency: 128.025, active: false },
];

export default function Airband() {
  const [channels, setChannels] = useState<Channel[]>(DEFAULT_CHANNELS);
  const [status, setStatus] = useState('stopped');

  const fetchStatus = async () => {
    try {
      const res = await api<AirbandStatus>('/airband/status');
      setChannels(res.channels ?? DEFAULT_CHANNELS);
      setStatus(res.status);
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div>
      <div className="page-header">
        <div className="flex-between">
          <div>
            <h2>✈️ Airband</h2>
            <p>Aviation radio monitoring - multi-channel receiver</p>
          </div>
          <span className={`badge badge-${status === 'running' ? 'running' : 'stopped'}`}>
            <span className="badge-dot" />
            {status}
          </span>
        </div>
      </div>

      <div className="grid-2">
        {channels.map((ch) => (
          <div className="card" key={ch.id}>
            <div className="card-header">
              <div>
                <div className="card-title">{ch.name}</div>
                <div className="font-mono text-sm text-muted" style={{ marginTop: 2 }}>
                  {ch.frequency.toFixed(3)} MHz
                </div>
              </div>
              <span className={`badge ${ch.active ? 'badge-running' : 'badge-stopped'}`}>
                <span className="badge-dot" />
                {ch.active ? 'Active' : 'Idle'}
              </span>
            </div>

            {/* Signal Strength Bar */}
            <div style={{ marginBottom: 12 }}>
              <div className="text-sm text-muted" style={{ marginBottom: 4 }}>
                Signal
              </div>
              <div
                style={{
                  width: '100%',
                  height: 6,
                  background: 'var(--bg)',
                  borderRadius: 3,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    width: `${ch.signal_strength ?? 0}%`,
                    height: '100%',
                    background: ch.active
                      ? 'var(--success)'
                      : 'var(--border)',
                    borderRadius: 3,
                    transition: 'width 0.5s',
                  }}
                />
              </div>
            </div>

            {/* Audio Player */}
            <div className="audio-player">
              <audio
                controls
                src={`/api/v1/airband/stream/${ch.id}`}
                style={{ width: '100%' }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
