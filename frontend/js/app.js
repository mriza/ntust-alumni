// app.js - Alumni NTUST Indonesia SPA (Ant Design v5 + Zero Hardcoded Styles)

const hostname = window.location.hostname;
const isFileOrigin = window.location.protocol === 'file:';
const API_BASE = isFileOrigin ? 'http://localhost:8080/api' : '/api';
const HEALTH_URL = isFileOrigin ? 'http://localhost:8080/health' : '/health';
const ANSWER_URL = (window.location.protocol === 'http:' || window.location.protocol === 'https:')
    ? `${window.location.protocol}//${hostname || 'localhost'}:9080`
    : 'http://localhost:9080';
const ANSWER_API_BASE = isFileOrigin ? `${ANSWER_URL}/answer/api/v1` : '/answer-api';

const WILAYAH_LIST = [
    'Jabodetabek', 'Jakarta', 'Bandung', 'Surabaya', 'Medan',
    'Yogyakarta', 'Semarang', 'Makassar', 'Denpasar', 'Balikpapan', 'Lainnya'
];
const MAX_PHOTO_BYTES = 2 * 1024 * 1024;

// Application State
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

// XSS & URL Safety Helpers
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
    if (/^data:image\/(png|jpeg|webp|gif);base64,[A-Za-z0-9+/=]+$/.test(raw)) {
        return raw;
    }
    return '';
}

function getVerifyStorage() {
    try {
        if (typeof sessionStorage !== 'undefined') return sessionStorage;
    } catch (_) {}
    return localStorage;
}

// SVG Icon Helpers (strictly presentational, no inline styles)
const ICONS = {
    shield: '<svg class="feature-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="m9 12 2 2 4-4"/></svg>',
    users: '<svg class="feature-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>',
    forum: '<svg class="feature-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/><path d="M8 9h8"/><path d="M8 13h6"/></svg>',
    map: '<svg class="feature-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/></svg>'
};

// Route Helpers
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

// Theme Switcher & Responsive Navbar Setup
function syncThemeSwitcherUI(pref) {
    const switcher = document.getElementById('themeSwitcher');
    if (!switcher) return;
    const activePref = pref || window.ntustTheme?.getPreference?.() || 'system';
    const buttons = switcher.children ? Array.from(switcher.children) : [];
    buttons.forEach(btn => {
        const val = btn.dataset?.themeValue || btn.getAttribute?.('data-theme-value') || '';
        const isActive = val === activePref;
        if (btn.classList) {
            btn.classList.toggle('is-active', isActive);
        } else {
            btn.className = isActive ? 'ant-theme-btn is-active' : 'ant-theme-btn';
        }
        btn.setAttribute?.('aria-checked', String(isActive));
    });
}

function initHeaderControls() {
    const switcher = document.getElementById('themeSwitcher');
    if (switcher && switcher.children) {
        Array.from(switcher.children).forEach(btn => {
            btn.addEventListener('click', () => {
                const val = btn.dataset?.themeValue || btn.getAttribute?.('data-theme-value') || 'system';
                const nextPref = window.ntustTheme ? window.ntustTheme.setPreference(val) : val;
                syncThemeSwitcherUI(nextPref);
            });
        });
        syncThemeSwitcherUI();
    }

    if (typeof window.addEventListener === 'function') {
        window.addEventListener('ntust:themechange', (ev) => {
            syncThemeSwitcherUI(ev?.detail?.preference);
        });
    }

    const mobileToggle = document.getElementById('mobileNavToggle');
    const siteHeader = document.getElementById('siteHeader');
    if (mobileToggle && siteHeader) {
        mobileToggle.addEventListener('click', () => {
            const isOpen = siteHeader.classList ? siteHeader.classList.toggle('is-mobile-open') : false;
            mobileToggle.setAttribute?.('aria-expanded', String(isOpen));
        });
    }
}

function closeMobileNav() {
    const siteHeader = document.getElementById('siteHeader');
    const mobileToggle = document.getElementById('mobileNavToggle');
    if (siteHeader?.classList) {
        siteHeader.classList.remove('is-mobile-open');
    }
    mobileToggle?.setAttribute?.('aria-expanded', 'false');
}

// API Fetch Helper
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

// Floating Ant Message Toast (falls back to #app if #toastContainer absent)
function showAlert(message, type) {
    const safeType = ['success', 'error', 'info'].includes(type) ? type : 'info';
    const toastContainer = document.getElementById('toastContainer');
    if (toastContainer) {
        const msgEl = document.createElement('div');
        msgEl.className = `ant-message ant-message-${safeType}`;
        msgEl.textContent = message;
        toastContainer.appendChild(msgEl);
        setTimeout(() => msgEl.remove(), 4500);
        return;
    }

    const app = document.getElementById('app');
    if (!app) return;
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${safeType}`;
    alertDiv.textContent = message;
    app.prepend(alertDiv);
    setTimeout(() => alertDiv.remove(), 4500);
}

// Health Check Polling
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

// Navigation State
function updateNav() {
    const navAuth = document.getElementById('navAuth');
    const navLinks = document.getElementById('navLinks');
    if (!navAuth || !navLinks) return;

    const { path: activePath } = parseRoute();
    navAuth.innerHTML = '';
    navLinks.innerHTML = '';

    const addNavLink = (container, text, href, routeMatch, className = '') => {
        const a = document.createElement('a');
        a.textContent = text;
        a.href = href;
        const isCurrent = routeMatch && activePath === routeMatch;
        if (className || isCurrent) {
            a.className = [className, isCurrent ? 'is-active' : ''].filter(Boolean).join(' ');
        }
        if (isCurrent) {
            a.setAttribute?.('aria-current', 'page');
        }
        container.appendChild(a);
    };

    if (currentUser) {
        addNavLink(navLinks, 'Beranda', '#/', '/');
        addNavLink(navLinks, 'Direktori', '#/directory', '/directory');
        addNavLink(navLinks, 'Forum Q&A', '#/forum', '/forum');
        addNavLink(navLinks, 'Subgroup', '#/subgroups', '/subgroups');
        addNavLink(navLinks, 'Profil', '#/profile', '/profile');
        if (currentUser.status !== 'verified') {
            addNavLink(navLinks, 'Verifikasi', '#/verify', '/verify');
        }

        const logoutBtn = document.createElement('button');
        logoutBtn.id = 'logoutBtn';
        logoutBtn.type = 'button';
        logoutBtn.className = 'btn btn-secondary btn-sm';
        logoutBtn.textContent = 'Keluar';
        logoutBtn.addEventListener('click', logout);
        navAuth.appendChild(logoutBtn);
    } else {
        addNavLink(navLinks, 'Beranda', '#/', '/');
        addNavLink(navLinks, 'Forum Q&A', '#/forum', '/forum');
        addNavLink(navAuth, 'Login', '#/login', '/login', 'btn btn-secondary btn-sm');
        addNavLink(navAuth, 'Daftar', '#/register', '/register', 'btn btn-primary btn-sm');
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
    closeMobileNav();
    const { path, params } = parseRoute();
    const app = document.getElementById('app');
    if (!app) return;
    app.innerHTML = '';
    updateNav();

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
        case '/oauth/authorize':
            renderOAuthAuthorize(params);
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

// =========================================================================
// 1. LANDING PAGE (#/)
// =========================================================================
function renderHome() {
    const app = document.getElementById('app');
    const heroActions = currentUser
        ? `
            <a href="#/directory" class="btn btn-primary btn-lg">Jelajahi Direktori</a>
            <a href="#/forum" class="btn btn-secondary btn-lg">Forum Q&amp;A</a>
            <a href="#/profile" class="btn btn-secondary btn-lg">Profil Saya</a>
        `
        : `
            <a href="#/register" class="btn btn-primary btn-lg">Gabung Sekarang</a>
            <a href="#/forum" class="btn btn-secondary btn-lg">Lihat Forum Q&amp;A</a>
            <a href="#/login" class="btn btn-secondary btn-lg">Masuk Akun</a>
        `;

    const featuredWilayah = ['Jabodetabek', 'Bandung', 'Surabaya', 'Yogyakarta', 'Medan', 'Semarang', 'Makassar', 'Denpasar', 'Balikpapan'];
    const regionChipsHtml = featuredWilayah.map(w => `<a href="#/subgroups" class="region-chip">${esc(w)}</a>`).join('');

    app.innerHTML = `
        <div class="landing-shell">
            <section class="landing-hero anim-fade-up" aria-labelledby="heroTitle">
                <span class="hero-orb hero-orb-1" aria-hidden="true"></span>
                <span class="hero-orb hero-orb-2" aria-hidden="true"></span>
                <div class="hero-grid">
                    <div>
                        <span class="section-eyebrow">Ikatan Alumni Resmi • National Taiwan University of Science and Technology</span>
                        <h1 id="heroTitle" class="hero-title">
                            Terhubung, Berbagi Pengetahuan, &amp; <span class="hero-title-highlight">Tumbuh Bersama</span> Alumni NTUST Indonesia
                        </h1>
                        <p class="hero-subtitle">
                            Portal terintegrasi untuk direktori alumni terverifikasi, forum tanya-jawab interaktif berbasis Apache Answer SSO, serta koordinasi ikatan alumni di berbagai wilayah Indonesia.
                        </p>
                        <div class="hero-actions">
                            ${heroActions}
                        </div>
                    </div>

                    <div class="hero-visual" aria-label="Sorotan Fitur Platform">
                        <article class="hero-preview-card">
                            <span class="feature-icon-box">${ICONS.shield}</span>
                            <div>
                                <div class="preview-card-title">Identitas Alumni Terverifikasi</div>
                                <p class="preview-card-desc">Validasi domain <code>@mail.ntust.edu.tw</code> atau OTP SMS/WhatsApp untuk menjaga keamanan direktori anggota.</p>
                            </div>
                        </article>
                        <article class="hero-preview-card">
                            <span class="feature-icon-box">${ICONS.forum}</span>
                            <div>
                                <div class="preview-card-title">Forum Q&amp;A + OAuth2 SSO</div>
                                <p class="preview-card-desc">Ajukan pertanyaan teknis, karier, dan riset langsung di Apache Answer dengan sekali klik SSO terverifikasi.</p>
                            </div>
                        </article>
                        <article class="hero-preview-card">
                            <span class="feature-icon-box">${ICONS.map}</span>
                            <div>
                                <div class="preview-card-title">Jejaring Wilayah Indonesia</div>
                                <p class="preview-card-desc">Kolaborasi lintas angkatan &amp; jurusan bersama pengurus chapter Jabodetabek hingga Indonesia Timur.</p>
                            </div>
                        </article>
                    </div>
                </div>
            </section>

            <section class="ant-grid-4 anim-fade-up anim-delay-1" aria-label="Ringkasan Platform">
                <article class="ant-stat-card">
                    <div class="ant-stat-label">Verifikasi Akun</div>
                    <div class="ant-stat-value">3 Kanal</div>
                    <div class="ant-stat-hint">Email NTUST, SMS &amp; WhatsApp OTP</div>
                </article>
                <article class="ant-stat-card">
                    <div class="ant-stat-label">Platform Diskusi</div>
                    <div class="ant-stat-value">Q&amp;A SSO</div>
                    <div class="ant-stat-hint">Terintegrasi Apache Answer OAuth2</div>
                </article>
                <article class="ant-stat-card">
                    <div class="ant-stat-label">Cakupan Chapter</div>
                    <div class="ant-stat-value">10+ Wilayah</div>
                    <div class="ant-stat-hint">Koordinasi regional seluruh Indonesia</div>
                </article>
                <article class="ant-stat-card">
                    <div class="ant-stat-label">Privasi &amp; Keamanan</div>
                    <div class="ant-stat-value">Terjaga</div>
                    <div class="ant-stat-hint">Direktori khusus alumni terverifikasi</div>
                </article>
            </section>

            <section class="anim-fade-up anim-delay-2" aria-labelledby="featuresHeading">
                <header class="section-heading-block">
                    <span class="section-eyebrow">Ekosistem Digital Alumni</span>
                    <h2 id="featuresHeading" class="section-title">Semua Kebutuhan Komunitas dalam Satu Portal</h2>
                    <p class="subtitle">Dirancang dengan standar Ant Design untuk navigasi yang ringkas, cepat, dan nyaman di desktop maupun ponsel.</p>
                </header>
                <div class="ant-grid-4">
                    <article class="feature-card">
                        <span class="feature-icon-box">${ICONS.users}</span>
                        <h3>Direktori Terfilter</h3>
                        <p>Temukan sesama lulusan NTUST berdasarkan wilayah, jurusan, perusahaan, maupun bidang keahlian secara cepat.</p>
                        <a href="#/directory" class="btn-link">Buka Direktori</a>
                    </article>
                    <article class="feature-card">
                        <span class="feature-icon-box">${ICONS.forum}</span>
                        <h3>Forum Q&amp;A Apache Answer</h3>
                        <p>Diskusi berbasis tag, voting jawaban terbaik, serta masuk otomatis menggunakan OAuth2 SSO portal alumni.</p>
                        <a href="#/forum" class="btn-link">Jelajahi Q&amp;A</a>
                    </article>
                    <article class="feature-card">
                        <span class="feature-icon-box">${ICONS.map}</span>
                        <h3>Ikatan Alumni Wilayah</h3>
                        <p>Lihat kepengurusan subgroup daerah, admin wilayah, dan perluas kolaborasi profesional terdekat Anda.</p>
                        <a href="#/subgroups" class="btn-link">Lihat Subgroup</a>
                    </article>
                    <article class="feature-card">
                        <span class="feature-icon-box">${ICONS.shield}</span>
                        <h3>Verifikasi Multi-Channel</h3>
                        <p>Aktivasi status <em>verified</em> lewat tautan unik email kampus <code>@mail.ntust.edu.tw</code> atau OTP instan.</p>
                        <a href="#/verify" class="btn-link">Pelajari Verifikasi</a>
                    </article>
                </div>
            </section>

            <section class="ant-steps-card anim-fade-up anim-delay-3" aria-labelledby="stepsHeading">
                <header class="section-heading-block">
                    <span class="section-eyebrow">Alur Bergabung</span>
                    <h2 id="stepsHeading" class="section-title">Aktif dalam 3 Langkah Mudah</h2>
                </header>
                <div class="ant-steps">
                    <div class="ant-step-item">
                        <span class="ant-step-num">1</span>
                        <div>
                            <div class="ant-step-title">Daftar Akun Portal</div>
                            <p class="ant-step-desc">Lengkapi nama, email aktif, NIM, jurusan, tahun lulus, serta domisili wilayah Indonesia.</p>
                        </div>
                    </div>
                    <div class="ant-step-item">
                        <span class="ant-step-num">2</span>
                        <div>
                            <div class="ant-step-title">Verifikasi Identitas</div>
                            <p class="ant-step-desc">Konfirmasi kepemilikan akun via tautan &amp; OTP email <code>@mail.ntust.edu.tw</code>, SMS, atau WhatsApp.</p>
                        </div>
                    </div>
                    <div class="ant-step-item">
                        <span class="ant-step-num">3</span>
                        <div>
                            <div class="ant-step-title">Akses Direktori &amp; Q&amp;A SSO</div>
                            <p class="ant-step-desc">Masuk otomatis ke Forum Q&amp;A Apache Answer, lengkapi riwayat karier, dan terhubung dengan jaringan alumni.</p>
                        </div>
                    </div>
                </div>
            </section>

            <section class="anim-fade-up anim-delay-4" aria-labelledby="regionHeading">
                <header class="section-heading-block">
                    <h2 id="regionHeading" class="section-title">Jaringan Alumni di Seluruh Indonesia</h2>
                    <p class="subtitle">Pilih wilayah untuk melihat struktur pengurus dan komunitas lokal Anda.</p>
                </header>
                <div class="region-chip-cloud">${regionChipsHtml}</div>
            </section>

            <section class="landing-cta anim-fade-up anim-delay-4" aria-label="Ajakan Bergabung">
                <div>
                    <h2>${currentUser ? 'Lanjutkan Kolaborasi di Komunitas NTUST' : 'Siap Terhubung dengan Sesama Alumni NTUST?'}</h2>
                    <p>${currentUser ? 'Perbarui profil karier Anda atau diskusikan topik terbaru bersama alumni terverifikasi di Forum Q&A.' : 'Daftarkan akun Anda sekarang dan selesaikan verifikasi untuk membuka seluruh fitur Direktori dan Forum Q&A.'}</p>
                </div>
                <div class="hero-actions">
                    ${currentUser
                        ? '<a href="#/forum" class="btn btn-lg landing-cta-btn">Masuk Forum Q&amp;A</a>'
                        : '<a href="#/register" class="btn btn-lg landing-cta-btn">Daftar Alumni Baru</a>'}
                </div>
            </section>
        </div>
    `;
}

// =========================================================================
// 2. AUTH VIEWS (LOGIN & REGISTER)
// =========================================================================
function renderLogin() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <section class="ant-container-narrow">
            <div class="card">
                <h2 class="card-title">Masuk ke Portal Alumni</h2>
                <p class="subtitle">Gunakan email dan password akun Alumni NTUST Indonesia Anda</p>
                <div id="loginError" class="ant-mt-4"></div>
                <form id="loginForm" class="ant-mt-4">
                    <div class="form-group">
                        <label for="loginEmail">Email</label>
                        <input type="email" id="loginEmail" name="email" required placeholder="nama@domain.com">
                    </div>
                    <div class="form-group">
                        <label for="loginPassword">Password</label>
                        <input type="password" id="loginPassword" name="password" required placeholder="Masukkan password">
                    </div>
                    <button type="submit" class="btn btn-primary btn-block">Login</button>
                </form>
                <div class="auth-footer-links">
                    <span>Belum punya akun? <a href="#/register" class="btn-link">Daftar sebagai alumni baru</a></span>
                    <a href="#/verify" class="btn-link">Ingin memverifikasi akun?</a>
                </div>
            </div>
        </section>
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
        <section class="ant-container-narrow">
            <div class="card">
                <h2 class="card-title">Daftar Alumni Baru</h2>
                <p class="subtitle">Status awal akun adalah <strong>Pending</strong> hingga verifikasi identitas alumni selesai.</p>
                <div id="registerError" class="ant-mt-4"></div>
                <form id="registerForm" class="ant-mt-4">
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
                        <input type="password" id="regPassword" name="password" required minlength="6" maxlength="72" placeholder="6 - 72 karakter">
                    </div>
                    <div class="ant-grid-2">
                        <div class="form-group">
                            <label for="regNim">NIM NTUST</label>
                            <input type="text" id="regNim" name="nim" placeholder="Mis. M11015001">
                        </div>
                        <div class="form-group">
                            <label for="regTahunLulus">Tahun Lulus</label>
                            <input type="number" id="regTahunLulus" name="tahun_lulus" min="1990" max="2035" placeholder="2020">
                        </div>
                    </div>
                    <div class="ant-grid-2">
                        <div class="form-group">
                            <label for="regFakultas">Jurusan / Fakultas</label>
                            <input type="text" id="regFakultas" name="fakultas" placeholder="Contoh: CSIE">
                        </div>
                        <div class="form-group">
                            <label for="regWilayah">Wilayah</label>
                            <select id="regWilayah" name="wilayah">
                                <option value="">-- Pilih Wilayah --</option>
                                ${wilayahOpts}
                            </select>
                        </div>
                    </div>
                    <div class="form-group">
                        <label for="regTelepon">No. Telepon / WhatsApp</label>
                        <input type="text" id="regTelepon" name="telepon" placeholder="08123456789">
                    </div>
                    <button type="submit" class="btn btn-primary btn-block">Daftar Sekarang</button>
                </form>
                <div class="auth-footer-links">
                    <span>Untuk akses Direktori &amp; Forum Q&amp;A, lanjutkan verifikasi dengan <code>@mail.ntust.edu.tw</code> atau SMS/WhatsApp.</span>
                    <span>Sudah punya akun? <a href="#/login" class="btn-link">Login di sini</a></span>
                </div>
            </div>
        </section>
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

// =========================================================================
// 3. VERIFICATION FLOW (EMAIL NTUST / SMS / WHATSAPP + OTP CONFIRMATION)
// =========================================================================
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
            <p class="subtitle">Masukkan email <code>@mail.ntust.edu.tw</code> Anda. Sistem akan mengirimkan tautan konfirmasi unik (berlaku 15 menit) beserta kode OTP 6 digit dari <code>ntust-alumni-verification@cerdas.club</code>.</p>
            <form id="verifyEmailRequestForm" class="ant-mt-4">
                <div class="form-group">
                    <label for="verifyEmail">Email <code>@mail.ntust.edu.tw</code></label>
                    <input type="email" id="verifyEmail" name="email_ntust" required value="${esc(defaultEmail)}" placeholder="nama@mail.ntust.edu.tw">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Kirim Tautan &amp; OTP ke Email</button>
            </form>
            <div id="verifyRequestStatus" class="ant-mt-4"></div>
        `;
    } else if (verifyPhoneStep === 'confirm') {
        const normalized = formatPhonePreview(verifyPhoneDraft);
        bodyHtml = `
            <div class="confirm-box">
                <h4>Konfirmasi Pengiriman OTP (${esc(channelLabel(verifyChannel))})</h4>
                <p class="ant-mt-2">Kode OTP 6 digit akan dikirimkan melalui <strong>${esc(channelLabel(verifyChannel))}</strong> ke nomor:</p>
                <p class="confirm-phone-target">+${esc(normalized)}</p>
                <p class="subtitle">Kode OTP berlaku selama 15 menit. Pastikan nomor aktif dan dapat menerima pesan.</p>
                <div class="ant-row ant-mt-4">
                    <button type="button" id="confirmSendOtpBtn" class="btn btn-primary">Kirim OTP Sekarang</button>
                    <button type="button" id="changePhoneBtn" class="btn btn-secondary">Ubah Nomor</button>
                </div>
            </div>
            <div id="verifyRequestStatus" class="ant-mt-4"></div>
        `;
    } else {
        const defaultPhone = verifyPhoneDraft || (verifyChannel === 'whatsapp' ? (currentUser?.no_wa || currentUser?.no_telp || '') : (currentUser?.no_telp || currentUser?.no_wa || ''));
        bodyHtml = `
            <p class="subtitle">Masukkan nomor HP/WhatsApp aktif untuk menerima kode OTP verifikasi melalui <strong>${esc(channelLabel(verifyChannel))}</strong>.</p>
            <form id="verifyPhoneForm" class="ant-mt-4">
                <div class="form-group">
                    <label for="verifyPhoneInput">Nomor ${esc(channelLabel(verifyChannel))}</label>
                    <input type="tel" id="verifyPhoneInput" name="phone" required value="${esc(defaultPhone)}" placeholder="08123456789 atau 628123456789">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Lanjut ke Konfirmasi</button>
            </form>
            <div id="verifyRequestStatus" class="ant-mt-4"></div>
        `;
    }

    app.innerHTML = `
        <section class="ant-container-narrow">
            <div class="card">
                <h2 class="card-title">Verifikasi Akun Alumni</h2>
                <div class="channel-tabs" id="verifyChannelTabs">
                    <button type="button" data-channel="email" class="channel-tab ${verifyChannel === 'email' ? 'active' : ''}">Email NTUST</button>
                    <button type="button" data-channel="sms" class="channel-tab ${verifyChannel === 'sms' ? 'active' : ''}">SMS</button>
                    <button type="button" data-channel="whatsapp" class="channel-tab ${verifyChannel === 'whatsapp' ? 'active' : ''}">WhatsApp</button>
                </div>
                ${bodyHtml}
            </div>
        </section>
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
                getVerifyStorage().setItem('verifyToken', vToken);
            }
            const targetMasked = data.data?.target || email;
            const confirmHref = `#/verify/confirm?token=${encodeURIComponent(vToken)}&channel=email`;
            statusEl.innerHTML = `
                <div class="alert alert-success">
                    <div>
                        Tautan verifikasi unik dan kode OTP 6 digit telah dikirim ke <strong>${esc(targetMasked)}</strong> (berlaku 15 menit). Silakan buka inbox email Anda dan klik tautan tersebut untuk memasukkan kode OTP.
                        ${vToken ? `<div class="ant-mt-2"><a href="${esc(confirmHref)}" class="btn-link">Sudah menerima OTP? Masukkan OTP di sini</a></div>` : ''}
                    </div>
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
                getVerifyStorage().setItem('verifyToken', vToken);
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
    const vStore = getVerifyStorage();
    if (paramToken) {
        vStore.setItem('verifyToken', paramToken);
        if (typeof window.history?.replaceState === 'function') {
            try {
                const cleanUrl = paramChannel ? `#/verify/confirm?channel=${encodeURIComponent(paramChannel)}` : '#/verify/confirm';
                window.history.replaceState(null, '', cleanUrl);
            } catch (_) {}
        }
    }
    const activeToken = paramToken || vStore.getItem('verifyToken') || '';
    const app = document.getElementById('app');

    if (!activeToken) {
        app.innerHTML = `
            <section class="ant-container-narrow">
                <div class="card">
                    <h2 class="card-title">Konfirmasi Verifikasi</h2>
                    <div class="alert alert-error ant-mt-4">Tautan atau sesi verifikasi tidak ditemukan.</div>
                    <a href="#/verify" class="btn btn-primary">Minta Verifikasi Ulang</a>
                </div>
            </section>
        `;
        return;
    }

    app.innerHTML = `
        <section class="ant-container-narrow">
            <div class="card">
                <h2 class="card-title">Konfirmasi Kode OTP</h2>
                <div id="verifySessionInfo" class="ant-mt-4"><div class="empty-state">Memeriksa tautan verifikasi...</div></div>
            </div>
        </section>
    `;

    try {
        const stRes = await apiFetch('/verify/status?token=' + encodeURIComponent(activeToken));
        const stData = await stRes.json();
        const infoEl = document.getElementById('verifySessionInfo');
        if (!infoEl) return;

        if (!stRes.ok || !stData.data?.valid) {
            vStore.removeItem('verifyToken');
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
            <div class="confirm-box ant-mb-4">
                <span class="badge badge-admin">${esc(channelLabel(ch))}</span>
                <p class="ant-mt-2">Tujuan: <strong>${esc(target)}</strong></p>
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
                <div class="ant-row">
                    <button type="submit" class="btn btn-primary">Verifikasi Akun</button>
                    <a href="#/verify" class="btn btn-secondary">Ganti Metode / Kirim Ulang</a>
                </div>
            </form>
            <div id="verifyOtpStatus" class="ant-mt-4"></div>
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
    const vStore = getVerifyStorage();
    const form = new FormData(e.target);
    const verifyToken = String(form.get('token') || vStore.getItem('verifyToken') || '').trim();
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
            vStore.removeItem('verifyToken');
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

// =========================================================================
// 4. PROFILE VIEW
// =========================================================================
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
                <span>Akun Anda masih <strong>Pending</strong>. Verifikasi akun lewat email <code>@mail.ntust.edu.tw</code> atau SMS/WhatsApp untuk mengakses Direktori &amp; Forum Q&amp;A SSO.</span>
                <a href="#/verify" class="btn btn-primary btn-sm">Verifikasi Sekarang</a>
           </div>`
        : '';

    app.innerHTML = `
        <section class="ant-container-medium">
            <div class="card">
                ${verifyBanner}
                <div class="profile-summary">
                    ${renderAvatarMarkup(u, 'avatar-preview')}
                    <div>
                        <h2>${esc(u.nama || '-')}</h2>
                        <p class="sub">${esc(u.email)} ${u.nim ? '• NIM: ' + esc(u.nim) : ''}</p>
                        <div class="ant-row">
                            <span class="badge badge-${u.status === 'verified' ? 'verified' : 'pending'}">${u.status === 'verified' ? 'Terverifikasi' : 'Pending'}</span>
                            <span class="badge badge-admin">${esc(u.role || 'member')}</span>
                        </div>
                    </div>
                </div>

                <h3 class="profile-section">Profil &amp; Kontak</h3>
                <form id="profileForm" class="profile-form-panel">
                    <div class="form-group">
                        <label for="pFotoFile">Foto Profil (PNG/JPG/WEBP/GIF, maks 2 MB)</label>
                        <div class="avatar-row">
                            <img id="profilePhotoPreview" class="avatar-preview ${pendingFotoProfil ? '' : 'is-hidden'}" src="${esc(pendingFotoProfil)}" alt="Preview Foto">
                            <input type="file" id="pFotoFile" accept="image/png,image/jpeg,image/webp,image/gif">
                        </div>
                    </div>

                    <div class="ant-grid-2">
                        <div class="form-group">
                            <label for="pNama">Nama Lengkap</label>
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
                            <label for="pWilayah">Wilayah</label>
                            <select id="pWilayah" name="wilayah">
                                <option value="">-- Pilih Wilayah --</option>
                                ${getWilayahOptions(u.wilayah)}
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="pNoTelp">No. Telepon</label>
                            <input type="text" id="pNoTelp" name="no_telp" value="${esc(u.no_telp || '')}">
                        </div>
                        <div class="form-group">
                            <label for="pNoWA">No. WhatsApp</label>
                            <input type="text" id="pNoWA" name="no_wa" value="${esc(u.no_wa || '')}">
                        </div>
                        <div class="form-group">
                            <label for="pTelegram">Telegram</label>
                            <input type="text" id="pTelegram" name="telegram" value="${esc(u.telegram || '')}">
                        </div>
                        <div class="form-group">
                            <label for="pLinkedIn">LinkedIn URL</label>
                            <input type="url" id="pLinkedIn" name="linkedin" placeholder="https://linkedin.com/in/..." value="${esc(u.linkedin || '')}">
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
                    </div>

                    <div class="form-group">
                        <label for="pBidangKeahlian">Bidang Keahlian (pisahkan dengan koma)</label>
                        <input type="text" id="pBidangKeahlian" name="bidang_keahlian" value="${esc((u.bidang_keahlian || []).join(', '))}" placeholder="AI, Cloud, Backend">
                    </div>

                    <div class="form-group">
                        <label>Akun Media Sosial</label>
                        <div class="sosmed-grid">
                            <input type="text" id="pFB" name="facebook" placeholder="Facebook" value="${esc(u.sosmed?.facebook || '')}">
                            <input type="text" id="pIG" name="instagram" placeholder="Instagram" value="${esc(u.sosmed?.instagram || '')}">
                            <input type="text" id="pTwitter" name="twitter" placeholder="Twitter/X" value="${esc(u.sosmed?.twitter || '')}">
                            <input type="text" id="pThreads" name="threads" placeholder="Threads" value="${esc(u.sosmed?.threads || '')}">
                        </div>
                    </div>

                    <button type="submit" id="saveProfileBtn" class="btn btn-primary">Simpan Profil</button>
                </form>

                <h3 class="profile-section">Pekerjaan</h3>
                <div class="edu-field" id="pekerjaanDisplay"></div>
                <button type="button" class="btn btn-secondary btn-sm" id="editPekerjaan">Edit Pekerjaan</button>

                <h3 class="profile-section">Pendidikan di NTUST</h3>
                <div class="edu-field" id="pendidikanDisplay"></div>
                <button type="button" class="btn btn-secondary btn-sm" id="editPendidikan">Edit Pendidikan NTUST</button>
            </div>
        </section>
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
            preview.classList?.remove('is-hidden');
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
        <form id="pekerjaanForm" class="ant-stack">
            <div class="ant-grid-2">
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
            </div>
            <div class="ant-row">
                <button type="submit" class="btn btn-primary btn-sm">Simpan Pekerjaan</button>
                <button type="button" id="cancelPekerjaan" class="btn btn-secondary btn-sm">Batal</button>
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
        <form id="pendidikanForm" class="ant-stack">
            <div class="ant-grid-2">
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
            </div>
            <div class="form-group">
                <label for="eduSupervisor">Supervisor</label><input type="text" id="eduSupervisor" name="supervisor" value="${esc(edu.supervisor || '')}">
            </div>
            <div class="ant-row">
                <button type="submit" class="btn btn-primary btn-sm">Simpan Pendidikan</button>
                <button type="button" id="cancelPendidikan" class="btn btn-secondary btn-sm">Batal</button>
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
    const linkedinRaw = String(form.get('linkedin') || '').trim();
    if (linkedinRaw && !safeUrl(linkedinRaw)) {
        showAlert('URL LinkedIn harus menggunakan http:// atau https:// yang valid', 'error');
        return;
    }
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
        linkedin: linkedinRaw,
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

// =========================================================================
// 5. VERIFIED ALUMNI DIRECTORY
// =========================================================================
async function renderDirectory() {
    const app = document.getElementById('app');

    if (!currentUser) {
        app.innerHTML = `
            <section class="ant-container-narrow">
                <div class="card">
                    <h2 class="card-title">Direktori Alumni</h2>
                    <p class="subtitle">Silakan login terlebih dahulu untuk mengakses direktori alumni.</p>
                    <div class="ant-mt-4"><a href="#/login" class="btn btn-primary">Login</a></div>
                </div>
            </section>
        `;
        return;
    }

    if (currentUser.status !== 'verified') {
        app.innerHTML = `
            <section class="ant-container-narrow">
                <div class="card">
                    <h2 class="card-title">Direktori Alumni</h2>
                    <p class="subtitle">Direktori alumni hanya dapat diakses oleh alumni yang telah terverifikasi email <code>@mail.ntust.edu.tw</code> atau OTP.</p>
                    <div class="ant-mt-4"><a href="#/verify" class="btn btn-primary">Verifikasi Akun</a></div>
                </div>
            </section>
        `;
        return;
    }

    app.innerHTML = `
        <div class="section-header">
            <div>
                <h3>Direktori Alumni</h3>
                <span id="dirSummary" class="subtitle">Alumni Terverifikasi</span>
            </div>
        </div>
        <form id="dirFilterForm" class="directory-controls">
            <input type="text" class="ant-input" id="dirSearchInput" name="q" value="${esc(dirState.q)}" placeholder="Cari nama, NIM, perusahaan, jabatan...">
            <input type="text" class="ant-input" id="dirJurusanInput" name="jurusan" value="${esc(dirState.jurusan)}" placeholder="Filter jurusan (mis. CSIE)">
            <select class="ant-select" id="dirSortSelect" name="sort">
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
    listEl.innerHTML = '<div class="empty-state">Memuat direktori...</div>';
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
            const card = document.createElement('article');
            card.className = 'alumni-card';
            card.innerHTML = `
                <div class="alumni-card-top">
                    ${renderAvatarMarkup(a, 'avatar-thumb')}
                    <div>
                        <h4 class="alumni-card-name">${esc(a.nama || '-')}</h4>
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
        <section class="ant-container-medium">
            <div class="card">
                <div class="profile-summary">
                    ${renderAvatarMarkup(alumni, 'avatar-preview')}
                    <div>
                        <h2 class="card-title">${esc(alumni.nama || '-')}</h2>
                        <p class="subtitle">${esc(alumni.wilayah || '-')} • ${esc(alumni.role || 'member')}</p>
                    </div>
                </div>
                <div class="sosmed-list ant-mt-4">
                    ${alumni.email ? `<span class="badge badge-verified">Email: ${esc(alumni.email)}</span>` : ''}
                    ${alumni.no_wa ? `<span class="badge badge-verified">WA: ${esc(alumni.no_wa)}</span>` : ''}
                    ${alumni.telegram ? `<span class="badge badge-admin">Telegram: ${esc(alumni.telegram)}</span>` : ''}
                    ${linkedinHref ? `<a href="${esc(linkedinHref)}" target="_blank" rel="noopener noreferrer" class="btn-link">Profil LinkedIn</a>` : ''}
                </div>
                <div class="alumni-detail-grid">
                    <div class="detail-panel">
                        <p><strong>Wilayah:</strong> ${esc(alumni.wilayah || '-')}</p>
                        <p><strong>NIM:</strong> ${esc(alumni.nim || '-')}</p>
                        <p><strong>Bidang Keahlian:</strong> ${esc((alumni.bidang_keahlian || []).join(', ') || '-')}</p>
                    </div>
                    <div class="detail-panel">
                        <p><strong>Pekerjaan:</strong> ${pekerjaanText}</p>
                        <p><strong>Pendidikan NTUST:</strong> ${pendidikanText}</p>
                    </div>
                </div>
                <button type="button" id="backToDirBtn" class="btn btn-secondary">Kembali ke Direktori</button>
            </div>
        </section>
    `;
    document.getElementById('backToDirBtn').addEventListener('click', () => navigate('/directory'));
}

// =========================================================================
// 6. FORUM Q&A (APACHE ANSWER INTEGRATION + OAUTH2 SSO)
// =========================================================================
let qaOrder = 'newest';

async function getAnswerSsoUrl() {
    try {
        const res = await fetch(ANSWER_API_BASE + '/connector/info');
        if (res.ok) {
            const payload = await res.json();
            const connectors = payload.data || [];
            if (connectors.length > 0 && connectors[0].link) {
                const expectedOrigin = new URL(ANSWER_URL).origin;
                const resolved = new URL(connectors[0].link, ANSWER_URL);
                if (resolved.origin === expectedOrigin) {
                    return resolved.href;
                }
            }
        }
    } catch (_) {}
    return ANSWER_URL + '/users/login';
}

async function renderForum() {
    const app = document.getElementById('app');
    const isVerified = currentUser && currentUser.status === 'verified';

    app.innerHTML = `
        <div class="section-header">
            <div>
                <h3>Forum Q&amp;A Alumni NTUST</h3>
                <p class="subtitle">Ditenagai oleh Apache Answer • Tanya jawab, voting, tag, &amp; solusi terkurasi</p>
            </div>
            <div class="qa-toolbar">
                <a href="${esc(ANSWER_URL)}" target="_blank" rel="noopener noreferrer" class="btn btn-secondary">Buka Forum Q&amp;A</a>
                ${isVerified ? `<a id="answerSsoBtn" href="${esc(ANSWER_URL + '/users/login')}" target="_blank" rel="noopener noreferrer" class="btn btn-primary">Login SSO ke Q&amp;A</a>` : ''}
            </div>
        </div>
        <div id="forumHint"></div>
        <div class="channel-tabs" id="qaOrderTabs">
            <button type="button" data-order="newest" class="channel-tab ${qaOrder === 'newest' ? 'active' : ''}">Terbaru</button>
            <button type="button" data-order="active" class="channel-tab ${qaOrder === 'active' ? 'active' : ''}">Aktif</button>
            <button type="button" data-order="unanswered" class="channel-tab ${qaOrder === 'unanswered' ? 'active' : ''}">Belum Terjawab</button>
        </div>
        <div id="forumList"></div>
    `;

    const hintEl = document.getElementById('forumHint');
    if (hintEl) {
        if (!currentUser) {
            hintEl.innerHTML = '<div class="alert alert-info"><span>Login dan verifikasi akun Anda untuk masuk otomatis (SSO) dan mengajukan pertanyaan di Q&amp;A.</span><a href="#/login" class="btn-link">Login</a></div>';
        } else if (!isVerified) {
            hintEl.innerHTML = '<div class="alert alert-info"><span>Verifikasi akun Anda terlebih dahulu agar dapat login SSO ke Forum Q&amp;A.</span><a href="#/verify" class="btn-link">Verifikasi Sekarang</a></div>';
        }
    }

    const tabsEl = document.getElementById('qaOrderTabs');
    if (tabsEl && tabsEl.children) {
        Array.from(tabsEl.children).forEach(btn => {
            btn.addEventListener('click', () => {
                qaOrder = btn.dataset?.order || btn.getAttribute?.('data-order') || 'newest';
                renderForum();
            });
        });
    }

    if (isVerified) {
        getAnswerSsoUrl().then(ssoHref => {
            const ssoBtn = document.getElementById('answerSsoBtn');
            if (ssoBtn) ssoBtn.href = ssoHref;
        });
    }

    await loadAnswerQuestions();
}

async function loadAnswerQuestions() {
    const listEl = document.getElementById('forumList');
    if (!listEl) return;
    listEl.innerHTML = '<div class="empty-state">Memuat pertanyaan Q&amp;A...</div>';

    try {
        const qs = new URLSearchParams({
            page: '1',
            page_size: '15',
            order: qaOrder
        });
        const res = await fetch(ANSWER_API_BASE + '/question/page?' + qs.toString());
        const payload = await res.json();

        if (!res.ok) {
            listEl.innerHTML = '<div class="empty-state">Forum Q&amp;A sedang tidak dapat diakses</div>';
            return;
        }

        const questions = payload.data?.list || [];
        if (questions.length === 0) {
            listEl.innerHTML = `<div class="empty-state">Belum ada pertanyaan Q&amp;A. <a href="${esc(ANSWER_URL + '/questions/ask')}" target="_blank" rel="noopener noreferrer" class="btn-link">Ajukan pertanyaan pertama di Apache Answer</a></div>`;
            return;
        }

        listEl.innerHTML = '';
        questions.forEach(q => {
            const qId = encodeURIComponent(q.id || '');
            const qUrl = `${ANSWER_URL}/questions/${qId}`;
            const tags = Array.isArray(q.tags) ? q.tags : [];
            const tagsHtml = tags.map(t => `<span class="qa-tag">${esc(t.display_name || t.slug_name || '')}</span>`).join('');
            const isAccepted = Boolean(q.accepted_answer_id && q.accepted_answer_id !== '0');
            const author = q.operator?.display_name || q.user_info?.display_name || 'Alumni';
            const createdVal = typeof q.created_at === 'number' ? new Date(q.created_at * 1000).toISOString() : q.created_at;

            const card = document.createElement('article');
            card.className = 'qa-card';
            card.innerHTML = `
                <div class="qa-stats">
                    <div class="qa-stat"><strong>${esc(q.vote_count ?? 0)}</strong><span>vote</span></div>
                    <div class="qa-stat ${isAccepted ? 'answered' : ''}"><strong>${esc(q.answer_count ?? 0)}</strong><span>jawaban</span></div>
                    <div class="qa-stat"><strong>${esc(q.view_count ?? 0)}</strong><span>lihat</span></div>
                </div>
                <div class="qa-main">
                    <h4><a href="${esc(qUrl)}" target="_blank" rel="noopener noreferrer">${esc(q.title || '-')}</a></h4>
                    ${tagsHtml ? `<div class="qa-tags">${tagsHtml}</div>` : ''}
                    <div class="qa-meta">Ditanyakan oleh <strong>${esc(author)}</strong> • ${esc(formatDate(createdVal))}</div>
                </div>
            `;
            listEl.appendChild(card);
        });
    } catch (_) {
        listEl.innerHTML = '<div class="empty-state">Gagal memuat pertanyaan dari Apache Answer</div>';
    }
}

async function renderOAuthAuthorize(params) {
    const app = document.getElementById('app');
    const clientId = params ? (params.get('client_id') || '') : '';
    const redirectUri = params ? (params.get('redirect_uri') || '') : '';
    const state = params ? (params.get('state') || '') : '';
    const responseType = params ? (params.get('response_type') || 'code') : 'code';
    const scope = params ? (params.get('scope') || 'openid profile email') : 'openid profile email';

    if (!clientId || !redirectUri) {
        app.innerHTML = `
            <section class="ant-container-narrow">
                <div class="card">
                    <h2 class="card-title">SSO OAuth2</h2>
                    <div class="alert alert-error ant-mt-4">Parameter otorisasi SSO tidak lengkap.</div>
                    <a href="#/forum" class="btn btn-primary">Kembali ke Forum Q&amp;A</a>
                </div>
            </section>
        `;
        return;
    }

    const safeRedirect = safeUrl(redirectUri);
    const expectedOrigin = new URL(ANSWER_URL).origin;
    if (!safeRedirect || new URL(safeRedirect).origin !== expectedOrigin) {
        app.innerHTML = `
            <section class="ant-container-narrow">
                <div class="card">
                    <h2 class="card-title">SSO OAuth2</h2>
                    <div class="alert alert-error ant-mt-4">URL redirect tujuan SSO tidak diizinkan.</div>
                    <a href="#/forum" class="btn btn-primary">Kembali ke Forum Q&amp;A</a>
                </div>
            </section>
        `;
        return;
    }

    if (!currentUser) {
        app.innerHTML = `
            <section class="ant-container-narrow">
                <div class="card">
                    <h2 class="card-title">Login SSO Forum Q&amp;A</h2>
                    <p class="subtitle">Masuk dengan akun Alumni NTUST Indonesia untuk melanjutkan ke Forum Q&amp;A Apache Answer.</p>
                    <div id="ssoLoginError" class="ant-mt-3"></div>
                    <form id="ssoLoginForm" class="ant-mt-4">
                        <div class="form-group">
                            <label for="ssoEmail">Email</label>
                            <input type="email" id="ssoEmail" name="email" required placeholder="nama@domain.com">
                        </div>
                        <div class="form-group">
                            <label for="ssoPassword">Password</label>
                            <input type="password" id="ssoPassword" name="password" required placeholder="Password">
                        </div>
                        <button type="submit" class="btn btn-primary btn-block">Login &amp; Lanjutkan ke Q&amp;A</button>
                    </form>
                </div>
            </section>
        `;
        document.getElementById('ssoLoginForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = new FormData(e.target);
            const errEl = document.getElementById('ssoLoginError');
            errEl.innerHTML = '';
            try {
                const res = await apiFetch('/login', {
                    method: 'POST',
                    body: JSON.stringify({
                        email: String(form.get('email') || '').trim(),
                        password: String(form.get('password') || '')
                    })
                });
                const data = await res.json();
                if (!res.ok) {
                    errEl.innerHTML = `<div class="alert alert-error">${esc(data.error || 'Login gagal')}</div>`;
                    return;
                }
                token = data.token;
                localStorage.setItem('token', token);
                const meRes = await apiFetch('/me');
                if (meRes.ok) {
                    currentUser = (await meRes.json()).data;
                }
                updateNav();
                renderOAuthAuthorize(params);
            } catch (_) {
                errEl.innerHTML = '<div class="alert alert-error">Gagal terhubung ke server</div>';
            }
        });
        return;
    }

    if (currentUser.status !== 'verified') {
        app.innerHTML = `
            <section class="ant-container-narrow">
                <div class="card">
                    <h2 class="card-title">Verifikasi Diperlukan</h2>
                    <div class="alert alert-info ant-mt-3">Akun Anda masih berstatus <strong>Pending</strong>. Hanya alumni terverifikasi yang dapat masuk ke Forum Q&amp;A melalui SSO.</div>
                    <a href="#/verify" class="btn btn-primary">Verifikasi Akun Sekarang</a>
                </div>
            </section>
        `;
        return;
    }

    app.innerHTML = `
        <section class="ant-container-narrow">
            <div class="card">
                <h2 class="card-title">Otorisasi Login SSO Forum Q&amp;A</h2>
                <p class="subtitle">Aplikasi <strong>Forum Q&amp;A Apache Answer</strong> (${esc(ANSWER_URL)}) meminta izin masuk menggunakan akun terverifikasi Anda:</p>
                <div class="confirm-box ant-mb-4">
                    <p><strong>Nama:</strong> ${esc(currentUser.nama)}</p>
                    <p><strong>Email:</strong> ${esc(currentUser.email)}</p>
                </div>
                <div id="oauthAuthStatus" class="ant-mb-4"></div>
                <div class="ant-row">
                    <button type="button" id="confirmOAuthBtn" class="btn btn-primary">Izinkan &amp; Masuk ke Forum Q&amp;A</button>
                    <a href="#/forum" class="btn btn-secondary">Batal</a>
                </div>
            </div>
        </section>
    `;

    const confirmBtn = document.getElementById('confirmOAuthBtn');
    if (confirmBtn) {
        confirmBtn.addEventListener('click', async () => {
            confirmBtn.disabled = true;
            const statusEl = document.getElementById('oauthAuthStatus');
            if (statusEl) statusEl.innerHTML = '<div class="empty-state">Mengalihkan ke Apache Answer...</div>';
            try {
                const res = await apiFetch('/oauth/authorize', {
                    method: 'POST',
                    body: JSON.stringify({
                        client_id: clientId,
                        redirect_uri: redirectUri,
                        state,
                        response_type: responseType,
                        scope
                    })
                });
                const data = await res.json();
                const targetUrl = safeUrl(data.data?.redirect_url);
                const expectedOrigin = new URL(ANSWER_URL).origin;
                if (res.ok && targetUrl && new URL(targetUrl).origin === expectedOrigin) {
                    window.location.href = targetUrl;
                } else {
                    confirmBtn.disabled = false;
                    if (statusEl) {
                        statusEl.innerHTML = `<div class="alert alert-error">${esc(data.error || 'Otorisasi SSO gagal')}</div>`;
                    }
                }
            } catch (_) {
                confirmBtn.disabled = false;
                if (statusEl) {
                    statusEl.innerHTML = '<div class="alert alert-error">Gagal menghubungi server otorisasi SSO</div>';
                }
            }
        });
    }
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleDateString('id-ID', {
        year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
    });
}

// =========================================================================
// 7. SUBGROUPS MANAGEMENT
// =========================================================================
async function renderSubgroups() {
    if (!currentUser) { navigate('/login'); return; }
    const isSuperAdmin = currentUser.role === 'super_admin';
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="section-header">
            <div>
                <h3>Subgroup Ikatan Alumni Wilayah</h3>
                <p class="subtitle">Struktur kepengurusan dan koordinator daerah Alumni NTUST di Indonesia</p>
            </div>
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
        <section class="ant-container-narrow">
            <div class="card">
                <h2 class="card-title">Buat Subgroup Baru</h2>
                <form id="subgroupForm" class="ant-mt-4">
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
                    <div class="ant-row">
                        <button type="submit" class="btn btn-primary">Simpan Subgroup</button>
                        <button type="button" id="cancelSubgroupBtn" class="btn btn-secondary">Batal</button>
                    </div>
                </form>
            </div>
        </section>
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
    listEl.innerHTML = '<div class="empty-state">Memuat subgroup wilayah...</div>';

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
            const card = document.createElement('article');
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
                ${s.deskripsi ? `<p class="subgroup-desc">${esc(s.deskripsi)}</p>` : ''}
                ${s.admin_id ? `<span class="badge badge-admin">Admin: ${esc(s.admin_id.substring(0, 12))}</span>` : ''}
                ${pengurusList}
            `;

            if (isAdmin) {
                const appointForm = document.createElement('form');
                appointForm.className = 'appoint-form';
                appointForm.innerHTML = `
                    <input type="text" name="alumni_id" class="appoint-input" required placeholder="ID Alumni">
                    <button type="submit" class="btn btn-primary btn-sm">Tunjuk Admin</button>
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
                actionsDiv.className = 'ant-mt-3';
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

// Bootstrap Application
window.addEventListener('hashchange', renderRoute);
initHeaderControls();
checkHealth();
if (typeof setInterval === 'function') {
    setInterval(checkHealth, 30000);
}
initUser();
