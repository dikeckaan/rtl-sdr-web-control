import { useState } from 'react';
import { api, post } from '../api/client';

type Band = 'ALL' | 'GSM900' | 'DCS1800';
type SortKey = 'freq' | 'arfcn' | 'power' | 'operator';

interface ScanResult {
  freq: number;
  arfcn: number;
  band: string;
  power: number;
  operator: string;
  mcc?: string;
  mnc?: string;
}

const BANDS: Band[] = ['ALL', 'GSM900', 'DCS1800'];

export default function GSM() {
  const [band, setBand] = useState<Band>('ALL');
  const [results, setResults] = useState<ScanResult[]>([]);
  const [scanning, setScanning] = useState(false);
  const [sortKey, setSortKey] = useState<SortKey>('power');
  const [sortAsc, setSortAsc] = useState(false);

  const startScan = async () => {
    setScanning(true);
    setResults([]);
    try {
      const res = await post<{ results: ScanResult[] }>('/gsm/scan', { band });
      setResults(res.results ?? []);
    } catch {
      // try to get cached results
      try {
        const res = await api<{ results: ScanResult[] }>('/gsm/results');
        setResults(res.results ?? []);
      } catch {
        /* ignore */
      }
    } finally {
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
    if (sortKey === 'freq') return (a.freq - b.freq) * mul;
    if (sortKey === 'arfcn') return (a.arfcn - b.arfcn) * mul;
    if (sortKey === 'power') return (a.power - b.power) * mul;
    if (sortKey === 'operator') return a.operator.localeCompare(b.operator) * mul;
    return 0;
  });

  const maxPower = results.length > 0 ? Math.max(...results.map((r) => r.power)) : -30;
  const minPower = results.length > 0 ? Math.min(...results.map((r) => r.power)) : -120;
  const powerRange = Math.max(maxPower - minPower, 1);

  const sortIndicator = (key: SortKey) => {
    if (sortKey !== key) return '';
    return sortAsc ? ' ▲' : ' ▼';
  };

  return (
    <div>
      <div className="page-header">
        <h2>📶 GSM Scanner</h2>
        <p>Scan and analyze GSM base stations in your area</p>
      </div>

      {/* Scan Controls */}
      <div className="card" style={{ marginBottom: 24 }}>
        <div className="flex-between" style={{ flexWrap: 'wrap', gap: 12 }}>
          <div className="flex-row gap-8">
            <span className="text-sm text-muted" style={{ fontWeight: 600 }}>Band:</span>
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
                Scanning...
              </span>
            )}
            <button
              className="btn btn-primary"
              onClick={startScan}
              disabled={scanning}
            >
              {scanning ? 'Scanning...' : 'Start Scan'}
            </button>
          </div>
        </div>
      </div>

      {/* Bar Chart */}
      {results.length > 0 && (
        <div className="section">
          <div className="section-title">Signal Strength Overview</div>
          <div className="bar-chart">
            {sorted.map((r, i) => {
              const height = ((r.power - minPower) / powerRange) * 100;
              return (
                <div
                  key={i}
                  className="bar"
                  style={{
                    height: `${Math.max(height, 3)}%`,
                    background: r.power > -60 ? 'var(--success)' : r.power > -80 ? 'var(--primary)' : r.power > -100 ? 'var(--warning)' : 'var(--danger)',
                  }}
                  title={`${r.freq.toFixed(1)} MHz | ${r.power} dBm | ${r.operator}`}
                />
              );
            })}
          </div>
          <div className="flex-between text-sm text-muted" style={{ marginTop: 4 }}>
            <span>Sorted by {sortKey}</span>
            <span>{results.length} cells found</span>
          </div>
        </div>
      )}

      {/* Results Table */}
      <div className="section">
        <div className="section-title">Scan Results</div>
        {results.length === 0 && !scanning ? (
          <div className="card text-center text-muted" style={{ padding: 40 }}>
            No scan results yet. Select a band and click Start Scan.
          </div>
        ) : (
          <div className="table-container">
            <table>
              <thead>
                <tr>
                  <th onClick={() => handleSort('freq')}>
                    Frequency{sortIndicator('freq')}
                  </th>
                  <th onClick={() => handleSort('arfcn')}>
                    ARFCN{sortIndicator('arfcn')}
                  </th>
                  <th>Band</th>
                  <th onClick={() => handleSort('power')}>
                    Power (dBm){sortIndicator('power')}
                  </th>
                  <th onClick={() => handleSort('operator')}>
                    Operator{sortIndicator('operator')}
                  </th>
                  <th>MCC/MNC</th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((r, i) => (
                  <tr key={i}>
                    <td className="font-mono">{r.freq.toFixed(1)}</td>
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
                          color: r.power > -60 ? 'var(--success)' : r.power > -80 ? 'var(--primary)' : r.power > -100 ? 'var(--warning)' : 'var(--danger)',
                        }}
                      >
                        {r.power}
                      </span>
                    </td>
                    <td>{r.operator}</td>
                    <td className="font-mono text-muted">
                      {r.mcc && r.mnc ? `${r.mcc}/${r.mnc}` : '--'}
                    </td>
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
