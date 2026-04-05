import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, post, clearToken } from '../api/client';
import type { SystemStatus } from '../api/client';

export default function Settings() {
  const navigate = useNavigate();
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [lang, setLang] = useState(() => localStorage.getItem('sdr_lang') || 'tr');

  useEffect(() => {
    api<SystemStatus>('/system/status')
      .then((res) => setStatus(res))
      .catch(() => {});
  }, []);

  const handleLanguage = (newLang: string) => {
    setLang(newLang);
    localStorage.setItem('sdr_lang', newLang);
  };

  const handleLogout = async () => {
    try {
      await post('/auth/logout');
    } catch {
      /* ignore */
    }
    clearToken();
    navigate('/login', { replace: true });
  };

  return (
    <div>
      <div className="page-header">
        <h2>Ayarlar</h2>
        <p>Sistem yapilandirmasi ve bilgileri</p>
      </div>

      <div className="grid-2" style={{ maxWidth: 900 }}>
        {/* Sistem Bilgisi */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            Sistem Bilgisi
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div className="flex-between">
              <span className="text-muted text-sm">Ana Bilgisayar Adi</span>
              <span className="font-mono">{status?.hostname ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">Calisma Suresi</span>
              <span className="font-mono">{status?.uptime_go ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">Bellek</span>
              <span className="font-mono">{status?.memory_mb ? `${status.memory_mb} MB` : '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">Goroutine</span>
              <span className="font-mono">{status?.goroutines ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">WS Istemcileri</span>
              <span className="font-mono">{status?.ws_clients ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">Aktif Hizmet</span>
              <span className="font-mono" style={{ color: 'var(--primary)' }}>
                {status?.active_service || 'Yok'}
              </span>
            </div>
          </div>
        </div>

        {/* Istasyon Bilgisi */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            Istasyon Bilgisi
          </div>
          {status?.station ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <div className="flex-between">
                <span className="text-muted text-sm">Ad</span>
                <span className="font-mono">{status.station.name}</span>
              </div>
              <div className="flex-between">
                <span className="text-muted text-sm">Enlem</span>
                <span className="font-mono">{status.station.lat.toFixed(6)}</span>
              </div>
              <div className="flex-between">
                <span className="text-muted text-sm">Boylam</span>
                <span className="font-mono">{status.station.lon.toFixed(6)}</span>
              </div>
              <div className="flex-between">
                <span className="text-muted text-sm">Yukseklik</span>
                <span className="font-mono">{status.station.alt} m</span>
              </div>
            </div>
          ) : (
            <div className="text-muted text-center" style={{ padding: 20 }}>
              Istasyon bilgisi mevcut degil
            </div>
          )}
        </div>

        {/* Dil */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            Dil
          </div>
          <div className="flex-row gap-8">
            <button
              className={`btn ${lang === 'tr' ? 'btn-primary' : 'btn-outline'}`}
              onClick={() => handleLanguage('tr')}
              style={{ flex: 1 }}
            >
              Turkce
            </button>
            <button
              className={`btn ${lang === 'en' ? 'btn-primary' : 'btn-outline'}`}
              onClick={() => handleLanguage('en')}
              style={{ flex: 1 }}
            >
              English
            </button>
          </div>
        </div>

        {/* Hesap */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            Hesap
          </div>
          <p className="text-sm text-muted" style={{ marginBottom: 16 }}>
            Hesabinizdan cikis yapin. Platforma erisim icin tekrar giris yapmaniz gerekecektir.
          </p>
          <button className="btn btn-danger" style={{ width: '100%' }} onClick={handleLogout}>
            Cikis Yap
          </button>
        </div>
      </div>
    </div>
  );
}
