package server

const builtinDashboard = `<!DOCTYPE html>
<html lang="tr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>SDR Yönetim Paneli</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{--bg:#0f172a;--card:#1e293b;--border:#334155;--primary:#3b82f6;--success:#22c55e;--danger:#ef4444;--warning:#f59e0b;--text:#e2e8f0;--muted:#94a3b8}
body{background:var(--bg);color:var(--text);font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;min-height:100vh}
.header{background:var(--card);border-bottom:1px solid var(--border);padding:1rem 2rem;display:flex;align-items:center;justify-content:space-between}
.header h1{font-size:1.25rem;font-weight:600}
.header .station{color:var(--muted);font-size:.875rem}
.container{max-width:1200px;margin:0 auto;padding:2rem}
.status-bar{display:flex;gap:1rem;margin-bottom:2rem;flex-wrap:wrap}
.status-item{background:var(--card);border:1px solid var(--border);border-radius:.5rem;padding:.75rem 1rem;font-size:.875rem}
.status-item .label{color:var(--muted);font-size:.75rem;text-transform:uppercase;letter-spacing:.05em}
.status-item .value{font-weight:600;margin-top:.25rem}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(300px,1fr));gap:1rem}
.card{background:var(--card);border:1px solid var(--border);border-radius:.75rem;padding:1.25rem;transition:all .2s}
.card:hover{border-color:var(--primary);transform:translateY(-2px)}
.card .top{display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:.75rem}
.card .name{font-weight:600;font-size:1rem}
.card .desc{color:var(--muted);font-size:.8rem;margin-bottom:1rem}
.badge{padding:.25rem .5rem;border-radius:.25rem;font-size:.7rem;font-weight:600;text-transform:uppercase}
.badge.running{background:rgba(34,197,94,.15);color:var(--success)}
.badge.stopped{background:rgba(148,163,184,.15);color:var(--muted)}
.badge.starting{background:rgba(245,158,11,.15);color:var(--warning)}
.badge.error{background:rgba(239,68,68,.15);color:var(--danger)}
.badge.docker{background:rgba(59,130,246,.1);color:var(--primary)}
.card .actions{display:flex;gap:.5rem}
.btn{padding:.5rem 1rem;border:none;border-radius:.375rem;font-size:.8rem;font-weight:500;cursor:pointer;transition:all .15s}
.btn-start{background:var(--success);color:#fff}
.btn-start:hover{filter:brightness(1.1)}
.btn-stop{background:var(--danger);color:#fff}
.btn-stop:hover{filter:brightness(1.1)}
.btn-open{background:var(--primary);color:#fff}
.btn-open:hover{filter:brightness(1.1)}
.btn:disabled{opacity:.5;cursor:not-allowed}
.btn-sm{padding:.375rem .75rem;font-size:.75rem}
.stop-all{background:var(--danger);color:#fff;padding:.5rem 1rem;border:none;border-radius:.375rem;cursor:pointer;font-weight:500}
.setup-form{max-width:400px;margin:4rem auto;background:var(--card);padding:2rem;border-radius:.75rem;border:1px solid var(--border)}
.setup-form h2{margin-bottom:1.5rem;text-align:center}
.setup-form input{width:100%;padding:.625rem;margin-bottom:1rem;background:var(--bg);border:1px solid var(--border);border-radius:.375rem;color:var(--text);font-size:.875rem}
.setup-form button{width:100%;padding:.625rem;background:var(--primary);color:#fff;border:none;border-radius:.375rem;cursor:pointer;font-weight:500}
.login-form{max-width:400px;margin:4rem auto;background:var(--card);padding:2rem;border-radius:.75rem;border:1px solid var(--border)}
.login-form h2{margin-bottom:1.5rem;text-align:center}
.login-form input{width:100%;padding:.625rem;margin-bottom:1rem;background:var(--bg);border:1px solid var(--border);border-radius:.375rem;color:var(--text);font-size:.875rem}
.login-form button{width:100%;padding:.625rem;background:var(--primary);color:#fff;border:none;border-radius:.375rem;cursor:pointer;font-weight:500}
.toast{position:fixed;bottom:2rem;right:2rem;padding:.75rem 1.25rem;border-radius:.5rem;font-size:.875rem;transform:translateY(100px);opacity:0;transition:all .3s}
.toast.show{transform:translateY(0);opacity:1}
.toast.success{background:var(--success);color:#fff}
.toast.error{background:var(--danger);color:#fff}
@media(max-width:640px){.container{padding:1rem}.grid{grid-template-columns:1fr}.header{padding:.75rem 1rem}}
</style>
</head>
<body>
<div id="app"></div>
<div id="toast" class="toast"></div>

<script>
const API = '/api/v1';
let token = localStorage.getItem('sdr_token') || '';
let refreshTimer = null;

async function api(path, opts = {}) {
  const headers = {'Content-Type': 'application/json'};
  if (token) headers['Authorization'] = 'Bearer ' + token;
  const res = await fetch(API + path, {...opts, headers});
  if (res.status === 401 && path !== '/auth/login' && path !== '/auth/setup') {
    token = '';
    localStorage.removeItem('sdr_token');
    render();
    return null;
  }
  return res.json();
}

function toast(msg, type = 'success') {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast ' + type + ' show';
  setTimeout(() => el.className = 'toast', 3000);
}

async function checkSetup() {
  const res = await api('/auth/me').catch(() => null);
  if (!res) {
    // Check if setup needed
    const setupCheck = await fetch(API + '/auth/setup', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({})}).then(r => r.json()).catch(() => null);
    if (setupCheck && setupCheck.error === 'username and password required') {
      return 'setup';
    }
    if (setupCheck && setupCheck.error === 'setup already completed') {
      return 'login';
    }
    return 'setup';
  }
  return 'dashboard';
}

async function render() {
  const state = await checkSetup();
  if (state === 'setup') renderSetup();
  else if (state === 'login' && !token) renderLogin();
  else renderDashboard();
}

function renderSetup() {
  document.getElementById('app').innerHTML = '<div class="setup-form"><h2>🛰️ SDR Kurulum</h2><p style="color:var(--muted);text-align:center;margin-bottom:1.5rem">Yönetici hesabı oluşturun</p><input id="su" placeholder="Kullanıcı adı" autocomplete="username"><input id="sp" type="password" placeholder="Şifre" autocomplete="new-password"><button onclick="doSetup()">Hesap Oluştur</button></div>';
}

async function doSetup() {
  const u = document.getElementById('su').value;
  const p = document.getElementById('sp').value;
  if (!u || !p) return toast('Kullanıcı adı ve şifre gerekli', 'error');
  const res = await api('/auth/setup', {method: 'POST', body: JSON.stringify({username: u, password: p})});
  if (res && res.status === 'setup complete') {
    toast('Hesap oluşturuldu');
    renderLogin();
  } else {
    toast(res?.error || 'Hata', 'error');
  }
}

function renderLogin() {
  document.getElementById('app').innerHTML = '<div class="login-form"><h2>🛰️ SDR Giriş</h2><input id="lu" placeholder="Kullanıcı adı" autocomplete="username"><input id="lp" type="password" placeholder="Şifre" autocomplete="current-password"><button onclick="doLogin()">Giriş Yap</button></div>';
}

async function doLogin() {
  const u = document.getElementById('lu').value;
  const p = document.getElementById('lp').value;
  const res = await api('/auth/login', {method: 'POST', body: JSON.stringify({username: u, password: p})});
  if (res && res.token) {
    token = res.token;
    localStorage.setItem('sdr_token', token);
    toast('Giriş başarılı');
    renderDashboard();
  } else {
    toast(res?.error || 'Giriş hatası', 'error');
  }
}

async function renderDashboard() {
  const [svcData, sysData] = await Promise.all([api('/services'), api('/system/status')]);
  if (!svcData) return;

  const svcs = svcData.services || [];
  const active = svcData.active_service || '';
  const sys = sysData || {};

  // Sort: native first, then docker; active service on top
  svcs.sort((a, b) => {
    if (a.id === active) return -1;
    if (b.id === active) return 1;
    if (a.category !== b.category) return a.category === 'native' ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  const icons = {radio:'📻',antenna:'📡',plane:'✈️','message-square':'📟',ship:'🚢',satellite:'🛰️',signal:'📶',container:'🐳'};

  let html = '<div class="header"><div><h1>🛰️ SDR Yönetim Paneli</h1><div class="station">' + (sys.station?.name || '') + ' · ' + (sys.station?.lat?.toFixed(4) || '') + '°N, ' + (sys.station?.lon?.toFixed(4) || '') + '°E</div></div><div style="display:flex;gap:.75rem;align-items:center"><button class="stop-all btn-sm" onclick="stopAll()">Tümünü Durdur</button><button class="btn btn-sm" style="background:var(--border)" onclick="doLogout()">Çıkış</button></div></div>';
  html += '<div class="container">';
  html += '<div class="status-bar">';
  html += '<div class="status-item"><div class="label">Aktif Servis</div><div class="value">' + (active || 'Yok') + '</div></div>';
  html += '<div class="status-item"><div class="label">Toplam Servis</div><div class="value">' + svcs.length + '</div></div>';
  html += '<div class="status-item"><div class="label">Bellek</div><div class="value">' + (sys.memory_mb || 0) + ' MB</div></div>';
  html += '<div class="status-item"><div class="label">WS İstemci</div><div class="value">' + (sys.ws_clients || 0) + '</div></div>';
  html += '</div>';
  html += '<div class="grid">';

  for (const svc of svcs) {
    const icon = icons[svc.icon] || '📡';
    const isRunning = svc.status === 'running';
    const isDocker = svc.category === 'docker';
    const statusClass = svc.status;

    html += '<div class="card">';
    html += '<div class="top"><div class="name">' + icon + ' ' + svc.name + '</div><div style="display:flex;gap:.375rem">';
    if (isDocker) html += '<span class="badge docker">Docker</span>';
    html += '<span class="badge ' + statusClass + '">' + statusLabel(svc.status) + '</span></div></div>';
    html += '<div class="desc">' + svc.description + '</div>';
    html += '<div class="actions">';
    if (isRunning) {
      html += '<button class="btn btn-stop btn-sm" onclick="stopSvc(\'' + svc.id + '\')">Durdur</button>';
      if (hasUI(svc)) html += '<button class="btn btn-open btn-sm" onclick="openSvc(\'' + svc.id + '\')">Aç</button>';
    } else {
      html += '<button class="btn btn-start btn-sm" onclick="startSvc(\'' + svc.id + '\')">Başlat</button>';
    }
    html += '</div></div>';
  }

  html += '</div></div>';
  document.getElementById('app').innerHTML = html;

  // Auto refresh
  if (refreshTimer) clearInterval(refreshTimer);
  refreshTimer = setInterval(renderDashboard, 10000);
}

function statusLabel(s) {
  return {running:'Çalışıyor',stopped:'Durdu',starting:'Başlıyor',stopping:'Duruyor',error:'Hata'}[s] || s;
}

function hasUI(svc) {
  return ['fmradio','hamradio','airband','pager','ais','iss','gsm'].includes(svc.id) || svc.category === 'docker';
}

function openSvc(id) {
  const nativePages = {fmradio:'/fm',hamradio:'/ham',airband:'/airband',pager:'/pager',ais:'/ais',iss:'/iss',gsm:'/gsm'};
  if (nativePages[id]) {
    window.location.href = nativePages[id];
  } else {
    window.open('http://' + window.location.hostname + ':8090', '_blank');
  }
}

async function startSvc(id) {
  toast('Başlatılıyor: ' + id);
  const res = await api('/services/' + id + '/start', {method: 'POST'});
  if (res?.error) toast(res.error, 'error');
  else toast('Başlatıldı: ' + id);
  renderDashboard();
}

async function stopSvc(id) {
  const res = await api('/services/' + id + '/stop', {method: 'POST'});
  if (res?.error) toast(res.error, 'error');
  else toast('Durduruldu: ' + id);
  renderDashboard();
}

async function stopAll() {
  await api('/services/stop-all', {method: 'POST'});
  toast('Tüm servisler durduruldu');
  renderDashboard();
}

async function doLogout() {
  await api('/auth/logout', {method: 'POST'});
  token = '';
  localStorage.removeItem('sdr_token');
  render();
}

render();
</script>
</body>
</html>`
