import { useEffect, useState } from 'react';
import { api, post } from '../api/client';

interface HamState {
  freq: string;
  mode: string;
  status: string;
  presets: Array<{ name: string; freq: string; mode: string; band: string }>;
  modes: string[];
}

export default function HamRadio() {
  const [state, setState] = useState<HamState>({
    freq: '145.500M', mode: 'fm', status: 'stopped', presets: [], modes: ['fm', 'am', 'usb', 'lsb'],
  });
  const [manualFreq, setManualFreq] = useState('');
  const [selectedMode, setSelectedMode] = useState('fm');
  const [tuning, setTuning] = useState(false);

  const fetchState = async () => {
    try {
      const res = await api<HamState>('/ham/state');
      setState(res);
      setSelectedMode(res.mode);
    } catch { /* */ }
  };

  useEffect(() => {
    fetchState();
    const interval = setInterval(fetchState, 5000);
    return () => clearInterval(interval);
  }, []);

  const tune = async (freq: string, mode: string) => {
    setTuning(true);
    try {
      await post('/ham/tune', { freq, mode });
      await fetchState();
    } catch { /* */ }
    finally { setTuning(false); }
  };

  const freqDisplay = state.freq.replace('M', '');
  const isRunning = state.status === 'running';

  return (
    <div>
      <div className="page-header">
        <h2>📡 Amatör Radyo</h2>
        <p>VHF/UHF amatör radyo alıcısı</p>
      </div>

      <div className="grid-2" style={{ marginBottom: 24 }}>
        <div>
          <div className="freq-display" style={{ marginBottom: 16 }}>
            <div className="freq-value">
              {freqDisplay}
              <span className="freq-unit">MHz</span>
            </div>
            <div style={{ marginTop: 8, display: 'flex', justifyContent: 'center', gap: 8 }}>
              <span className={`badge badge-${isRunning ? 'running' : 'stopped'}`}>
                <span className="badge-dot" />
                {isRunning ? 'Dinleniyor' : 'Durdu'}
              </span>
              <span className="badge" style={{ background: 'rgba(59,130,246,0.15)', color: 'var(--primary)' }}>
                {state.mode.toUpperCase()}
              </span>
            </div>
          </div>

          <div className="card" style={{ marginBottom: 16 }}>
            <div className="card-title" style={{ marginBottom: 12 }}>🔊 Ses Akışı</div>
            <div className="audio-player">
              <audio controls src={isRunning ? '/api/v1/ham/stream' : undefined} style={{ width: '100%' }} />
            </div>
          </div>

          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Manuel Ayarlama</div>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
              {(state.modes || ['fm', 'am', 'usb', 'lsb']).map((m) => (
                <button
                  key={m}
                  className={`btn btn-sm ${selectedMode === m ? 'btn-primary' : 'btn-outline'}`}
                  onClick={() => setSelectedMode(m)}
                >
                  {m.toUpperCase()}
                </button>
              ))}
            </div>
            <div className="flex-row">
              <input
                className="form-input"
                placeholder="Örn: 145.500"
                value={manualFreq}
                onChange={(e) => setManualFreq(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && tune(manualFreq + 'M', selectedMode)}
                style={{ flex: 1 }}
              />
              <button className="btn btn-primary" onClick={() => tune(manualFreq + 'M', selectedMode)} disabled={tuning}>
                {tuning ? '...' : 'Ayarla'}
              </button>
            </div>
          </div>
        </div>

        <div>
          <div className="card">
            <div className="card-title" style={{ marginBottom: 16 }}>📡 Hazır Frekanslar</div>
            <div className="preset-grid">
              {(state.presets || []).map((p) => (
                <button
                  key={p.name}
                  className={`preset-btn ${state.freq === p.freq && state.mode === p.mode ? 'active' : ''}`}
                  onClick={() => tune(p.freq, p.mode)}
                  disabled={tuning}
                >
                  <span className="preset-name">{p.name}</span>
                  <span className="preset-freq">{p.freq.replace('M', '')} · {p.mode.toUpperCase()}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
