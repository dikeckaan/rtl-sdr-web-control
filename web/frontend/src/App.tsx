import { useState } from 'react';
import { BrowserRouter, Routes, Route, NavLink, Navigate, Outlet, useLocation } from 'react-router-dom';
import './App.css';
import './i18n';

import Dashboard from './pages/Dashboard';
import FMRadio from './pages/FMRadio';
import HamRadio from './pages/HamRadio';
import Airband from './pages/Airband';
import Pager from './pages/Pager';
import AIS from './pages/AIS';
import ISS from './pages/ISS';
import GSM from './pages/GSM';
import Login from './pages/Login';
import Settings from './pages/Settings';

function isAuthenticated(): boolean {
  return !!localStorage.getItem('sdr_token');
}

function ProtectedRoute() {
  const location = useLocation();
  if (!isAuthenticated()) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <Outlet />;
}

const navItems = [
  { to: '/', icon: '📊', label: 'Dashboard' },
  { to: '/fm', icon: '📻', label: 'FM Radio' },
  { to: '/ham', icon: '📡', label: 'Ham Radio' },
  { to: '/airband', icon: '✈️', label: 'Airband' },
  { to: '/pager', icon: '📟', label: 'Pager' },
  { to: '/ais', icon: '🚢', label: 'AIS' },
  { to: '/iss', icon: '🛰️', label: 'ISS Tracker' },
  { to: '/gsm', icon: '📶', label: 'GSM Scanner' },
  { to: '/settings', icon: '⚙️', label: 'Settings' },
];

function Layout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="app-layout">
      <button className="menu-toggle" onClick={() => setSidebarOpen(!sidebarOpen)}>
        {sidebarOpen ? '✕' : '☰'}
      </button>
      <div
        className={`sidebar-overlay ${sidebarOpen ? 'open' : ''}`}
        onClick={() => setSidebarOpen(false)}
      />
      <aside className={`sidebar ${sidebarOpen ? 'open' : ''}`}>
        <div className="sidebar-header">
          <span className="logo-icon">📡</span>
          <h1>SDR Platform</h1>
        </div>
        <nav className="sidebar-nav">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) => isActive ? 'active' : ''}
              onClick={() => setSidebarOpen(false)}
            >
              <span className="nav-icon">{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div style={{ padding: '8px 12px', fontSize: 11, color: 'var(--text-muted)' }}>
            SDR Management v1.0
          </div>
        </div>
      </aside>
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  );
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route element={<ProtectedRoute />}>
          <Route element={<Layout />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/fm" element={<FMRadio />} />
            <Route path="/ham" element={<HamRadio />} />
            <Route path="/airband" element={<Airband />} />
            <Route path="/pager" element={<Pager />} />
            <Route path="/ais" element={<AIS />} />
            <Route path="/iss" element={<ISS />} />
            <Route path="/gsm" element={<GSM />} />
            <Route path="/settings" element={<Settings />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
