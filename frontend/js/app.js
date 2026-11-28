// app.js - Alumni NTUST Indonesia SPA
// Konfigurasi
const hostname = window.location.hostname;
const isLocalStatic = window.location.protocol === 'file:' ||
    ((hostname === 'localhost' || hostname === '127.0.0.1') && window.location.port !== '8080');
const API_BASE = isLocalStatic ? 'http://localhost:8080/api' : '/api';
const HEALTH_URL = isLocalStatic ? 'http://localhost:8080/health' : '/health';

const WILAYAH_LIST = [
    'Jabodetabek', 'Jakarta', 'Bandung', 'Surabaya', 'Medan',
    'Yogyakarta', 'Semarang', 'Makassar', 'Denpasar', 'Balikpapan', 'Lainnya'
];
const MAX_PHOTO_BYTES = 2 * 1024 * 1024;

// State
let currentUser = null;
let token = localStorage.getItem('token');
let pendingFotoProfil = '';
let dirState = {
    q: '',
    wilayah: '',
    jurusan: '',
    sort: 'nama_asc',
    page: 1,
    limit: 12,
    total: 0
};

// XSS & URL safety helpers
function esc(str) {
    return String(str ?? '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#x27;');
}

function safeUrl(url) {
    const raw = String(url ?? '').trim();
    if (!raw) return '';
    try {
        const parsed = new URL(raw);
        if (parsed.protocol === 'http:' || parsed.protocol === 'https:') {
            return parsed.href;
        }
    } catch (_) {
        return '';
    }
    return '';
}

function safeImageDataUrl(val) {
    const raw = String(val ?? '').trim();
    if (raw.startsWith('data:image/') && !raw.startsWith('data:image/svg')) {
        return raw;
    }
    return '';
}

// Route helpers
function parseRoute() {
    const raw = (window.location.hash || '').replace(/^#/, '');
    const qIndex = raw.indexOf('?');
    const pathPart = qIndex >= 0 ? raw.slice(0, qIndex) : raw;
    const queryPart = qIndex >= 0 ? raw.slice(qIndex + 1) : '';
    let path = pathPart || '/';
    if (!path.startsWith('/')) {
        path = '/' + path;
    }
    return { path, params: new URLSearchParams(queryPart) };
}

function navigate(path) {
    const normalized = path.startsWith('#') ? path : '#' + (path.startsWith('/') ? path : '/' + path);
    if (window.location.hash === normalized) {
        renderRoute();
    } else {
        window.location.hash = normalized;
    }
}

// API helper
async function apiFetch(path, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        ...(options.headers || {})
    };
    if (token) {
        headers['Authorization'] = 'Bearer ' + token;
    }
    const res = await fetch(API_BASE + path, {
        ...options,
        headers
    });
    if (res.status === 401 && path !== '/login' && path !== '/verify') {
        token = null;
        currentUser = null;
        localStorage.removeItem('token');
        updateNav();
        const { path: curPath } = parseRoute();
        if (curPath !== '/login' && path !== '/me') {
            navigate('/login');
        }
    }
    return res;
}

function showAlert(message, type) {
    const app = document.getElementById('app');
    if (!app) return;
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${esc(type)}`;
    alertDiv.textContent = message;
    app.prepend(alertDiv);
    setTimeout(() => alertDiv.remove(), 5000);
}

// Health check polling
async function checkHealth() {
    const el = document.getElementById('healthIndicator');
    if (!el) return;
    try {
        const res = await fetch(HEALTH_URL);
        if (res.ok) {
            const data = await res.json();
            const latency = data?.couchdb?.latency_ms;
            el.className = 'health-indicator online';
            el.textContent = latency !== undefined ? `Online (${latency}ms)` : 'Online';
        } else {
            el.className = 'health-indicator offline';
            el.textContent = 'Offline';
        }
    } catch (_) {
        el.className = 'health-indicator offline';
        el.textContent = 'Offline';
    }
}

// Navigation
function updateNav() {
    const navAuth = document.getElementById('navAuth');
    const navLinks = document.getElementById('navLinks');
    if (!navAuth || !navLinks) return;

    navAuth.innerHTML = '';
    navLinks.innerHTML = '';

    const addNavLink = (container, text, href) => {
        const a = document.createElement('a');
        a.textContent = text;
        a.href = href;
        container.appendChild(a);
    };

    if (currentUser) {
        addNavLink(navLinks, 'Direktori', '#/directory');
        addNavLink(navLinks, 'Forum', '#/forum');
        addNavLink(navLinks, 'Profil', '#/profile');
        if (currentUser.status !== 'verified') {
            addNavLink(navLinks, 'Verifikasi', '#/verify');
        }
        addNavLink(navLinks, 'Subgroup', '#/subgroups');

        const logoutBtn = document.createElement('button');
        logoutBtn.id = 'logoutBtn';
        logoutBtn.textContent = 'Keluar';
        logoutBtn.addEventListener('click', logout);
        navAuth.appendChild(logoutBtn);
    } else {
        addNavLink(navLinks, 'Forum', '#/forum');
        addNavLink(navAuth, 'Login', '#/login');
        addNavLink(navAuth, 'Daftar', '#/register');
    }
}

async function initUser() {
    if (token) {
        try {
            const res = await apiFetch('/me');
            if (res.ok) {
                const payload = await res.json();
                currentUser = payload.data || null;
            } else {
                token = null;
                currentUser = null;
                localStorage.removeItem('token');
            }
        } catch (_) {
            token = null;
            currentUser = null;
            localStorage.removeItem('token');
        }
    } else {
        currentUser = null;
    }

    updateNav();
    renderRoute();
}

function renderRoute() {
    const { path, params } = parseRoute();
    const app = document.getElementById('app');
    if (!app) return;
    app.innerHTML = '';

    switch (path) {
        case '/register':
            if (currentUser) { navigate('/profile'); return; }
            renderRegister();
            break;
        case '/login':
            if (currentUser) { navigate('/profile'); return; }
            renderLogin();
            break;
        case '/verify':
            if (params && params.get('token')) {
                renderVerifyConfirmPage(params);
            } else {
                renderVerifyPage(params);
            }
            break;
        case '/verify/confirm':
            renderVerifyConfirmPage(params);
            break;
        case '/profile':
            if (!currentUser) { navigate('/login'); return; }
            renderProfile();
            break;
        case '/directory':
            renderDirectory();
            break;
        case '/forum':
            renderForum();
            break;
        case '/subgroups':
            renderSubgroups();
            break;
        default:
            renderHome();
    }
}

function logout() {
    token = null;
    localStorage.removeItem('token');
    currentUser = null;
    updateNav();
    navigate('/');
}

// Page renderers
function renderHome() {
    const app = document.getElementById('app');
    const actions = currentUser
        ? `
            <a href="#/profile" class="btn btn-primary">Profil Saya</a>
            <a href="#/directory" class="btn btn-primary">Lihat Direktori</a>
            <a href="#/forum" class="btn btn-secondary">Buka Forum</a>
        `
        : `
            <a href="#/register" class="btn btn-primary">Daftar Alumni</a>
            <a href="#/directory" class="btn btn-primary">Lihat Direktori</a>
            <a href="#/login" class="btn btn-secondary">Login</a>
        `;

    app.innerHTML = `
        <div class="card" style="text-align: center; padding: 60px 20px;">
            <h1>Alumni NTUST Indonesia</h1>
            <p style="margin: 20px 0; font-size: 18px;">Jaringan Alumni National Taiwan University of Science and Technology (NTUST) di Indonesia</p>
            <div style="display: flex; gap: 12px; justify-content: center; flex-wrap: wrap;">
                ${actions}
            </div>
        </div>
    `;
}

function renderLogin() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Login</h2>
            <p class="subtitle">Gunakan email dan password Anda</p>
            <div id="loginError"></div>
            <form id="loginForm">
                <div class="form-group">
                    <label for="loginEmail">Email</label>
                    <input type="email" id="loginEmail" name="email" required placeholder="nama@domain.com">
                </div>
                <div class="form-group">
                    <label for="loginPassword">Password</label>
                    <input type="password" id="loginPassword" name="password" required placeholder="******">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Login</button>
            </form>
            <p style="margin-top: 20px; text-align: center; font-size: 14px;">
                Belum punya akun? <a href="#/register" class="btn-link">Daftar sebagai alumni baru</a>
            </p>
            <p style="margin-top: 10px; text-align: center;">
                <a href="#/verify" class="btn-link">Ingin memverifikasi akun?</a>
            </p>
        </div>
    `;
    document.getElementById('loginForm').addEventListener('submit', handleLogin);
}

async function handleLogin(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const email = String(form.get('email') || '').trim();
    const password = String(form.get('password') || '');
    const errorEl = document.getElementById('loginError');
    errorEl.innerHTML = '';

    try {
        const res = await apiFetch('/login', {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
        const data = await res.json();

        if (res.ok) {
            token = data.token;
            localStorage.setItem('token', token);
            const meRes = await apiFetch('/me');
            if (meRes.ok) {
                currentUser = (await meRes.json()).data;
            }
            updateNav();
            showAlert('Login berhasil', 'success');
            navigate('/profile');
        } else {
            errorEl.innerHTML = `<div class="alert alert-error">${esc(data.error || 'Login gagal')}</div>`;
        }
    } catch (_) {
        errorEl.innerHTML = '<div class="alert alert-error">Gagal terhubung ke server</div>';
    }
}

function renderRegister() {
    const app = document.getElementById('app');
    const wilayahOpts = WILAYAH_LIST.map(w => `<option value="${esc(w)}">${esc(w)}</option>`).join('');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Daftar Alumni Baru</h2>
            <p class="subtitle">Daftarkan diri Anda sebagai alumni. Status Anda akan <strong>pending</strong> sampai verifikasi email selesai.</p>
            <div id="registerError"></div>
            <form id="registerForm">
                <div class="form-group">
                    <label for="regNama">Nama Lengkap</label>
                    <input type="text" id="regNama" name="nama" required placeholder="Nama Anda">
                </div>
                <div class="form-group">
                    <label for="regEmail">Email</label>
                    <input type="email" id="regEmail" name="email" required placeholder="nama@domain.com">
                </div>
                <div class="form-group">
                    <label for="regPassword">Password</label>
                    <input type="password" id="regPassword" name="password" required minlength="6" placeholder="Minimal 6 karakter">
                </div>
                <div class="form-group">
                    <label for="regNim">NIM</label>
                    <input type="text" id="regNim" name="nim" placeholder="Nomor Induk Mahasiswa">
                </div>
                <div class="form-group">
                    <label for="regFakultas">Jurusan / Fakultas</label>
                    <input type="text" id="regFakultas" name="fakultas" placeholder="Contoh: CSIE / Teknik">
                </div>
                <div class="form-group">
                    <label for="regTahunLulus">Tahun Lulus</label>
                    <input type="number" id="regTahunLulus" name="tahun_lulus" min="1990" max="2035" placeholder="2020">
                </div>
                <div class="form-group">
                    <label for="regWilayah">Wilayah</label>
                    <select id="regWilayah" name="wilayah">
                        <option value="">-- Pilih Wilayah --</option>
                        ${wilayahOpts}
                    </select>
                </div>
                <div class="form-group">
                    <label for="regTelepon">No. Telepon</label>
                    <input type="text" id="regTelepon" name="telepon" placeholder="08123456789">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Daftar</button>
            </form>
            <p style="margin-top: 15px; text-align: center; font-size: 13px;">
                Untuk bisa bergabung di forum dan direktori, verifikasi akun menggunakan <strong>email @mail.ntust.edu.tw</strong> atau SMS/WhatsApp di halaman verifikasi.
            </p>
        </div>
    `;
    document.getElementById('registerForm').addEventListener('submit', handleRegister);
}

async function handleRegister(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const data = {
        nama: String(form.get('nama') || '').trim(),
        email: String(form.get('email') || '').trim(),
        password: String(form.get('password') || ''),
        nim: String(form.get('nim') || '').trim(),
        jurusan: String(form.get('fakultas') || '').trim(),
        tahun_lulus: parseInt(form.get('tahun_lulus'), 10) || 0,
        wilayah: String(form.get('wilayah') || '').trim(),
        telepon: String(form.get('telepon') || '').trim(),
    };
    const errorEl = document.getElementById('registerError');
    errorEl.innerHTML = '';

    try {
        const res = await apiFetch('/register', { method: 'POST', body: JSON.stringify(data) });
        const resp = await res.json();

        if (res.ok || res.status === 201) {
            showAlert(resp.message || 'Pendaftaran berhasil! Silakan login.', 'success');
            navigate('/login');
        } else {
            errorEl.innerHTML = `<div class="alert alert-error">${esc(resp.error || 'Pendaftaran gagal')}</div>`;
        }
    } catch (_) {
        errorEl.innerHTML = '<div class="alert alert-error">Gagal terhubung ke server</div>';
    }
}

let verifyChannel = 'email';
let verifyPhoneDraft = '';
let verifyPhoneStep = 'input'; // 'input' | 'confirm'

function formatPhonePreview(raw) {
    let digits = String(raw || '').replace(/\D/g, '');
    if (digits.startsWith('0')) {
        digits = '62' + digits.slice(1);
    }
    return digits;
}

function channelLabel(ch) {
    if (ch === 'sms') return 'SMS';
    if (ch === 'whatsapp') return 'WhatsApp';
    return 'Email NTUST';
}

function renderVerifyPage(params) {
    if (params && params.get('channel')) {
        const ch = params.get('channel').toLowerCase();
        if (ch === 'email' || ch === 'sms' || ch === 'whatsapp') {
            verifyChannel = ch;
        }
    }
    const app = document.getElementById('app');

    let bodyHtml = '';
    if (verifyChannel === 'email') {
        const defaultEmail = currentUser?.email_ntust || '';
        bodyHtml = `
            <p class="subtitle">Masukkan email <code>@mail.ntust.edu.tw</code> Anda. Sistem akan mengirimkan tautan konfirmasi unik (berlaku 15 menit) beserta kode OTP 6 digit dari <code>ntust-alumni-verification@cerdas.club</code> ke inbox Anda.</p>
            <form id="verifyEmailRequestForm" style="margin-top: 20px;">
                <div class="form-group">
                    <label for="verifyEmail">Email <code>@mail.ntust.edu.tw</code></label>
                    <input type="email" id="verifyEmail" name="email_ntust" required value="${esc(defaultEmail)}" placeholder="nama@mail.ntust.edu.tw">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Kirim Tautan &amp; OTP ke Email</button>
            </form>
            <div id="verifyRequestStatus" style="margin-top: 16px;"></div>
        `;
    } else if (verifyPhoneStep === 'confirm') {
        const normalized = formatPhonePreview(verifyPhoneDraft);
        bodyHtml = `
            <div class="confirm-box">
                <h4>Konfirmasi Pengiriman OTP (${esc(channelLabel(verifyChannel))})</h4>
                <p style="margin: 10px 0;">Kode OTP 6 digit akan dikirimkan melalui <strong>${esc(channelLabel(verifyChannel))}</strong> ke nomor:</p>
                <p style="font-size: 20px; font-weight: 700; color: #2c3e50; margin-bottom: 10px;">+${esc(normalized)}</p>
                <p class="subtitle">Kode OTP berlaku selama 15 menit. Pastikan nomor aktif dan dapat menerima pesan.</p>
                <div style="display:flex; gap:10px; flex-wrap:wrap; margin-top: 16px;">
                    <button type="button" id="confirmSendOtpBtn" class="btn btn-primary">Kirim OTP Sekarang</button>
                    <button type="button" id="changePhoneBtn" class="btn btn-secondary">Ubah Nomor</button>
                </div>
            </div>
            <div id="verifyRequestStatus" style="margin-top: 16px;"></div>
        `;
    } else {
        const defaultPhone = verifyPhoneDraft || (verifyChannel === 'whatsapp' ? (currentUser?.no_wa || currentUser?.no_telp || '') : (currentUser?.no_telp || currentUser?.no_wa || ''));
        bodyHtml = `
            <p class="subtitle">Masukkan nomor HP/WhatsApp aktif untuk menerima kode OTP verifikasi melalui <strong>${esc(channelLabel(verifyChannel))}</strong>.</p>
            <form id="verifyPhoneForm" style="margin-top: 20px;">
                <div class="form-group">
                    <label for="verifyPhoneInput">Nomor ${esc(channelLabel(verifyChannel))}</label>
                    <input type="tel" id="verifyPhoneInput" name="phone" required value="${esc(defaultPhone)}" placeholder="08123456789 atau 628123456789">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Lanjut ke Konfirmasi</button>
            </form>
            <div id="verifyRequestStatus" style="margin-top: 16px;"></div>
        `;
    }

        app.innerHTML = `
            <div class="card">
                <h2 class="card-title">Verifikasi Akun</h2>
            <div class="channel-tabs" id="verifyChannelTabs">
                <button type="button" data-channel="email" class="channel-tab ${verifyChannel === 'email' ? 'active' : ''}">Email NTUST</button>
                <button type="button" data-channel="sms" class="channel-tab ${verifyChannel === 'sms' ? 'active' : ''}">SMS</button>
                <button type="button" data-channel="whatsapp" class="channel-tab ${verifyChannel === 'whatsapp' ? 'active' : ''}">WhatsApp</button>
            </div>
                ${bodyHtml}
            </div>
        `;

    const tabsEl = document.getElementById('verifyChannelTabs');
    if (tabsEl) {
        tabsEl.children && Array.from(tabsEl.children).forEach(btn => {
            btn.addEventListener('click', () => {
                verifyChannel = btn.dataset?.channel || btn.getAttribute?.('data-channel') || 'email';
                verifyPhoneStep = 'input';
                renderVerifyPage();
            });
        });
    }

    const emailForm = document.getElementById('verifyEmailRequestForm');
    if (emailForm) {
        emailForm.addEventListener('submit', handleRequestEmailVerification);
    }
    const phoneForm = document.getElementById('verifyPhoneForm');
    if (phoneForm) {
        phoneForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const rawPhone = String(new FormData(e.target).get('phone') || '').trim();
            const normalized = formatPhonePreview(rawPhone);
            const statusEl = document.getElementById('verifyRequestStatus');
            if (normalized.length < 9 || normalized.length > 15) {
                if (statusEl) statusEl.innerHTML = '<div class="alert alert-error">Nomor telepon tidak valid (9-15 digit, mis. 08123456789).</div>';
                return;
            }
            verifyPhoneDraft = rawPhone;
            verifyPhoneStep = 'confirm';
            renderVerifyPage();
        });
    }
    const confirmSendBtn = document.getElementById('confirmSendOtpBtn');
    if (confirmSendBtn) {
        confirmSendBtn.addEventListener('click', () => handleRequestPhoneVerification(verifyChannel, verifyPhoneDraft));
    }
    const changePhoneBtn = document.getElementById('changePhoneBtn');
    if (changePhoneBtn) {
        changePhoneBtn.addEventListener('click', () => {
            verifyPhoneStep = 'input';
            renderVerifyPage();
        });
    }
}

async function handleRequestEmailVerification(e) {
    e.preventDefault();
    if (!token || !currentUser) {
        showAlert('Silakan login dulu', 'info');
        navigate('/login');
        return;
    }
    const form = new FormData(e.target);
    const email = String(form.get('email_ntust') || '').trim();
    const statusEl = document.getElementById('verifyRequestStatus');
    statusEl.innerHTML = '';

    try {
        const res = await apiFetch('/request-verification', {
            method: 'POST',
            body: JSON.stringify({ channel: 'email', email_ntust: email })
        });
        const data = await res.json();

        if (res.ok) {
            const vToken = data.data?.verification_token || '';
            if (vToken) {
                localStorage.setItem('verifyToken', vToken);
            }
            const targetMasked = data.data?.target || email;
            const confirmHref = `#/verify/confirm?token=${encodeURIComponent(vToken)}&channel=email`;
            statusEl.innerHTML = `
                <div class="alert alert-success">
                    Tautan verifikasi unik dan kode OTP 6 digit telah dikirim ke <strong>${esc(targetMasked)}</strong> (berlaku 15 menit). Silakan buka inbox email Anda dan klik tautan tersebut untuk memasukkan kode OTP.
                    ${vToken ? `<div style="margin-top:8px;"><a href="${esc(confirmHref)}" class="btn-link">Sudah menerima OTP? Masukkan OTP di sini</a></div>` : ''}
                </div>
            `;
        } else {
            statusEl.innerHTML = `<div class="alert alert-error">${esc(data.error || 'Gagal mengirim email verifikasi')}</div>`;
        }
    } catch (_) {
        statusEl.innerHTML = '<div class="alert alert-error">Gagal terhubung ke server</div>';
    }
}

async function handleRequestPhoneVerification(channel, rawPhone) {
    if (!token || !currentUser) {
        showAlert('Silakan login dulu', 'info');
        navigate('/login');
        return;
    }
    const statusEl = document.getElementById('verifyRequestStatus');
    if (statusEl) statusEl.innerHTML = '';

    try {
        const res = await apiFetch('/request-verification', {
            method: 'POST',
            body: JSON.stringify({ channel, phone: rawPhone })
        });
        const data = await res.json();

        if (res.ok) {
            const vToken = data.data?.verification_token || '';
            if (vToken) {
                localStorage.setItem('verifyToken', vToken);
            }
            verifyPhoneStep = 'input';
            showAlert(data.message || 'Kode OTP berhasil dikirim', 'success');
            navigate(`/verify/confirm?token=${encodeURIComponent(vToken)}&channel=${encodeURIComponent(channel)}`);
        } else if (statusEl) {
            statusEl.innerHTML = `<div class="alert alert-error">${esc(data.error || 'Gagal mengirim OTP')}</div>`;
        }
    } catch (_) {
        if (statusEl) statusEl.innerHTML = '<div class="alert alert-error">Gagal terhubung ke server</div>';
    }
}

async function renderVerifyConfirmPage(params) {
    const paramToken = params ? (params.get('token') || '') : '';
    const paramChannel = params ? (params.get('channel') || '') : '';
    if (paramToken) {
        localStorage.setItem('verifyToken', paramToken);
    }
    const activeToken = paramToken || localStorage.getItem('verifyToken') || '';
    const app = document.getElementById('app');

    if (!activeToken) {
        app.innerHTML = `
            <div class="card">
                <h2 class="card-title">Konfirmasi Verifikasi</h2>
                <div class="alert alert-error">Tautan atau sesi verifikasi tidak ditemukan.</div>
                <a href="#/verify" class="btn btn-primary">Minta Verifikasi Ulang</a>
            </div>
        `;
        return;
    }

    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Konfirmasi Kode OTP</h2>
            <div id="verifySessionInfo"><div class="empty-state">Memeriksa tautan verifikasi...</div></div>
        </div>
    `;

    try {
        const stRes = await apiFetch('/verify/status?token=' + encodeURIComponent(activeToken));
        const stData = await stRes.json();
        const infoEl = document.getElementById('verifySessionInfo');
        if (!infoEl) return;

        if (!stRes.ok || !stData.data?.valid) {
            localStorage.removeItem('verifyToken');
            infoEl.innerHTML = `
                <div class="alert alert-error">${esc(stData.error || 'Tautan atau sesi verifikasi sudah tidak berlaku atau kadaluarsa.')}</div>
                <a href="#/verify" class="btn btn-primary">Minta Verifikasi Ulang</a>
            `;
            return;
        }

        const ch = stData.data.channel || paramChannel || 'email';
        const target = stData.data.target || '-';
        const expStr = stData.data.expires_at ? formatDate(stData.data.expires_at) : '15 menit';
        const defaultNim = currentUser?.nim || '';

        infoEl.innerHTML = `
            <div class="confirm-box" style="margin-bottom: 18px;">
                <span class="badge badge-admin">${esc(channelLabel(ch))}</span>
                <p style="margin-top: 8px;">Tujuan: <strong>${esc(target)}</strong></p>
                <p class="subtitle">Berlaku hingga: ${esc(expStr)}</p>
            </div>
            <form id="verifyOtpForm">
                <input type="hidden" id="verifyTokenInput" name="token" value="${esc(activeToken)}">
                <div class="form-group">
                    <label for="verifyOtp">Kode OTP (6 digit)</label>
                    <input type="text" id="verifyOtp" name="otp" class="otp-input" required maxlength="6" placeholder="123456">
                </div>
                <div class="form-group">
                    <label for="verifyNim">NIM (opsional jika sudah diisi saat pendaftaran)</label>
                    <input type="text" id="verifyNim" name="nim" value="${esc(defaultNim)}" placeholder="Nomor Induk Mahasiswa">
                </div>
                <div style="display:flex; gap:10px; flex-wrap:wrap;">
                    <button type="submit" class="btn btn-primary">Verifikasi Akun</button>
                    <a href="#/verify" class="btn btn-secondary">Ganti Metode / Kirim Ulang</a>
                </div>
            </form>
            <div id="verifyOtpStatus" style="margin-top: 16px;"></div>
        `;
        document.getElementById('verifyOtpForm').addEventListener('submit', handleVerifyOtp);
    } catch (_) {
        const infoEl = document.getElementById('verifySessionInfo');
        if (infoEl) {
            infoEl.innerHTML = `
                <div class="alert alert-error">Gagal memeriksa status tautan verifikasi.</div>
                <a href="#/verify" class="btn btn-primary">Kembali ke Verifikasi</a>
            `;
        }
    }
}

async function handleVerifyOtp(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const verifyToken = String(form.get('token') || localStorage.getItem('verifyToken') || '').trim();
    const otp = String(form.get('otp') || '').trim();
    const nim = String(form.get('nim') || '').trim();
    const statusEl = document.getElementById('verifyOtpStatus');
    statusEl.innerHTML = '';

    try {
        const res = await apiFetch('/verify', {
            method: 'POST',
            body: JSON.stringify({ token: verifyToken, otp, nim })
        });
        const data = await res.json();

        if (res.ok) {
            localStorage.removeItem('verifyToken');
            if (data.data && data.data.token) {
                token = data.data.token;
                localStorage.setItem('token', token);
            }
            if (token) {
                const meRes = await apiFetch('/me');
                if (meRes.ok) {
                    currentUser = (await meRes.json()).data;
                }
            }
            updateNav();
            showAlert(data.message || 'Verifikasi berhasil', 'success');
            navigate(currentUser ? '/profile' : '/login');
        } else {
            statusEl.innerHTML = `<div class="alert alert-error">${esc(data.error || 'Verifikasi gagal')}</div>`;
        }
    } catch (_) {
        statusEl.innerHTML = '<div class="alert alert-error">Gagal terhubung ke server</div>';
    }
}

function getWilayahOptions(current) {
    return WILAYAH_LIST.map(w => {
        const selected = w === current ? ' selected' : '';
        return `<option value="${esc(w)}"${selected}>${esc(w)}</option>`;
    }).join('');
}

function renderAvatarMarkup(user, cssClass) {
    const safeImg = safeImageDataUrl(user?.foto_profil);
    if (safeImg) {
        return `<img src="${esc(safeImg)}" alt="${esc(user?.nama || 'Foto')}" class="${cssClass}">`;
    }
    const initial = esc((user?.nama || '?').trim().charAt(0).toUpperCase() || '?');
    return `<div class="${cssClass} avatar-placeholder">${initial}</div>`;
}

function renderProfile() {
    if (!currentUser) { navigate('/login'); return; }
    const app = document.getElementById('app');
    const u = currentUser;
    pendingFotoProfil = safeImageDataUrl(u.foto_profil);

    const verifyBanner = u.status !== 'verified'
        ? `<div class="verify-banner">
                <span>Akun Anda masih <strong>Pending</strong>. Verifikasi akun lewat email <code>@mail.ntust.edu.tw</code> atau SMS/WhatsApp untuk mengakses Direktori &amp; membuat postingan Forum.</span>
                <a href="#/verify" class="btn btn-primary btn-sm">Verifikasi Sekarang</a>
           </div>`
        : '';

    app.innerHTML = `
        <div class="card">
            ${verifyBanner}
            <div class="profile-summary">
                ${renderAvatarMarkup(u, 'avatar-preview')}
                <div>
                    <h2>${esc(u.nama || '-')}</h2>
                    <p class="sub">${esc(u.email)} ${u.nim ? '• NIM: ' + esc(u.nim) : ''}</p>
                    <span class="badge badge-${u.status === 'verified' ? 'verified' : 'pending'}">${u.status === 'verified' ? 'Terverifikasi' : 'Pending'}</span>
                    <span class="badge badge-admin">${esc(u.role || 'member')}</span>
                </div>
            </div>
            <div style="margin-top: 24px;">
                <h3 class="profile-section">Profil &amp; Kontak</h3>
                <form id="profileForm" style="background: #f8f9fa; padding: 20px; border-radius: 8px;">
                    <div class="form-group">
                        <label for="pFotoFile">Foto Profil (PNG/JPG/WEBP, maks 2 MB)</label>
                        <div style="display:flex; align-items:center; gap:12px; margin-bottom:8px;">
                            <img id="profilePhotoPreview" class="avatar-preview" style="${pendingFotoProfil ? '' : 'display:none;'}" src="${esc(pendingFotoProfil)}" alt="Preview Foto">
                            <input type="file" id="pFotoFile" accept="image/png,image/jpeg,image/webp,image/gif">
                        </div>
                    </div>
                    <div class="form-group">
                        <label for="pNama">Nama</label>
                        <input type="text" id="pNama" name="nama" value="${esc(u.nama || '')}">
                    </div>
                    <div class="form-group">
                        <label for="pNim">NIM</label>
                        <input type="text" id="pNim" name="nim" value="${esc(u.nim || '')}">
                    </div>
                    <div class="form-group">
                        <label for="pJurusan">Jurusan</label>
                        <input type="text" id="pJurusan" name="jurusan" value="${esc(u.jurusan || '')}">
                    </div>
                    <div class="form-group">
                        <label for="pNoTelp">No. Telepon</label>
                        <input type="text" id="pNoTelp" name="no_telp" value="${esc(u.no_telp || '')}">
                    </div>
                    <div class="form-group">
                        <label for="pNoWA">No. WA</label>
                        <input type="text" id="pNoWA" name="no_wa" value="${esc(u.no_wa || '')}">
                    </div>
                    <div class="form-group">
                        <label for="pTelegram">Telegram</label>
                        <input type="text" id="pTelegram" name="telegram" value="${esc(u.telegram || '')}">
                    </div>
                    <div class="form-group">
                        <label>Akun Sosmed</label>
                        <div class="sosmed-list" style="margin-bottom: 8px;">
                            <input type="text" id="pFB" name="facebook" placeholder="Facebook" value="${esc(u.sosmed?.facebook || '')}">
                            <input type="text" id="pIG" name="instagram" placeholder="Instagram" value="${esc(u.sosmed?.instagram || '')}">
                            <input type="text" id="pTwitter" name="twitter" placeholder="Twitter/X" value="${esc(u.sosmed?.twitter || '')}">
                            <input type="text" id="pThreads" name="threads" placeholder="Threads" value="${esc(u.sosmed?.threads || '')}">
                        </div>
                    </div>
                    <div class="form-group">
                        <label for="pLinkedIn">LinkedIn URL</label>
                        <input type="url" id="pLinkedIn" name="linkedin" placeholder="https://linkedin.com/in/..." value="${esc(u.linkedin || '')}">
                    </div>
                    <div class="form-group">
                        <label for="pBidangKeahlian">Bidang Keahlian (pisahkan dengan koma)</label>
                        <input type="text" id="pBidangKeahlian" name="bidang_keahlian" value="${esc((u.bidang_keahlian || []).join(', '))}" placeholder="AI, Cloud, Backend">
                    </div>
                    <div class="form-group">
                        <label for="pPreferensiKontak">Preferensi Kontak</label>
                        <input type="text" id="pPreferensiKontak" name="preferensi_kontak" value="${esc(u.preferensi_kontak || '')}" placeholder="WhatsApp / Email / Telegram">
                    </div>
                    <div class="form-group">
                        <label for="pStatusKeaktifan">Status Keaktifan</label>
                        <select id="pStatusKeaktifan" name="status_keaktifan">
                            <option value="aktif"${u.status_keaktifan === 'aktif' ? ' selected' : ''}>Aktif</option>
                            <option value="tidak aktif"${u.status_keaktifan === 'tidak aktif' ? ' selected' : ''}>Tidak Aktif</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="pWilayah">Wilayah</label>
                        <select id="pWilayah" name="wilayah">
                            <option value="">-- Pilih Wilayah --</option>
                            ${getWilayahOptions(u.wilayah)}
                        </select>
                    </div>
                    <button type="submit" id="saveProfileBtn" class="btn btn-primary" style="margin-top: 12px;">Simpan Profil</button>
                </form>

                <h3 class="profile-section" style="margin-top: 24px;">Pekerjaan</h3>
                <div class="edu-field" id="pekerjaanDisplay"></div>
                <button type="button" class="btn btn-link" id="editPekerjaan">Edit Pekerjaan</button>

                <h3 class="profile-section" style="margin-top: 24px;">Pendidikan di NTUST</h3>
                <div class="edu-field" id="pendidikanDisplay"></div>
                <button type="button" class="btn btn-link" id="editPendidikan">Edit Pendidikan NTUST</button>
            </div>
        </div>
    `;

    document.getElementById('pFotoFile').addEventListener('change', handlePhotoSelect);
    document.getElementById('profileForm').addEventListener('submit', handleProfileSave);
    renderPekerjaanDisplay(u);
    document.getElementById('editPekerjaan').addEventListener('click', () => renderPekerjaanEdit(u));
    renderPendidikanDisplay(u);
    document.getElementById('editPendidikan').addEventListener('click', () => renderPendidikanEdit(u));
}

function handlePhotoSelect(e) {
    const file = e.target.files && e.target.files[0];
    if (!file) return;

    if (!file.type.startsWith('image/') || file.type === 'image/svg+xml') {
        showAlert('Format foto harus PNG, JPG, WEBP, atau GIF (SVG tidak didukung)', 'error');
        e.target.value = '';
        return;
    }
    if (file.size > MAX_PHOTO_BYTES) {
        showAlert('Ukuran foto maksimal 2 MB', 'error');
        e.target.value = '';
        return;
    }

    const reader = new FileReader();
    reader.onload = () => {
        const dataUrl = safeImageDataUrl(reader.result);
        if (!dataUrl) {
            showAlert('Format foto tidak valid', 'error');
            return;
        }
        pendingFotoProfil = dataUrl;
        const preview = document.getElementById('profilePhotoPreview');
        if (preview) {
            preview.src = dataUrl;
            preview.style.display = 'block';
        }
    };
    reader.readAsDataURL(file);
}

function renderPekerjaanDisplay(user) {
    const div = document.getElementById('pekerjaanDisplay');
    const p = user?.pekerjaan;
    if (!p) { div.textContent = 'Belum diisi'; return; }
    div.innerHTML = `
        <p><strong>Institusi:</strong> ${esc(p.institusi || '-')}</p>
        <p><strong>Jenis Industri:</strong> ${esc(p.jenis_industri || '-')}</p>
        <p><strong>Jabatan:</strong> ${esc(p.jabatan || '-')}</p>
        <p><strong>Alamat Kantor:</strong> ${esc(p.alamat_kantor || '-')}</p>
        <p><strong>Telp Kantor:</strong> ${esc(p.telp_kantor || '-')}</p>
        <p><strong>Email Kerja:</strong> ${esc(p.email_kerja || '-')}</p>
    `;
}

function renderPekerjaanEdit(user) {
    const pekerjaan = user?.pekerjaan || {};
    document.getElementById('pekerjaanDisplay').innerHTML = `
        <form id="pekerjaanForm">
            <div class="form-group">
                <label for="pInstitusi">Institusi</label><input type="text" id="pInstitusi" name="institusi" value="${esc(pekerjaan.institusi || '')}">
            </div>
            <div class="form-group">
                <label for="pIndustri">Jenis Industri</label><input type="text" id="pIndustri" name="jenis_industri" value="${esc(pekerjaan.jenis_industri || '')}">
            </div>
            <div class="form-group">
                <label for="pJabatan">Jabatan</label><input type="text" id="pJabatan" name="jabatan" value="${esc(pekerjaan.jabatan || '')}">
            </div>
            <div class="form-group">
                <label for="pAlamatKantor">Alamat Kantor</label><input type="text" id="pAlamatKantor" name="alamat_kantor" value="${esc(pekerjaan.alamat_kantor || '')}">
            </div>
            <div class="form-group">
                <label for="pTelpKantor">Telp Kantor</label><input type="text" id="pTelpKantor" name="telp_kantor" value="${esc(pekerjaan.telp_kantor || '')}">
            </div>
            <div class="form-group">
                <label for="pEmailKerja">Email Kerja</label><input type="email" id="pEmailKerja" name="email_kerja" value="${esc(pekerjaan.email_kerja || '')}">
            </div>
            <div style="display:flex; gap:10px;">
                <button type="submit" class="btn btn-primary">Simpan Pekerjaan</button>
                <button type="button" id="cancelPekerjaan" class="btn btn-secondary">Batal</button>
            </div>
        </form>
    `;
    document.getElementById('pekerjaanForm').addEventListener('submit', handlePekerjaanSave);
    document.getElementById('cancelPekerjaan').addEventListener('click', () => renderPekerjaanDisplay(currentUser));
}

async function handlePekerjaanSave(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const pekerjaan = {
        institusi: String(form.get('institusi') || '').trim(),
        jenis_industri: String(form.get('jenis_industri') || '').trim(),
        jabatan: String(form.get('jabatan') || '').trim(),
        alamat_kantor: String(form.get('alamat_kantor') || '').trim(),
        telp_kantor: String(form.get('telp_kantor') || '').trim(),
        email_kerja: String(form.get('email_kerja') || '').trim(),
    };
    const res = await apiFetch('/alumni/' + encodeURIComponent(currentUser.id), {
        method: 'PUT',
        body: JSON.stringify({ pekerjaan })
    });
    if (res.ok) {
        const resp = await res.json();
        currentUser = resp.data || { ...currentUser, pekerjaan };
        showAlert('Pekerjaan berhasil disimpan', 'success');
        renderPekerjaanDisplay(currentUser);
    } else {
        const data = await res.json();
        showAlert(data.error || 'Gagal menyimpan pekerjaan', 'error');
    }
}

function renderPendidikanDisplay(user) {
    const div = document.getElementById('pendidikanDisplay');
    const edu = user?.pendidikan_ntust;
    if (!edu) { div.textContent = 'Belum diisi'; return; }
    div.innerHTML = `
        <p><strong>Fakultas:</strong> ${esc(edu.fakultas || '-')}</p>
        <p><strong>Jurusan:</strong> ${esc(edu.jurusan || '-')}</p>
        <p><strong>Tahun Masuk:</strong> ${esc(edu.tahun_masuk || '-')}</p>
        <p><strong>Tahun Lulus:</strong> ${esc(edu.tahun_lulus_studi || '-')}</p>
        <p><strong>Supervisor:</strong> ${esc(edu.supervisor || '-')}</p>
    `;
}

function renderPendidikanEdit(user) {
    const edu = user?.pendidikan_ntust || {};
    document.getElementById('pendidikanDisplay').innerHTML = `
        <form id="pendidikanForm">
            <div class="form-group">
                <label for="eduFakultas">Fakultas</label><input type="text" id="eduFakultas" name="fakultas" value="${esc(edu.fakultas || '')}">
            </div>
            <div class="form-group">
                <label for="eduJurusan">Jurusan</label><input type="text" id="eduJurusan" name="jurusan" value="${esc(edu.jurusan || '')}">
            </div>
            <div class="form-group">
                <label for="eduTahunMasuk">Tahun Masuk</label><input type="number" id="eduTahunMasuk" name="tahun_masuk" value="${esc(edu.tahun_masuk || '')}">
            </div>
            <div class="form-group">
                <label for="eduTahunLulus">Tahun Lulus</label><input type="number" id="eduTahunLulus" name="tahun_lulus_studi" value="${esc(edu.tahun_lulus_studi || '')}">
            </div>
            <div class="form-group">
                <label for="eduSupervisor">Supervisor</label><input type="text" id="eduSupervisor" name="supervisor" value="${esc(edu.supervisor || '')}">
            </div>
            <div style="display:flex; gap:10px;">
                <button type="submit" class="btn btn-primary">Simpan Pendidikan</button>
                <button type="button" id="cancelPendidikan" class="btn btn-secondary">Batal</button>
            </div>
        </form>
    `;
    document.getElementById('pendidikanForm').addEventListener('submit', handlePendidikanSave);
    document.getElementById('cancelPendidikan').addEventListener('click', () => renderPendidikanDisplay(currentUser));
}

async function handlePendidikanSave(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const pendidikan_ntust = {
        fakultas: String(form.get('fakultas') || '').trim(),
        jurusan: String(form.get('jurusan') || '').trim(),
        tahun_masuk: parseInt(form.get('tahun_masuk'), 10) || 0,
        tahun_lulus_studi: parseInt(form.get('tahun_lulus_studi'), 10) || 0,
        supervisor: String(form.get('supervisor') || '').trim(),
    };
    const res = await apiFetch('/alumni/' + encodeURIComponent(currentUser.id), {
        method: 'PUT',
        body: JSON.stringify({ pendidikan_ntust })
    });
    if (res.ok) {
        const resp = await res.json();
        currentUser = resp.data || { ...currentUser, pendidikan_ntust };
        showAlert('Pendidikan NTUST berhasil disimpan', 'success');
        renderPendidikanDisplay(currentUser);
    } else {
        const data = await res.json();
        showAlert(data.error || 'Gagal menyimpan pendidikan', 'error');
    }
}

async function handleProfileSave(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const bidangStr = String(form.get('bidang_keahlian') || '');
    const bidang_keahlian = bidangStr.split(',').map(s => s.trim()).filter(Boolean);

    const update = {
        nama: String(form.get('nama') || '').trim(),
        nim: String(form.get('nim') || '').trim(),
        jurusan: String(form.get('jurusan') || '').trim(),
        no_telp: String(form.get('no_telp') || '').trim(),
        no_wa: String(form.get('no_wa') || '').trim(),
        telegram: String(form.get('telegram') || '').trim(),
        sosmed: {
            facebook: String(form.get('facebook') || '').trim(),
            instagram: String(form.get('instagram') || '').trim(),
            twitter: String(form.get('twitter') || '').trim(),
            threads: String(form.get('threads') || '').trim(),
        },
        foto_profil: pendingFotoProfil || '',
        linkedin: String(form.get('linkedin') || '').trim(),
        bidang_keahlian,
        preferensi_kontak: String(form.get('preferensi_kontak') || '').trim(),
        status_keaktifan: String(form.get('status_keaktifan') || '').trim(),
        wilayah: String(form.get('wilayah') || '').trim(),
    };
    const res = await apiFetch('/alumni/' + encodeURIComponent(currentUser.id), {
        method: 'PUT',
        body: JSON.stringify(update)
    });
    if (res.ok) {
        showAlert('Profil berhasil disimpan', 'success');
        const meRes = await apiFetch('/me');
        if (meRes.ok) {
            currentUser = (await meRes.json()).data;
        }
        renderProfile();
    } else {
        const data = await res.json();
        showAlert(data.error || 'Gagal menyimpan', 'error');
    }
}

// Directory
async function renderDirectory() {
    const app = document.getElementById('app');

    if (!currentUser) {
        app.innerHTML = `
            <div class="card">
                <h2 class="card-title">Direktori Alumni</h2>
                <p style="margin: 12px 0;">Silakan login terlebih dahulu untuk mengakses direktori alumni.</p>
                <a href="#/login" class="btn btn-primary">Login</a>
            </div>
        `;
        return;
    }

    if (currentUser.status !== 'verified') {
        app.innerHTML = `
            <div class="card">
                <h2 class="card-title">Direktori Alumni</h2>
                <p style="margin: 12px 0;">Direktori alumni hanya dapat diakses oleh alumni yang telah terverifikasi email <code>@mail.ntust.edu.tw</code>.</p>
                <a href="#/verify" class="btn btn-primary">Verifikasi Akun</a>
            </div>
        `;
        return;
    }

    app.innerHTML = `
        <div class="section-header">
            <h3>Direktori Alumni</h3>
            <span id="dirSummary" style="color: #7f8c8d; font-size: 14px;">Alumni Terverifikasi</span>
        </div>
        <form id="dirFilterForm" class="directory-controls">
            <input type="text" id="dirSearchInput" name="q" value="${esc(dirState.q)}" placeholder="Cari nama, NIM, perusahaan, jabatan...">
            <input type="text" id="dirJurusanInput" name="jurusan" value="${esc(dirState.jurusan)}" placeholder="Filter jurusan (mis. CSIE)">
            <select id="dirSortSelect" name="sort">
                <option value="nama_asc"${dirState.sort === 'nama_asc' ? ' selected' : ''}>Nama A-Z</option>
                <option value="nama_desc"${dirState.sort === 'nama_desc' ? ' selected' : ''}>Nama Z-A</option>
            </select>
            <button type="submit" class="btn btn-primary">Cari</button>
        </form>
        <div class="wilayah-filter" id="dirWilayahFilter"></div>
        <div id="dirList"></div>
        <div id="dirPagination" class="pagination"></div>
    `;

    document.getElementById('dirFilterForm').addEventListener('submit', (e) => {
        e.preventDefault();
        dirState.q = document.getElementById('dirSearchInput').value.trim();
        dirState.jurusan = document.getElementById('dirJurusanInput').value.trim();
        dirState.sort = document.getElementById('dirSortSelect').value;
        dirState.page = 1;
        loadDirectory();
    });

    document.getElementById('dirSortSelect').addEventListener('change', (e) => {
        dirState.sort = e.target.value;
        dirState.page = 1;
        loadDirectory();
    });

    initWilayahFilter('dirWilayahFilter');
    await loadDirectory();
}

function initWilayahFilter(containerId) {
    const container = document.getElementById(containerId);
    if (!container) return;
    const chips = [''].concat(WILAYAH_LIST);
    container.innerHTML = '';
    chips.forEach(w => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.textContent = w || 'Semua';
        btn.className = dirState.wilayah === w ? 'active' : '';
        btn.addEventListener('click', () => {
            dirState.wilayah = w;
            dirState.page = 1;
            initWilayahFilter(containerId);
            loadDirectory();
        });
        container.appendChild(btn);
    });
}

async function loadDirectory() {
    const listEl = document.getElementById('dirList');
    const summaryEl = document.getElementById('dirSummary');
    const pageEl = document.getElementById('dirPagination');
    if (!listEl) return;

    listEl.className = '';
    listEl.innerHTML = '<div class="empty-state">Memuat...</div>';
    if (pageEl) pageEl.innerHTML = '';

    const qs = new URLSearchParams();
    if (dirState.q) qs.set('q', dirState.q);
    if (dirState.wilayah) qs.set('wilayah', dirState.wilayah);
    if (dirState.jurusan) qs.set('jurusan', dirState.jurusan);
    qs.set('sort', dirState.sort);
    qs.set('page', String(dirState.page));
    qs.set('limit', String(dirState.limit));

    try {
        const res = await apiFetch('/alumni?' + qs.toString());
        const payload = await res.json();

        if (!res.ok) {
            listEl.innerHTML = `<div class="empty-state">${esc(payload.error || 'Gagal memuat direktori')}</div>`;
            return;
        }

        const alumni = payload.data?.alumni || [];
        dirState.total = payload.data?.total ?? alumni.length;

        if (summaryEl) {
            const prefix = dirState.wilayah ? dirState.wilayah + ' • ' : '';
            summaryEl.textContent = `${prefix}${dirState.total} Alumni Terverifikasi`;
        }

        if (alumni.length === 0) {
            listEl.innerHTML = '<div class="empty-state">Tidak ada alumni yang cocok dengan filter</div>';
            return;
        }

        listEl.className = 'alumni-grid';
        listEl.innerHTML = '';
        alumni.forEach(a => {
            const card = document.createElement('div');
            card.className = 'alumni-card';
            card.innerHTML = `
                <div style="display:flex; align-items:center; gap:12px; margin-bottom:8px;">
                    ${renderAvatarMarkup(a, 'avatar-thumb')}
                    <div>
                        <h4 class="alumni-card-name" style="cursor:pointer; text-decoration:underline;">${esc(a.nama || '-')}</h4>
                        <div class="wilayah">${esc(a.wilayah || '-')} ${a.jurusan ? '• ' + esc(a.jurusan) : ''}</div>
                    </div>
                </div>
                <span class="badge badge-${a.role === 'member' ? 'verified' : 'admin'}">${esc(a.role || 'member')}</span>
            `;
            card.querySelector('.alumni-card-name').addEventListener('click', async () => {
                const detailRes = await apiFetch('/alumni/' + encodeURIComponent(a.id));
                if (detailRes.ok) {
                    const detailData = await detailRes.json();
                    showAlumniDetail(detailData.data);
                }
            });
            listEl.appendChild(card);
        });

        if (pageEl) {
            const totalPages = Math.max(1, Math.ceil(dirState.total / dirState.limit));
            const prevBtn = document.createElement('button');
            prevBtn.type = 'button';
            prevBtn.className = 'btn btn-secondary btn-sm';
            prevBtn.textContent = 'Sebelumnya';
            prevBtn.disabled = dirState.page <= 1;
            prevBtn.addEventListener('click', () => {
                if (dirState.page > 1) {
                    dirState.page--;
                    loadDirectory();
                }
            });

            const info = document.createElement('span');
            info.textContent = `Halaman ${dirState.page} dari ${totalPages}`;

            const nextBtn = document.createElement('button');
            nextBtn.type = 'button';
            nextBtn.className = 'btn btn-secondary btn-sm';
            nextBtn.textContent = 'Berikutnya';
            nextBtn.disabled = dirState.page >= totalPages;
            nextBtn.addEventListener('click', () => {
                if (dirState.page < totalPages) {
                    dirState.page++;
                    loadDirectory();
                }
            });

            pageEl.appendChild(prevBtn);
            pageEl.appendChild(info);
            pageEl.appendChild(nextBtn);
        }
    } catch (_) {
        listEl.innerHTML = '<div class="empty-state">Gagal memuat direktori</div>';
    }
}

function showAlumniDetail(alumni) {
    if (!alumni) return;
    const app = document.getElementById('app');
    const linkedinHref = safeUrl(alumni.linkedin);
    const pekerjaanText = alumni.pekerjaan
        ? `${esc(alumni.pekerjaan.jabatan || '-')} di ${esc(alumni.pekerjaan.institusi || '-')}`
        : '-';
    const pendidikanText = alumni.pendidikan_ntust
        ? `${esc(alumni.pendidikan_ntust.fakultas || '-')} / ${esc(alumni.pendidikan_ntust.jurusan || alumni.jurusan || '-')}`
        : esc(alumni.jurusan || '-');

    app.innerHTML = `
        <div class="card">
            <div style="display:flex; align-items:center; gap:16px; margin-bottom:16px;">
                ${renderAvatarMarkup(alumni, 'avatar-preview')}
                <div>
                    <h2 class="card-title">${esc(alumni.nama || '-')}</h2>
                    <p class="subtitle">${esc(alumni.wilayah || '-')} • ${esc(alumni.role || 'member')}</p>
                </div>
            </div>
            <div class="sosmed-list" style="margin: 12px 0;">
                ${alumni.email ? `<span>Email: ${esc(alumni.email)}</span>` : ''}
                ${alumni.no_wa ? `<span>WA: ${esc(alumni.no_wa)}</span>` : ''}
                ${alumni.telegram ? `<span>Telegram: ${esc(alumni.telegram)}</span>` : ''}
                ${linkedinHref ? `<a href="${esc(linkedinHref)}" target="_blank" rel="noopener noreferrer" style="color: #3498db;">LinkedIn</a>` : ''}
            </div>
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin: 16px 0;">
                <div>
                    <p><strong>Wilayah:</strong> ${esc(alumni.wilayah || '-')}</p>
                    <p><strong>NIM:</strong> ${esc(alumni.nim || '-')}</p>
                    <p><strong>Bidang Keahlian:</strong> ${esc((alumni.bidang_keahlian || []).join(', ') || '-')}</p>
                </div>
                <div>
                    <p><strong>Pekerjaan:</strong><br>${pekerjaanText}</p>
                    <p><strong>Pendidikan:</strong><br>${pendidikanText}</p>
                </div>
            </div>
            <button type="button" id="backToDirBtn" class="btn btn-secondary">Kembali</button>
        </div>
    `;
    document.getElementById('backToDirBtn').addEventListener('click', () => navigate('/directory'));
}

// Forum
async function renderForum() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="section-header">
            <h3>Forum Diskusi</h3>
            <div id="createPostBtn"></div>
        </div>
        <div id="forumHint"></div>
        <div id="forumList"></div>
    `;
    initCreatePostButton();
    await loadForum();
}

function initCreatePostButton() {
    const btnContainer = document.getElementById('createPostBtn');
    const hintContainer = document.getElementById('forumHint');
    if (!btnContainer) return;

    if (currentUser && currentUser.status === 'verified') {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'btn btn-primary';
        btn.textContent = 'Buat Postingan Baru';
        btn.addEventListener('click', openCreatePostForm);
        btnContainer.appendChild(btn);
    } else if (currentUser) {
        hintContainer.innerHTML = '<div class="alert alert-info">Verifikasi akun NTUST Anda terlebih dahulu untuk membuat postingan baru. <a href="#/verify" class="btn-link">Verifikasi</a></div>';
    }
}

function openCreatePostForm() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Buat Postingan</h2>
            <form id="forumPostForm">
                <div class="form-group">
                    <label for="postJudul">Judul</label>
                    <input type="text" id="postJudul" name="judul" required placeholder="Judul Diskusi">
                </div>
                <div class="form-group">
                    <label for="postIsi">Isi</label>
                    <textarea id="postIsi" name="isi" rows="5" required placeholder="Tulis diskusi Anda..."></textarea>
                </div>
                <div style="display:flex; gap:10px;">
                    <button type="submit" class="btn btn-primary">Posting</button>
                    <button type="button" id="cancelPostBtn" class="btn btn-secondary">Batal</button>
                </div>
            </form>
        </div>
    `;
    document.getElementById('forumPostForm').addEventListener('submit', handleCreatePost);
    document.getElementById('cancelPostBtn').addEventListener('click', () => renderForum());
}

async function loadForum() {
    const listEl = document.getElementById('forumList');
    if (!listEl) return;
    listEl.innerHTML = '<div class="empty-state">Memuat...</div>';

    try {
        const res = await apiFetch('/forum');
        const data = await res.json();

        if (!res.ok) {
            listEl.innerHTML = '<div class="empty-state">Gagal memuat forum</div>';
            return;
        }

        const posts = data.data || [];
        if (posts.length === 0) {
            listEl.innerHTML = '<div class="empty-state">Belum ada postingan. Jadi yang pertama!</div>';
            return;
        }

        listEl.innerHTML = '';
        posts.forEach(p => {
            const postEl = document.createElement('div');
            postEl.className = 'forum-post-item';
            postEl.innerHTML = `
                <div class="forum-post-header">
                    <span class="forum-post-author">${esc(p.penulis_nama || '-')}</span>
                    <span class="forum-post-date">${esc(formatDate(p.dibuat_pada))}</span>
                </div>
                <h4 class="forum-post-title">${esc(p.judul || '-')}</h4>
                <div class="forum-post-body">${esc(p.isi || '')}</div>
            `;
            listEl.appendChild(postEl);
        });
    } catch (_) {
        listEl.innerHTML = '<div class="empty-state">Gagal memuat forum</div>';
    }
}

async function handleCreatePost(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const res = await apiFetch('/forum', {
        method: 'POST',
        body: JSON.stringify({
            judul: String(form.get('judul') || '').trim(),
            isi: String(form.get('isi') || '').trim(),
        })
    });
    const data = await res.json();
    if (res.ok || res.status === 201) {
        await renderForum();
        showAlert('Postingan berhasil diposting', 'success');
    } else {
        showAlert(data.error || 'Gagal memposting', 'error');
    }
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('id-ID', {
        year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
    });
}

// Subgroup management
async function renderSubgroups() {
    if (!currentUser) { navigate('/login'); return; }
    const isSuperAdmin = currentUser.role === 'super_admin';
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="section-header">
            <h3>Subgroup Ikatan Alumni Wilayah</h3>
            <div id="subgroupActions"></div>
        </div>
        <div id="subgroupList"></div>
    `;
    if (isSuperAdmin) {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'btn btn-primary';
        btn.textContent = 'Buat Subgroup Baru';
        btn.addEventListener('click', openCreateSubgroupForm);
        document.getElementById('subgroupActions').appendChild(btn);
    }
    await loadSubgroups();
}

function openCreateSubgroupForm() {
    const app = document.getElementById('app');
    const wilayahOpts = WILAYAH_LIST.map(w => `<option value="${esc(w)}">${esc(w)}</option>`).join('');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Buat Subgroup Baru</h2>
            <form id="subgroupForm">
                <div class="form-group">
                    <label for="sgNama">Nama Subgroup</label>
                    <input type="text" id="sgNama" name="nama" required placeholder="IKATAN ALUMNI NTUST WILAYAH JAKARTA">
                </div>
                <div class="form-group">
                    <label for="sgWilayah">Wilayah</label>
                    <select id="sgWilayah" name="wilayah" required>
                        <option value="">Pilih Wilayah</option>
                        ${wilayahOpts}
                    </select>
                </div>
                <div class="form-group">
                    <label for="sgDeskripsi">Deskripsi</label>
                    <textarea id="sgDeskripsi" name="deskripsi" rows="3"></textarea>
                </div>
                <div class="form-group">
                    <label for="sgAdminId">Admin Wilayah (opsional)</label>
                    <input type="text" id="sgAdminId" name="admin_id" placeholder="ID alumni (bisa ditunjuk nanti)">
                </div>
                <div style="display:flex; gap:10px;">
                    <button type="submit" class="btn btn-primary">Simpan Subgroup</button>
                    <button type="button" id="cancelSubgroupBtn" class="btn btn-secondary">Batal</button>
                </div>
            </form>
        </div>
    `;
    document.getElementById('subgroupForm').addEventListener('submit', handleCreateSubgroup);
    document.getElementById('cancelSubgroupBtn').addEventListener('click', () => renderSubgroups());
}

async function handleCreateSubgroup(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const res = await apiFetch('/subgroups', {
        method: 'POST',
        body: JSON.stringify({
            nama: String(form.get('nama') || '').trim(),
            wilayah: String(form.get('wilayah') || '').trim(),
            deskripsi: String(form.get('deskripsi') || '').trim(),
            admin_id: String(form.get('admin_id') || '').trim(),
        })
    });
    const data = await res.json();
    if (res.ok || res.status === 201) {
        await renderSubgroups();
        showAlert('Subgroup berhasil dibuat', 'success');
    } else {
        showAlert(data.error || 'Gagal membuat subgroup', 'error');
    }
}

async function loadSubgroups() {
    const listEl = document.getElementById('subgroupList');
    if (!listEl) return;
    listEl.innerHTML = '<div class="empty-state">Memuat...</div>';

    const isSuperAdmin = currentUser?.role === 'super_admin';
    const isAdmin = currentUser?.role === 'super_admin' || currentUser?.role === 'regional_admin';

    try {
        const res = await apiFetch('/subgroups');
        const data = await res.json();

        if (!res.ok) {
            listEl.innerHTML = '<div class="empty-state">Gagal memuat data</div>';
            return;
        }

        const subs = data.data || [];
        if (subs.length === 0) {
            listEl.innerHTML = '<div class="empty-state">Belum ada subgroup</div>';
            return;
        }

        listEl.className = 'alumni-grid';
        listEl.innerHTML = '';
        subs.forEach(s => {
            const card = document.createElement('div');
            card.className = 'alumni-card';
            const pengurusList = Array.isArray(s.pengurus) && s.pengurus.length > 0
                ? `<div class="pengurus-list">
                        <div class="pengurus-title">Struktur Pengurus</div>
                        ${s.pengurus.map(p => `<div class="pengurus-item"><strong>${esc(p.jabatan || '-')}:</strong> ${esc(p.nama || '-')}</div>`).join('')}
                   </div>`
                : '';

            card.innerHTML = `
                <h4>${esc(s.nama)}</h4>
                <div class="wilayah">${esc(s.wilayah)}</div>
                ${s.deskripsi ? `<p style="font-size: 13px; color: #555;">${esc(s.deskripsi)}</p>` : ''}
                ${s.admin_id ? `<span class="badge badge-admin">Admin: ${esc(s.admin_id.substring(0, 12))}</span>` : ''}
                ${pengurusList}
            `;

            if (isAdmin) {
                const appointForm = document.createElement('form');
                appointForm.style.marginTop = '12px';
                appointForm.innerHTML = `
                    <div style="display:flex; gap:6px;">
                        <input type="text" name="alumni_id" required placeholder="ID Alumni" style="flex:1; padding:6px 8px; border:1px solid #ddd; border-radius:6px; font-size:13px;">
                        <button type="submit" class="btn btn-primary btn-sm">Tunjuk Admin</button>
                    </div>
                `;
                appointForm.addEventListener('submit', async (ev) => {
                    ev.preventDefault();
                    const alumniId = String(new FormData(appointForm).get('alumni_id') || '').trim();
                    const apRes = await apiFetch('/subgroups/' + encodeURIComponent(s.id) + '/admin', {
                        method: 'POST',
                        body: JSON.stringify({ alumni_id: alumniId })
                    });
                    const apData = await apRes.json();
                    if (apRes.ok) {
                        showAlert(apData.message || 'Admin wilayah berhasil ditunjuk', 'success');
                        loadSubgroups();
                    } else {
                        showAlert(apData.error || 'Gagal menunjuk admin', 'error');
                    }
                });
                card.appendChild(appointForm);
            }

            if (isSuperAdmin) {
                const actionsDiv = document.createElement('div');
                actionsDiv.style.marginTop = '10px';
                const delBtn = document.createElement('button');
                delBtn.type = 'button';
                delBtn.className = 'btn btn-danger btn-sm';
                delBtn.textContent = 'Hapus';
                delBtn.addEventListener('click', () => deleteSubgroup(s.id));
                actionsDiv.appendChild(delBtn);
                card.appendChild(actionsDiv);
            }

            listEl.appendChild(card);
        });
    } catch (_) {
        listEl.innerHTML = '<div class="empty-state">Gagal memuat data</div>';
    }
}

async function deleteSubgroup(id) {
    if (typeof confirm === 'function' && !confirm('Hapus subgroup ini?')) return;
    const res = await apiFetch('/subgroups/' + encodeURIComponent(id), { method: 'DELETE' });
    if (res.ok) {
        showAlert('Subgroup berhasil dihapus', 'success');
        loadSubgroups();
    } else {
        showAlert('Gagal menghapus', 'error');
    }
}

// Initialize
window.addEventListener('hashchange', renderRoute);
checkHealth();
if (typeof setInterval === 'function') {
    setInterval(checkHealth, 30000);
}
initUser();
