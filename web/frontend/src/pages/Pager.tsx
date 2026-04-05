import { useEffect, useState, useRef } from 'react';
import { api, post, connectWS } from '../api/client';
import type { WSMessage } from '../api/client';

interface PagerMessage {
  id: number;
  timestamp: string;
  msg_type: string;
  address: string;
  message: string;
  frequency: string;
}

interface PagerState {
  freq: string;
  status: string;
  message_count: number;
}

export default function Pager() {
  const [messages, setMessages] = useState<PagerMessage[]>([]);
  const [state, setState] = useState<PagerState>({ freq: '153.350M', status: 'stopped', message_count: 0 });
  const [newFreq, setNewFreq] = useState('');
  const feedRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    api<PagerState>('/pager/state').then(setState).catch(() => {});
    api<{ messages: PagerMessage[] }>('/pager/messages')
      .then((res) => setMessages((res.messages || []).reverse()))
      .catch(() => {});

    wsRef.current = connectWS(['pager.messages'], (msg: WSMessage) => {
      if (msg.topic === 'pager.messages') {
        const newMsg = msg.data as PagerMessage;
        setMessages((prev) => [newMsg, ...prev].slice(0, 500));
        setState((prev) => ({ ...prev, message_count: prev.message_count + 1 }));
      }
    });

    return () => { wsRef.current?.close(); };
  }, []);

  const changeFrequency = async () => {
    if (!newFreq) return;
    const freq = newFreq.includes('M') ? newFreq : newFreq + 'M';
    await post('/pager/freq', { freq });
    setState((prev) => ({ ...prev, freq }));
    setNewFreq('');
  };

  const formatTime = (ts: string) => {
    try {
      return new Date(ts).toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch { return ts; }
  };

  const isRunning = state.status === 'running';

  return (
    <div>
      <div className="page-header">
        <div className="flex-between">
          <div>
            <h2>📟 Pager Çözücü</h2>
            <p>POCSAG/FLEX çağrı cihazı mesaj çözücü</p>
          </div>
          <span className={`badge badge-${isRunning ? 'running' : 'stopped'}`}>
            <span className="badge-dot" />
            {isRunning ? 'Dinleniyor' : 'Durdu'}
          </span>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '300px 1fr', gap: 16, marginBottom: 24 }}>
        <div>
          <div className="freq-display" style={{ marginBottom: 16 }}>
            <div className="freq-value" style={{ fontSize: 32 }}>
              {state.freq.replace('M', '')}
              <span className="freq-unit">MHz</span>
            </div>
          </div>

          <div className="card" style={{ marginBottom: 16 }}>
            <div className="card-title" style={{ marginBottom: 12 }}>Frekans Değiştir</div>
            <div className="flex-row">
              <input
                className="form-input"
                placeholder="Örn: 153.350"
                value={newFreq}
                onChange={(e) => setNewFreq(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && changeFrequency()}
                style={{ flex: 1 }}
              />
              <button className="btn btn-primary" onClick={changeFrequency}>Ayarla</button>
            </div>
          </div>

          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>İstatistik</div>
            <div className="flex-between" style={{ marginBottom: 8 }}>
              <span className="text-muted text-sm">Alınan Mesaj</span>
              <span className="font-mono" style={{ fontWeight: 600 }}>{state.message_count}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">Tamponda</span>
              <span className="font-mono" style={{ fontWeight: 600 }}>{messages.length}</span>
            </div>
          </div>
        </div>

        <div>
          <div className="card" style={{ padding: 0 }}>
            <div style={{ padding: '16px 20px', borderBottom: '1px solid var(--border)' }}>
              <div className="flex-between">
                <div className="card-title">Canlı Mesaj Akışı</div>
                <span className="text-sm text-muted">{messages.length} mesaj</span>
              </div>
            </div>
            <div className="message-feed" ref={feedRef}>
              {messages.length === 0 ? (
                <div style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)' }}>
                  📟 Mesaj bekleniyor...
                </div>
              ) : messages.map((msg, i) => (
                <div className={`message-item ${i === 0 ? 'new' : ''}`} key={msg.id || i}>
                  <div className="flex-between" style={{ marginBottom: 4 }}>
                    <div className="flex-row gap-8">
                      <span className="message-time">{formatTime(msg.timestamp)}</span>
                      <span className="badge" style={{
                        background: 'rgba(59,130,246,0.12)', color: 'var(--primary)',
                        padding: '2px 6px', fontSize: 10,
                      }}>
                        {msg.msg_type}
                      </span>
                    </div>
                    <span className="font-mono text-sm text-muted">{msg.address}</span>
                  </div>
                  <div className="message-content">{msg.message}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
