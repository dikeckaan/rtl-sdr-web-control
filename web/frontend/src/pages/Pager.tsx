import { useEffect, useState, useRef } from 'react';
import { api, post, connectWS } from '../api/client';
import type { WSMessage } from '../api/client';

interface PagerMessage {
  id: string;
  timestamp: string;
  type: string;
  address: string;
  content: string;
}

interface PagerStatus {
  frequency: number;
  status: string;
  message_count: number;
}

export default function Pager() {
  const [messages, setMessages] = useState<PagerMessage[]>([]);
  const [status, setStatus] = useState<PagerStatus>({ frequency: 153.350, status: 'stopped', message_count: 0 });
  const [newFreq, setNewFreq] = useState('');
  const feedRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const fetchStatus = async () => {
    try {
      const res = await api<PagerStatus>('/pager/status');
      setStatus(res);
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    fetchStatus();

    // Fetch initial messages
    api<{ messages: PagerMessage[] }>('/pager/messages')
      .then((res) => setMessages(res.messages ?? []))
      .catch(() => {});

    // WebSocket for real-time
    wsRef.current = connectWS(['pager.messages'], (msg: WSMessage) => {
      if (msg.topic === 'pager.messages') {
        const newMsg = msg.data as PagerMessage;
        setMessages((prev) => [newMsg, ...prev].slice(0, 500));
        setStatus((prev) => ({ ...prev, message_count: prev.message_count + 1 }));
      }
    });

    return () => {
      wsRef.current?.close();
    };
  }, []);

  useEffect(() => {
    if (feedRef.current) {
      feedRef.current.scrollTop = 0;
    }
  }, [messages]);

  const changeFrequency = async () => {
    const freq = parseFloat(newFreq);
    if (!isNaN(freq) && freq > 0) {
      try {
        await post('/pager/frequency', { frequency: freq });
        setStatus((prev) => ({ ...prev, frequency: freq }));
        setNewFreq('');
      } catch {
        /* ignore */
      }
    }
  };

  const formatTime = (ts: string) => {
    try {
      const d = new Date(ts);
      return d.toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch {
      return ts;
    }
  };

  return (
    <div>
      <div className="page-header">
        <div className="flex-between">
          <div>
            <h2>📟 Pager Decoder</h2>
            <p>POCSAG/FLEX pager message decoder</p>
          </div>
          <span className={`badge badge-${status.status === 'running' ? 'running' : 'stopped'}`}>
            <span className="badge-dot" />
            {status.status}
          </span>
        </div>
      </div>

      <div className="grid-2" style={{ marginBottom: 24, gridTemplateColumns: '1fr 2fr' }}>
        {/* Left Panel */}
        <div>
          {/* Frequency */}
          <div className="freq-display" style={{ marginBottom: 16 }}>
            <div className="freq-value" style={{ fontSize: 36 }}>
              {status.frequency.toFixed(3)}
              <span className="freq-unit">MHz</span>
            </div>
          </div>

          {/* Change Frequency */}
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="card-title" style={{ marginBottom: 12 }}>Change Frequency</div>
            <div className="flex-row">
              <input
                className="form-input"
                type="number"
                step="0.001"
                placeholder="MHz"
                value={newFreq}
                onChange={(e) => setNewFreq(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && changeFrequency()}
                style={{ flex: 1 }}
              />
              <button className="btn btn-primary" onClick={changeFrequency}>
                Set
              </button>
            </div>
          </div>

          {/* Stats */}
          <div className="card">
            <div className="card-title" style={{ marginBottom: 12 }}>Statistics</div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <div className="flex-between">
                <span className="text-muted text-sm">Messages Received</span>
                <span className="font-mono" style={{ fontWeight: 600 }}>
                  {status.message_count}
                </span>
              </div>
              <div className="flex-between">
                <span className="text-muted text-sm">Buffer Size</span>
                <span className="font-mono" style={{ fontWeight: 600 }}>
                  {messages.length}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Right Panel - Message Feed */}
        <div>
          <div className="card" style={{ padding: 0 }}>
            <div style={{ padding: '16px 20px', borderBottom: '1px solid var(--border)' }}>
              <div className="flex-between">
                <div className="card-title">Live Message Feed</div>
                <span className="text-sm text-muted">{messages.length} messages</span>
              </div>
            </div>
            <div className="message-feed" ref={feedRef}>
              {messages.length === 0 ? (
                <div style={{ padding: 40, textAlign: 'center', color: 'var(--text-muted)' }}>
                  Waiting for messages...
                </div>
              ) : (
                messages.map((msg, i) => (
                  <div className={`message-item ${i === 0 ? 'new' : ''}`} key={msg.id || i}>
                    <div className="flex-between" style={{ marginBottom: 4 }}>
                      <div className="flex-row gap-8">
                        <span className="message-time">{formatTime(msg.timestamp)}</span>
                        <span
                          className="badge"
                          style={{
                            background: 'rgba(59,130,246,0.12)',
                            color: 'var(--primary)',
                            padding: '2px 6px',
                            fontSize: 10,
                          }}
                        >
                          {msg.type}
                        </span>
                      </div>
                      <span className="font-mono text-sm text-muted">{msg.address}</span>
                    </div>
                    <div className="message-content">{msg.content}</div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
