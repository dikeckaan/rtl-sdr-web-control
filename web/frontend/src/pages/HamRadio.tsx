import { useEffect, useState } from 'react';
import { api, post } from '../api/client';

interface HamStatus {
  frequency: number;
  mode: string;
  status: string;
}

type Mode = 'FM' | 'AM' | 'USB' | 'LSB';

const MODES: Mode[] = ['FM', 'AM', 'USB', 'LSB'];

const PRESETS = [
  { name: '2m Call', freq: 145.500, mode: 'FM' as Mode, band: '2m' },
  { name: '2m Simplex', freq: 145.000, mode: 'FM' as Mode, band: '2m' },
  { name: '70cm Call', freq: 433.500, mode: 'FM' as Mode, band: '70cm' },
  { name: '70cm Rep', freq: 434.650, mode: 'FM' as Mode, band: '70cm' },
  { name: '40m SSB', freq: 7.074, mode: 'USB' as Mode, band: '40m' },
  { name: '20m SSB', freq: 14.074, mode: 'USB' as Mode, band: '20m' },
  { name: '10m FM', freq: 29.600, mode: 'FM' as Mode, band: '10m' },
  { name: '80m LSB', freq: 3.573, mode: 'LSB' as Mode, band: '80m' },
  { name: '15m USB', freq: 21.074, mode: 'USB' as Mode, band: '15m' },
  { name: '6m SSB', freq: 50.313, mode: 'USB' as Mode, band: '6m' },
];

export default function HamRadio() {
  const [status, setStatus] = useState<HamStatus>({ frequency: 145.5, mode: 'FM', status: 'stopped' });
  const [manualFreq, setManualFreq] = useState('');
  const [manualMode, setManualMode] = useState<Mode>('FM');
  const [tuning, setTuning] = useState(false);

  const fetchStatus = async () => {
    try {
      const res = await api<HamStatus>('/ham/status');
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

  const tune = async (freq: number, mode: string) => {
    setTuning(true);
    try {
      await post('/ham/tune', { frequency: freq, mode });
      setStatus((prev) => ({ ...prev, frequency: freq, mode }));
      await fetchStatus();
    } catch {
      /* ignore */
    } finally {
      setTuning(false);
    }
  };

  const handleManualTune = () => {
    const freq = parseFloat(manualFreq);
    if (!isNaN(freq) && freq > 0) {
      tune(freq, manualMode);
      setManualFreq('');
    }
  };

  return (
    <div>
      <div className="page-header">
        <h2>📡 Ham Radio</h2>
        <p>Amateur radio receiver with multi-mode support</p>
      </div>

      <div className="grid-2" style={{ marginBottom: 24 }}>
        <div>
          {/* Frequency Display */}
          <div className="freq-display" style={{ marginBottom: 16 }}>
            <div className="freq-value">
              {status.frequency >= 1000
                ? status.frequency.toFixed(3)
                : status.frequency < 30
                ? status.frequency.toFixed(3)
                : status.frequency.toFixed(4)}
              <span className="freq-unit">MHz</span>
            </div>
            <div style={{ display: 'flex', justifyContent: 'center', gap: 12, marginTop: 12 }}>
              <span
                className="badge"
                style={{ background: 'rgba(168,85,247,0.15)', color: '#a855f7', fontSize: 14 }}
              >
                {status.mode}
              </span>
              <span className={`badge badge-${status.status === 'running' ? 'running' : 'stopped'}`}>
                <span className="badge-dot" />
                {status.status}
              </span>
            </div>
          </div>

          {/* Mode Selector */}
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="card-title" style={{ marginBottom: 12 }}>Mode</div>
            <div className="flex-row gap-8">
              {MODES.map((m) => (
                <button
                  key={m}
                  className={`btn ${status.mode === m ? 'btn-primary' : 'btn-outline'}`}
                  style={{ flex: 1 }}
                  onClick={() => tune(status.frequency, m)}
                  disabled={tuning}
                >
                  {m}
                </button>
              ))}
            </div>
          </div>

          {/* Audio Player */}
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="card-title" style={{ marginBottom: 12 }}>Audio Stream</div>
            <div className="audio-player">
              <audio controls src="/api/v1/ham/stream" style={{ width: '100%' }} />
            </div>
          </div>

          {/* Manual Tune */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Manual Tune</div>
            <div className="flex-row">
              <input
                className="form-input"
                type="number"
                step="0.001"
                placeholder="Freq (MHz)"
                value={manualFreq}
                onChange={(e) => setManualFreq(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleManualTune()}
                style={{ flex: 1 }}
              />
              <select
                className="form-select"
                value={manualMode}
                onChange={(e) => setManualMode(e.target.value as Mode)}
                style={{ width: 80 }}
              >
                {MODES.map((m) => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
              <button className="btn btn-primary" onClick={handleManualTune} disabled={tuning}>
                {tuning ? '...' : 'Tune'}
              </button>
            </div>
          </div>
        </div>

        <div>
          {/* Presets */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 16 }}>Band Presets</div>
            <div className="preset-grid">
              {PRESETS.map((p) => (
                <button
                  key={p.name}
                  className={`preset-btn ${
                    Math.abs(status.frequency - p.freq) < 0.001 ? 'active' : ''
                  }`}
                  onClick={() => tune(p.freq, p.mode)}
                  disabled={tuning}
                >
                  <span
                    style={{
                      fontSize: 10,
                      fontWeight: 600,
                      color: 'var(--primary)',
                      textTransform: 'uppercase',
                      letterSpacing: 0.5,
                    }}
                  >
                    {p.band}
                  </span>
                  <span className="preset-name">{p.name}</span>
                  <span className="preset-freq">
                    {p.freq < 30 ? p.freq.toFixed(3) : p.freq.toFixed(3)} MHz / {p.mode}
                  </span>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
