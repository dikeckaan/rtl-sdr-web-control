import { useState, useEffect, useRef } from 'react';
import { api, post } from '../api/client';

type Band = 'ALL' | 'GSM900' | 'DCS1800';
type SortKey = 'freq_mhz' | 'arfcn' | 'power_db' | 'operator';

interface ScanResult {
  freq_mhz: number;
  arfcn: number;
  band: string;
  power_db: number;
  operator: string;
}


const BANDS: Band[] = ['ALL', 'GSM900', 'DCS1800'];

export default function GSM() {
  const [band, setBand] = useState<Band>('ALL');
  const [results, setResults] = useState<ScanResult[]>([]);
  const [scanning, setScanning] = useState(false);
  const [lastScan, setLastScan] = useState('');
  const [sortKey, setSortKey] = useState<SortKey>('power_db');
  const [sortAsc, setSortAsc] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchResults = async () => {
    try {
      const res = await api<{ results: ScanResult[]; scanning: boolean; last_scan: string }>('/gsm/results');
      setResults(res.results ?? []);
      setScanning(res.scanning ?? false);
      if (res.last_scan) setLastScan(res.last_scan);
      return res.scanning;
    } catch {
      return false;
    }
  };


  useEffect(() => {
    // Sayfa yuklendiginde mevcut sonuclari ve durumu cek
    fetchResults();
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const startScan = async () => {
    setScanning(true);
    try {
      await post<{ started: boolean; band: string }>('/gsm/scan', { band });
      // Tarama basladiktan sonra sonuclari yokla
      pollRef.current = setInterval(async () => {
        const stillScanning = await fetchResults();
        if (!stillScanning && pollRef.current) {
          clearInterval(pollRef.current);
          pollRef.current = null;
        }
      }, 2000);
    } catch {
      // Tarama baslatilamazsa mevcut sonuclari cek
      await fetchResults();
      setScanning(false);
    }
  };

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortAsc(!sortAsc);
    } else {
      setSortKey(key);
      setSortAsc(false);
    }
  };

  const sorted = [...results].sort((a, b) => {
    const mul = sortAsc ? 1 : -1;
    if (sortKey === 'freq_mhz') return (a.freq_mhz - b.freq_mhz) * mul;
    if (sortKey === 'arfcn') return (a.arfcn - b.arfcn) * mul;
    if (sortKey === 'power_db') return (a.power_db - b.power_db) * mul;
    if (sortKey === 'operator') return a.operator.localeCompare(b.operator) * mul;
    return 0;
  });

  const maxPower = results.length > 0 ? Math.max(...results.map((r) => r.power_db)) : -30;
  const minPower = results.length > 0 ? Math.min(...results.map((r) => r.power_db)) : -120;
  const powerRange = Math.max(maxPower - minPower, 1);

  const sortIndicator = (key: SortKey) => {
    if (sortKey !== key) return '';
    return sortAsc ? ' ▲' : ' ▼';
  };

  return (
    <div>
      <div className="page-header">
        <h2>GSM Tarayici</h2>
        <p>Bolgenizdeki GSM baz istasyonlarini tarayin ve analiz edin</p>
      </div>

      {/* Tarama Kontrolleri */}
      <div className="card" style={{ marginBottom: 24 }}>
        <div className="flex-between" style={{ flexWrap: 'wrap', gap: 12 }}>
          <div className="flex-row gap-8">
            <span className="text-sm text-muted" style={{ fontWeight: 600 }}>Bant:</span>
            {BANDS.map((b) => (
              <button
                key={b}
                className={`btn btn-sm ${band === b ? 'btn-primary' : 'btn-outline'}`}
                onClick={() => setBand(b)}
                disabled={scanning}
              >
                {b}
              </button>
            ))}
          </div>
          <div className="flex-row gap-8">
            {scanning && (
              <span className="badge badge-starting">
                <span className="badge-dot" />
                Taraniyor...
              </span>
            )}
            {lastScan && !scanning && (
              <span className="text-sm text-muted">
                Son tarama: {new Date(lastScan).toLocaleString('tr-TR')}
              </span>
            )}
            <button
              className="btn btn-primary"
              onClick={startScan}
              disabled={scanning}
            >
              {scanning ? 'Taraniyor...' : 'Taramayi Baslat'}
            </button>
          </div>
        </div>
      </div>

      {/* Cubuk Grafigi */}
      {results.length > 0 && (
        <div className="section">
          <div className="section-title">Sinyal Gucu Genel Bakis</div>
          <div className="bar-chart">
            {sorted.map((r, i) => {
              const height = ((r.power_db - minPower) / powerRange) * 100;
              return (
                <div
                  key={i}
                  className="bar"
                  style={{
                    height: `${Math.max(height, 3)}%`,
                    background: r.power_db > -60 ? 'var(--success)' : r.power_db > -80 ? 'var(--primary)' : r.power_db > -100 ? 'var(--warning)' : 'var(--danger)',
                  }}
                  title={`${r.freq_mhz.toFixed(1)} MHz | ${r.power_db} dBm | ${r.operator}`}
                />
              );
            })}
          </div>
          <div className="flex-between text-sm text-muted" style={{ marginTop: 4 }}>
            <span>Siralama: {sortKey}</span>
            <span>{results.length} hucre bulundu</span>
          </div>
        </div>
      )}

      {/* Sonuc Tablosu */}
      <div className="section">
        <div className="section-title">Tarama Sonuclari</div>
        {results.length === 0 && !scanning ? (
          <div className="card text-center text-muted" style={{ padding: 40 }}>
            Henuz tarama sonucu yok. Bir bant secin ve Taramayi Baslat'a tiklayin.
          </div>
        ) : (
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th style={{ cursor: 'pointer' }} onClick={() => handleSort('freq_mhz')}>
                    Frekans (MHz){sortIndicator('freq_mhz')}
                  </th>
                  <th style={{ cursor: 'pointer' }} onClick={() => handleSort('arfcn')}>
                    ARFCN{sortIndicator('arfcn')}
                  </th>
                  <th>Bant</th>
                  <th style={{ cursor: 'pointer' }} onClick={() => handleSort('power_db')}>
                    Guc (dBm){sortIndicator('power_db')}
                  </th>
                  <th style={{ cursor: 'pointer' }} onClick={() => handleSort('operator')}>
                    Operator{sortIndicator('operator')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((r, i) => (
                  <tr key={i}>
                    <td className="font-mono">{r.freq_mhz.toFixed(1)}</td>
                    <td className="font-mono">{r.arfcn}</td>
                    <td>
                      <span
                        className="badge"
                        style={{
                          background: r.band === 'GSM900' ? 'rgba(59,130,246,0.12)' : 'rgba(168,85,247,0.12)',
                          color: r.band === 'GSM900' ? 'var(--primary)' : '#a855f7',
                        }}
                      >
                        {r.band}
                      </span>
                    </td>
                    <td>
                      <span
                        style={{
                          fontFamily: 'monospace',
                          fontWeight: 600,
                          color: r.power_db > -60 ? 'var(--success)' : r.power_db > -80 ? 'var(--primary)' : r.power_db > -100 ? 'var(--warning)' : 'var(--danger)',
                        }}
                      >
                        {r.power_db}
                      </span>
                    </td>
                    <td>{r.operator}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
