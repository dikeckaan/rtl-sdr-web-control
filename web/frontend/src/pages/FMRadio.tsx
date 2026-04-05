import { useEffect, useState } from 'react';
import { api, post } from '../api/client';

interface FMStatus {
  frequency: number;
  status: string;
  playing: boolean;
}

const PRESETS = [
  { name: 'TRT FM', freq: 93.3 },
  { name: 'Power FM', freq: 100.0 },
  { name: 'Kral FM', freq: 92.0 },
  { name: 'Best FM', freq: 98.0 },
  { name: 'Super FM', freq: 90.8 },
  { name: 'NTV Radyo', freq: 102.8 },
  { name: 'Joy Turk', freq: 88.7 },
  { name: 'Radyo D', freq: 96.4 },
  { name: 'Metro FM', freq: 97.2 },
  { name: 'Slow Turk', freq: 95.3 },
];

export default function FMRadio() {
  const [status, setStatus] = useState<FMStatus>({ frequency: 93.3, status: 'stopped', playing: false });
  const [manualFreq, setManualFreq] = useState('');
  const [tuning, setTuning] = useState(false);

  const fetchStatus = async () => {
    try {
      const res = await api<FMStatus>('/fm/status');
      setStatus(res);
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const tune = async (freq: number) => {
    setTuning(true);
    try {
      await post('/fm/tune', { frequency: freq });
      setStatus((prev) => ({ ...prev, frequency: freq }));
      await fetchStatus();
    } catch {
      /* ignore */
    } finally {
      setTuning(false);
    }
  };

  const handleManualTune = () => {
    const freq = parseFloat(manualFreq);
    if (freq >= 87.5 && freq <= 108.0) {
      tune(freq);
      setManualFreq('');
    }
  };

  return (
    <div>
      <div className="page-header">
        <h2>📻 FM Radio</h2>
        <p>Listen to FM broadcast stations</p>
      </div>

      <div className="grid-2" style={{ marginBottom: 24 }}>
        <div>
          {/* Frequency Display */}
          <div className="freq-display" style={{ marginBottom: 16 }}>
            <div className="freq-value">
              {status.frequency.toFixed(1)}
              <span className="freq-unit">MHz</span>
            </div>
            <div style={{ marginTop: 8 }}>
              <span className={`badge badge-${status.status === 'running' ? 'running' : 'stopped'}`}>
                <span className="badge-dot" />
                {status.status}
              </span>
            </div>
          </div>

          {/* Audio Player */}
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="card-title" style={{ marginBottom: 12 }}>Audio Stream</div>
            <div className="audio-player">
              <audio
                controls
                src="/api/v1/fm/stream"
                style={{ width: '100%' }}
              />
            </div>
          </div>

          {/* Manual Tune */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Manual Tune</div>
            <div className="flex-row">
              <input
                className="form-input"
                type="number"
                min="87.5"
                max="108.0"
                step="0.1"
                placeholder="e.g. 93.3"
                value={manualFreq}
                onChange={(e) => setManualFreq(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleManualTune()}
                style={{ flex: 1 }}
              />
              <button
                className="btn btn-primary"
                onClick={handleManualTune}
                disabled={tuning}
              >
                {tuning ? 'Tuning...' : 'Tune'}
              </button>
            </div>
          </div>
        </div>

        <div>
          {/* Presets */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 16 }}>Station Presets</div>
            <div className="preset-grid">
              {PRESETS.map((p) => (
                <button
                  key={p.name}
                  className={`preset-btn ${Math.abs(status.frequency - p.freq) < 0.05 ? 'active' : ''}`}
                  onClick={() => tune(p.freq)}
                  disabled={tuning}
                >
                  <span className="preset-name">{p.name}</span>
                  <span className="preset-freq">{p.freq.toFixed(1)} MHz</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
