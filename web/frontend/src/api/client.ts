const API_BASE = '/api/v1';

function getToken(): string {
  return localStorage.getItem('sdr_token') || '';
}

export function setToken(token: string) {
  localStorage.setItem('sdr_token', token);
}

export function clearToken() {
  localStorage.removeItem('sdr_token');
}

export async function api<T = unknown>(path: string, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(opts.headers as Record<string, string>),
  };
  const token = getToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, { ...opts, headers });

  if (res.status === 401) {
    clearToken();
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  return res.json();
}

export async function post<T = unknown>(path: string, body?: unknown): Promise<T> {
  return api(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined });
}

// WebSocket connection helper
export function connectWS(topics: string[], onMessage: (msg: WSMessage) => void): WebSocket {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const token = getToken();
  const url = `${proto}//${window.location.host}/ws?topics=${topics.join(',')}&token=${token}`;
  const ws = new WebSocket(url);

  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data);
      onMessage(msg);
    } catch {}
  };

  ws.onclose = () => {
    setTimeout(() => {
      const newWs = connectWS(topics, onMessage);
      Object.assign(ws, newWs);
    }, 3000);
  };

  return ws;
}

export interface WSMessage {
  topic: string;
  data: unknown;
}

export interface ServiceInfo {
  id: string;
  name: string;
  description: string;
  category: 'native' | 'docker';
  status: 'stopped' | 'starting' | 'running' | 'stopping' | 'error';
  icon: string;
}

export interface ServicesResponse {
  services: ServiceInfo[];
  active_service: string;
}

export interface SystemStatus {
  hostname: string;
  active_service: string;
  uptime_go: string;
  goroutines: number;
  memory_mb: number;
  ws_clients: number;
  station: { name: string; lat: number; lon: number; alt: number };
}
