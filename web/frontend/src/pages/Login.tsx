import { useState, useEffect, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, post, setToken } from '../api/client';

export default function Login() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [setupMode, setSetupMode] = useState(false);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    api<{ setup_required: boolean }>('/auth/setup-check')
      .then((res) => {
        setSetupMode(res.setup_required);
      })
      .catch(() => {})
      .finally(() => setChecking(false));
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      if (setupMode) {
        const res = await post<{ token: string }>('/auth/setup', { username, password });
        if (res.token) {
          setToken(res.token);
          navigate('/', { replace: true });
        }
      } else {
        const res = await post<{ token: string }>('/auth/login', { username, password });
        if (res.token) {
          setToken(res.token);
          navigate('/', { replace: true });
        }
      }
    } catch {
      setError(setupMode ? 'Hesap oluşturulamadı.' : 'Kullanıcı adı veya şifre hatalı.');
    } finally {
      setSubmitting(false);
    }
  };

  if (checking) {
    return (
      <div className="login-page">
        <div style={{ color: 'var(--text-muted)', fontSize: 14 }}>Yükleniyor...</div>
      </div>
    );
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div style={{ textAlign: 'center', fontSize: 48, marginBottom: 8 }}>📡</div>
        <h1>{setupMode ? 'Yönetici Hesabı Oluştur' : 'SDR Platform'}</h1>
        <p className="login-subtitle">
          {setupMode
            ? 'Başlamak için bir yönetici hesabı oluşturun.'
            : 'SDR servislerinizi yönetmek için giriş yapın.'}
        </p>

        {error && <div className="error-msg">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label" htmlFor="username">Kullanıcı Adı</label>
            <input
              id="username"
              className="form-input"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="admin"
              autoComplete="username"
              required
            />
          </div>
          <div className="form-group">
            <label className="form-label" htmlFor="password">Şifre</label>
            <input
              id="password"
              className="form-input"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete={setupMode ? 'new-password' : 'current-password'}
              required
            />
          </div>
          <button
            className="btn btn-primary btn-lg"
            type="submit"
            disabled={submitting}
            style={{ width: '100%', marginTop: 8 }}
          >
            {submitting ? 'Lütfen bekleyin...' : setupMode ? 'Hesap Oluştur' : 'Giriş Yap'}
          </button>
        </form>
      </div>
    </div>
  );
}
