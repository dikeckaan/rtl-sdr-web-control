import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { api, clearToken } from '../api/client';
import type { SystemStatus } from '../api/client';

export default function Settings() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const [status, setStatus] = useState<SystemStatus | null>(null);

  useEffect(() => {
    api<SystemStatus>('/system/status')
      .then((res) => setStatus(res))
      .catch(() => {});
  }, []);

  const handleLanguage = (lang: string) => {
    i18n.changeLanguage(lang);
    localStorage.setItem('sdr_lang', lang);
  };

  const handleLogout = () => {
    clearToken();
    navigate('/login', { replace: true });
  };

  return (
    <div>
      <div className="page-header">
        <h2>{t('settings', 'Settings')}</h2>
        <p>{t('settings_desc', 'System configuration and information')}</p>
      </div>

      <div className="grid-2" style={{ maxWidth: 900 }}>
        {/* System Info */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            {t('system_info', 'System Information')}
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div className="flex-between">
              <span className="text-muted text-sm">{t('hostname', 'Hostname')}</span>
              <span className="font-mono">{status?.hostname ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">{t('uptime', 'Uptime')}</span>
              <span className="font-mono">{status?.uptime_go ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">{t('memory', 'Memory')}</span>
              <span className="font-mono">{status?.memory_mb ? `${status.memory_mb} MB` : '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">{t('goroutines', 'Goroutines')}</span>
              <span className="font-mono">{status?.goroutines ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">{t('ws_clients', 'WS Clients')}</span>
              <span className="font-mono">{status?.ws_clients ?? '--'}</span>
            </div>
            <div className="flex-between">
              <span className="text-muted text-sm">{t('active_service', 'Active Service')}</span>
              <span className="font-mono" style={{ color: 'var(--primary)' }}>
                {status?.active_service || 'None'}
              </span>
            </div>
          </div>
        </div>

        {/* Station Info */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            {t('station_info', 'Station Information')}
          </div>
          {status?.station ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <div className="flex-between">
                <span className="text-muted text-sm">{t('station_name', 'Name')}</span>
                <span className="font-mono">{status.station.name}</span>
              </div>
              <div className="flex-between">
                <span className="text-muted text-sm">{t('latitude', 'Latitude')}</span>
                <span className="font-mono">{status.station.lat.toFixed(6)}</span>
              </div>
              <div className="flex-between">
                <span className="text-muted text-sm">{t('longitude', 'Longitude')}</span>
                <span className="font-mono">{status.station.lon.toFixed(6)}</span>
              </div>
              <div className="flex-between">
                <span className="text-muted text-sm">{t('altitude', 'Altitude')}</span>
                <span className="font-mono">{status.station.alt} m</span>
              </div>
            </div>
          ) : (
            <div className="text-muted text-center" style={{ padding: 20 }}>
              {t('no_station', 'No station info available')}
            </div>
          )}
        </div>

        {/* Language */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            {t('language', 'Language')}
          </div>
          <div className="flex-row gap-8">
            <button
              className={`btn ${i18n.language === 'tr' ? 'btn-primary' : 'btn-outline'}`}
              onClick={() => handleLanguage('tr')}
              style={{ flex: 1 }}
            >
              🇹🇷 Turkce
            </button>
            <button
              className={`btn ${i18n.language === 'en' ? 'btn-primary' : 'btn-outline'}`}
              onClick={() => handleLanguage('en')}
              style={{ flex: 1 }}
            >
              🇬🇧 English
            </button>
          </div>
        </div>

        {/* Account */}
        <div className="card">
          <div className="card-title" style={{ marginBottom: 16 }}>
            {t('account', 'Account')}
          </div>
          <p className="text-sm text-muted" style={{ marginBottom: 16 }}>
            {t('logout_desc', 'Sign out of your account. You will need to sign in again to access the platform.')}
          </p>
          <button className="btn btn-danger" style={{ width: '100%' }} onClick={handleLogout}>
            {t('logout', 'Logout')}
          </button>
        </div>
      </div>
    </div>
  );
}
