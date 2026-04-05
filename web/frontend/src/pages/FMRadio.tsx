import { useEffect, useState } from 'react';
import { api, post } from '../api/client';

interface FMState {
  freq: string;
  status: string;
  presets: Array<{ name: string; freq: string }>;
}

export default function FMRadio() {
  const [state, setState] = useState<FMState>({ freq: '93.0M', status: 'stopped', presets: [] });
  const [manualFreq, setManualFreq] = useState('');
  const [tuning, setTuning] = useState(false);

  const fetchState = async () => {
    try {
      const res = await api<FMState>('/fm/state');
      setState(res);
    } catch { /* */ }
  };

  useEffect(() => {
    fetchState();
    const interval = setInterval(fetchState, 5000);
    return () => clearInterval(interval);
  }, []);

  const tune = async (freq: string) => {
    setTuning(true);
    try {
      const res = await post<{ freq: string; status: string }>('/fm/tune', { freq });
      setState((prev) => ({ ...prev, freq: res.freq || freq, status: res.status || prev.status }));
    } catch { /* */ }
    finally { setTuning(false); }
  };

  const handleManualTune = () => {
    const f = parseFloat(manualFreq);
    if (f >= 87.5 && f <= 108.0) {
      tune(f.toFixed(1) + 'M');
      setManualFreq('');
    }
  };

  const freqDisplay = state.freq.replace('M', '');
  const isRunning = state.status === 'running';

  return (
    <div>
      <div className="page-header">
        <h2>📻 FM Radyo</h2>
        <p>FM radyo istasyonlarını dinleyin</p>
      </div>

      <div className="grid-2" style={{ marginBottom: 24 }}>
        <div>
          {/* Frekans Göstergesi */}
          <div className="freq-display" style={{ marginBottom: 16 }}>
            <div className="freq-value">
              {freqDisplay}
              <span className="freq-unit">MHz</span>
            </div>
            <div style={{ marginTop: 8 }}>
              <span className={`badge badge-${isRunning ? 'running' : 'stopped'}`}>
                <span className="badge-dot" />
                {isRunning ? 'Yayında' : 'Durdu'}
              </span>
            </div>
          </div>

          {/* Ses Oynatıcı */}
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="card-title" style={{ marginBottom: 12 }}>🔊 Ses Akışı</div>
            <div className="audio-player">
              <audio
                controls
                src={isRunning ? '/api/v1/fm/stream' : undefined}
                style={{ width: '100%' }}
              />
            </div>
            {!isRunning && (
              <div style={{ color: 'var(--text-muted)', fontSize: 13, marginTop: 8, textAlign: 'center' }}>
                Ses akışı için servisi başlatın
              </div>
            )}
          </div>

          {/* Manuel Ayarlama */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Manuel Frekans</div>
            <div className="flex-row">
              <input
                className="form-input"
                type="number"
                min="87.5"
                max="108.0"
                step="0.1"
                placeholder="Örn: 93.3"
                value={manualFreq}
                onChange={(e) => setManualFreq(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleManualTune()}
                style={{ flex: 1 }}
              />
              <button className="btn btn-primary" onClick={handleManualTune} disabled={tuning}>
                {tuning ? 'Ayarlanıyor...' : 'Ayarla'}
              </button>
            </div>
          </div>
        </div>

        <div>
          {/* Preset'ler */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 16 }}>📻 Hazır İstasyonlar</div>
            <div className="preset-grid">
              {(state.presets || []).map((p) => (
                <button
                  key={p.name}
                  className={`preset-btn ${state.freq === p.freq ? 'active' : ''}`}
                  onClick={() => tune(p.freq)}
                  disabled={tuning}
                >
                  <span className="preset-name">{p.name}</span>
                  <span className="preset-freq">{p.freq.replace('M', '')} MHz</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
