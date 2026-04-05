import { useState, useEffect, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, post, setToken } from '../api/client';

interface AuthResponse {
  token: string;
}

interface SetupCheck {
  setup_required: boolean;
}

export default function Login() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [setupMode, setSetupMode] = useState(false);

  useEffect(() => {
    api<SetupCheck>('/auth/setup-check')
      .then((res) => {
        if (res.setup_required) setSetupMode(true);
      })
      .catch(() => {});
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      const endpoint = setupMode ? '/auth/setup' : '/auth/login';
      const res = await post<AuthResponse>(endpoint, { username, password });
      setToken(res.token);
      navigate('/', { replace: true });
    } catch {
      setError(setupMode ? 'Failed to create admin account.' : 'Invalid username or password.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-card">
        <div style={{ textAlign: 'center', fontSize: 48, marginBottom: 16 }}>📡</div>
        <h1>{setupMode ? 'Create Admin Account' : 'SDR Platform'}</h1>
        <p className="login-subtitle">
          {setupMode
            ? 'Set up the first administrator account to get started.'
            : 'Sign in to manage your SDR services.'}
        </p>

        {error && <div className="error-msg">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label" htmlFor="username">
              Username
            </label>
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
            <label className="form-label" htmlFor="password">
              Password
            </label>
            <input
              id="password"
              className="form-input"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="********"
              autoComplete="current-password"
              required
            />
          </div>
          <button
            className="btn btn-primary btn-lg"
            type="submit"
            disabled={submitting}
            style={{ width: '100%', marginTop: 8 }}
          >
            {submitting ? 'Please wait...' : setupMode ? 'Create Account' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}
