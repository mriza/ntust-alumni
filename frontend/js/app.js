// app.js - Alumni NTUST Indonesia SPA
// Konfigurasi
const API_BASE = (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1')
    ? 'http://localhost:8080/api'
    : '/api';

// State
let currentUser = null;
let token = localStorage.getItem('token');

// XSS protection: escape HTML special chars
function esc(str) {
    const d = document.createElement("div");
    d.textContent = String(str ?? "");
    return d.innerHTML;
}

// Helper functions
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
    return res;
}

function showAlert(message, type) {
    const app = document.getElementById('app');
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type}`;
    alertDiv.textContent = message;
    app.prepend(alertDiv);
    setTimeout(() => alertDiv.remove(), 5000);
}

function updateNav() {
    const navAuth = document.getElementById('navAuth');
    const navLinks = document.getElementById('navLinks');
    const app = document.getElementById('app');

    navAuth.innerHTML = '';
    navLinks.innerHTML = '';

    if (currentUser) {
        // Nav: Directory, Forum, Profile, Logout
        const directoryLink = document.createElement('a');
        directoryLink.textContent = 'Direktori';
        directoryLink.href = '#directory';
        navLinks.appendChild(directoryLink);

        const forumLink = document.createElement('a');
        forumLink.textContent = 'Forum';
        forumLink.href = '#forum';
        navLinks.appendChild(forumLink);

        const logoutBtn = document.createElement('button');
        logoutBtn.textContent = 'Keluar';
        logoutBtn.onclick = logout;
        navAuth.appendChild(logoutBtn);
    } else {
        const loginLink = document.createElement('a');
        loginLink.textContent = 'Login';
        loginLink.href = '#login';
        navAuth.appendChild(loginLink);

        const registerLink = document.createElement('a');
        registerLink.textContent = 'Daftar';
        registerLink.href = '#register';
        navAuth.appendChild(registerLink);
    }
}

async function initUser() {
    if (!token) { currentUser = null; updateNav(); return; }
    try {
        const res = await apiFetch('/me');
        if (res.ok) {
            const data = await res.json();
            currentUser = data.data;
        } else {
            token = null;
            localStorage.removeItem('token');
        }
    } catch (e) {
        token = null;
        localStorage.removeItem('token');
    }
    updateNav();
    renderRoute();
}

// Route handler
function renderRoute() {
    const hash = location.hash.substring(1) || '/';
    const app = document.getElementById('app');
    app.innerHTML = '';

    switch (hash) {
        case '/register':
            if (currentUser) { navigate('/profile'); return; }
            renderRegister();
            break;
        case '/login':
            if (currentUser) { navigate('/profile'); return; }
            renderLogin();
            break;
        case '/verify':
            renderVerifyPage();
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

function navigate(path) {
    location.hash = path;
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
    app.innerHTML = `
        <div class="card" style="text-align: center; padding: 60px 20px;">
            <h1>Alumni NTUST Indonesia</h1>
            <p style="margin: 20px 0; font-size: 18px;">Jaringan Alumni Teknik Informatika Universitas Negeri Tangerang Selatan</p>
            <div style="display: flex; gap: 12px; justify-content: center; flex-wrap: wrap;">
                <a href="#register" class="btn btn-primary">Daftar Alumni</a>
                <a href="#directory" class="btn btn-primary">Lihat Direktori</a>
                <a href="#login" class="btn btn-secondary">Login</a>
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
                    <label>Email</label>
                    <input type="email" id="loginEmail" required placeholder="nama@domain.com">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" id="loginPassword" required placeholder="******">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Login</button>
            </form>
            <p style="margin-top: 20px; text-align: center; font-size: 14px;">
                Belum punya akun? <a href="#register" class="btn-link">Daftar sebagai alumni baru</a>
            </p>
            <p style="margin-top: 10px; text-align: center;">
                <a href="#verify" class="btn-link">Ingin memverifikasi akun?</a>
            </p>
        </div>
    `;
    document.getElementById('loginForm').addEventListener('submit', handleLogin);
}

async function handleLogin(e) {
    e.preventDefault();
    const email = document.getElementById('loginEmail').value;
    const password = document.getElementById('loginPassword').value;
    const errorEl = document.getElementById('loginError');
    errorEl.innerHTML = '';

    const res = await apiFetch('/login', { method: 'POST', body: JSON.stringify({ email, password }) });
    const data = await res.json();

    if (res.ok) {
        token = data.token;
        localStorage.setItem('token', token);
        // Fetch user profile
        const meRes = await apiFetch('/me');
        currentUser = (await meRes.json()).data;
        updateNav();
        showAlert('Login berhasil', 'success');
        navigate('/profile');
    } else {
        errorEl.innerHTML = `<div class="alert alert-error">${data.error}</div>`;
    }
}

function renderRegister() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Daftar Alumni Baru</h2>
            <p class="subtitle">Daftarkan diri Anda sebagai alumni. Status Anda akan <strong>pending</strong> sampai verifikasi email selesai.</p>
            <div id="registerError"></div>
            <form id="registerForm">
                <div class="form-group">
                    <label>Nama Lengkap</label>
                    <input type="text" id="regNama" required placeholder="Nama Anda">
                </div>
                <div class="form-group">
                    <label>Email</label>
                    <input type="email" id="regEmail" required placeholder="nama@domain.com">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" id="regPassword" required placeholder="Minimal 6 karakter">
                </div>
                <div class="form-group">
                    <label>NIM</label>
                    <input type="text" id="regNim" placeholder="Nomor Induk Mahasiswa">
                </div>
                <div class="form-group">
                    <label>Fakultas</label>
                    <input type="text" id="regFakultas" placeholder="Teknik">
                </div>
                <div class="form-group">
                    <label>Tahun Lulus</label>
                    <input type="number" id="regTahunLulus" min="1990" max="2030" placeholder="2020">
                </div>
                <div class="form-group">
                    <label>Wilayah</label>
                    <select id="regWilayah">
                        <option value="">-- Pilih Wilayah --</option>
                        <option>Jakarta</option><option>Bandung</option><option>Surabaya</option>
                        <option>Medan</option><option>Yogyakarta</option><option>Semarang</option>
                        <option>Makassar</option><option>Balikpapan</option><option>Denpasar</option>
                        <option>Lainnya</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>No. Telepon</label>
                    <input type="text" id="regTelepon" placeholder="08123456789">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Daftar</button>
            </form>
            <p style="margin-top: 15px; text-align: center; font-size: 13px;">
                Untuk bisa bergabung di forum dan direktori, sertakan <strong>email @mail.ntust.edu.tw</strong> Anda di halaman verifikasi.
            </p>
        </div>
    `;
    document.getElementById('registerForm').addEventListener('submit', handleRegister);
}

async function handleRegister(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const data = {
        nama: form.get('nama'),
        email: form.get('email'),
        password: form.get('password'),
        nim: form.get('nim'),
        jurusan: form.get('fakultas'),
        tahun_lulus: parseInt(form.get('tahun_lulus')) || 0,
        wilayah: form.get('wilayah'),
        telepon: form.get('telepon'),
    };
    const errorEl = document.getElementById('registerError');
    errorEl.innerHTML = '';

    const res = await apiFetch('/register', { method: 'POST', body: JSON.stringify(data) });
    const resp = await res.json();

    if (res.ok || res.status === 201) {
        showAlert('Pendaftaran berhasil! Silakan login.', 'success');
        navigate('/login');
    } else {
        errorEl.innerHTML = `<div class="alert alert-error">${resp.error}</div>`;
    }
}

function renderVerifyPage() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Verifikasi Akun</h2>
            <form id="verifyRequestForm" style="margin-top: 20px;">
                <div class="form-group">
                    <label>Email <code style="font-size: 12px">@mail.ntust.edu.tw</code></label>
                    <input type="email" id="verifyEmail" required placeholder="nama@mail.ntust.edu.tw">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Kirim Kode OTP</button>
            </form>
            <div id="verifyRequestStatus" style="margin-top: 16px;"></div>

            <div class="divider">VERIFIKASI OTP</div>

            <form id="verifyOtpForm" style="margin-top: 20px;">
                <div class="form-group">
                    <label>Kode OTP (6 digit)</label>
                    <input type="text" id="verifyOtp" required placeholder="123456">
                </div>
                <div class="form-group">
                    <label>NIM</label>
                    <input type="text" id="verifyNim" required placeholder="Nomor Induk Mahasiswa">
                </div>
                <button type="submit" class="btn btn-primary btn-block">Verifikasi Sekarang</button>
            </form>
            <div id="verifyOtpStatus" style="margin-top: 16px;"></div>
        </div>
    `;

    document.getElementById('verifyRequestForm').addEventListener('submit', handleRequestVerification);
    document.getElementById('verifyOtpForm').addEventListener('submit', handleVerifyOtp);
}

async function handleRequestVerification(e) {
    e.preventDefault();
    if (!token || !currentUser) {
        showAlert('Silakan login dulu', 'info');
        navigate('/login');
        return;
    }
    const email = document.getElementById('verifyEmail').value;
    const statusEl = document.getElementById('verifyRequestStatus');
    statusEl.innerHTML = '';

    const res = await apiFetch('/request-verification', {
        method: 'POST',
        body: JSON.stringify({ email_ntust: email })
    });
    const data = await res.json();

    if (res.ok) {
        statusEl.innerHTML = `<div class="alert alert-success">${data.message}</div>`;
    } else {
        statusEl.innerHTML = `<div class="alert alert-error">${data.error}</div>`;
    }
}

async function handleVerifyOtp(e) {
    e.preventDefault();
    const otp = document.getElementById('verifyOtp').value;
    const nim = document.getElementById('verifyNim').value;
    const statusEl = document.getElementById('verifyOtpStatus');
    statusEl.innerHTML = '';

    // Find token from localStorage or use one we got earlier
    // We need the verification token - for simplicity, we assume the user entered it from email
    // In real impl, we could store token in localStorage
    const res = await apiFetch('/verify', {
        method: 'POST',
        body: JSON.stringify({
            token: localStorage.getItem('verifyToken') || '',
            otp, nim
        })
    });
    const data = await res.json();

    if (res.ok) {
        showAlert(data.message, 'success');
        localStorage.removeItem('verifyToken');
        // Refresh user
        const meRes = await apiFetch('/me');
        currentUser = (await meRes.json()).data;
        updateNav();
        navigate('/profile');
    } else {
        statusEl.innerHTML = `<div class="alert alert-error">${data.error}</div>`;
    }
}

function renderProfile() {
    if (!currentUser) { navigate('/login'); return; }
    const app = document.getElementById('app');
    const u = currentUser;

    app.innerHTML = `
        <div class="card">
            <div class="profile-summary">
                <div>
                    <h2>${u.nama || '-'}</h2>
                    <p class="sub">${u.email}</p>
                    <span class="badge badge-${u.status === 'verified' ? 'verified' : 'pending'}">${u.status === 'verified' ? 'Terverifikasi' : 'Pending'}</span>
                </div>
            </div>
            <div style="margin-top: 24px;">
                <h3 class="profile-section">Profil &amp; Pasca</h3>
                <form id="profileForm" style="background: #f8f9fa; padding: 20px; border-radius: 8px;">
                    <div class="form-group">
                        <label>Nama</label>
                        <input type="text" id="pNama" value="${u.nama || ''}">
                    </div>
                    <div class="form-group">
                        <label>No. Telepon</label>
                        <input type="text" id="pNoTelp" value="${u.no_telp || ''}">
                    </div>
                    <div class="form-group">
                        <label>No. WA</label>
                        <input type="text" id="pNoWA" value="${u.no_wa || ''}">
                    </div>
                    <div class="form-group">
                        <label>Telegram</label>
                        <input type="text" id="pTelegram" value="${u.telegram || ''}">
                    </div>
                    <div class="form-group">
                        <label>Akun Sosmed</label>
                        <div class="sosmed-list" style="margin-bottom: 8px;">
                            <input type="text" id="pFB" placeholder="facebook.com/username" style="flex:1; margin-right:4px;" value="${u.sosmed?.facebook || ''}">
                        </div>
                        <input type="text" id="pIG" placeholder="Instagram" value="${u.sosmed?.instagram || ''}">
                        <input type="text" id="pTwitter" placeholder="Twitter/X" value="${u.sosmed?.twitter || ''}">
                        <input type="text" id="pThreads" placeholder="Threads" value="${u.sosmed?.threads || ''}">
                    </div>
                    <div class="form-group">
                        <label>Foto Profil</label>
                        <input type="text" id="pFoto" value="${u.foto_profil || ''}">
                    </div>
                    <div class="form-group">
                        <label>LinkedIn</label>
                        <input type="text" id="pLinkedIn" value="${u.linkedin || ''}">
                    </div>
                    <div class="form-group">
                        <label>Wilayah</label>
                        <select id="pWilayah">
                            <option value="">${u.wilayah || ''}</option>
                            ${getWilayahOptions(u.wilayah)}
                        </select>
                    </div>
                </form>
                <button type="submit" id="saveProfileBtn" class="btn btn-primary" style="margin-top: 12px;">Simpan Profil</button>

                <h3 class="profile-section" style="margin-top: 24px;">Pekerjaan</h3>
                <div class="edu-field" id="pekerjaanDisplay"></div>
                <button class="btn btn-link" id="editPekerjaan">Edit Pekerjaan</button>
            </div>
        </div>
    `;
    document.getElementById('profileForm').addEventListener('submit', handleProfileSave);
    renderPekerjaanDisplay(u);
    document.getElementById('editPekerjaan').addEventListener('click', function() {
        renderPekerjaanEdit(u);
    });
}

function getWilayahOptions(current) {
    const wilayahs = ['Jakarta', 'Bandung', 'Surabaya', 'Medan', 'Yogyakarta', 'Semarang', 'Makassar', 'Denpasar', 'Balikpapan'];
    return wilayahs.filter(w => w !== current)
        .map(w => `<option>${w}</option>`).join('');
}

function renderPekerjaanDisplay(user) {
    const div = document.getElementById('pekerjaanDisplay');
    const p = user.pekerjaan;
    if (!p) { div.innerHTML = 'Belum diisi'; return; }
    div.innerHTML = `
        <p><strong>Institusi:</strong> ${p.institusi || '-'}</p>
        <p><strong>Jenis Industri:</strong> ${p.jenis_industri || '-'}</p>
        <p><strong>Jabatan:</strong> ${p.jabatan || '-'}</p>
        <p><strong>Alamat Kantor:</strong> ${p.alamat_kantor || '-'}</p>
        <p><strong>Email Kerja:</strong> ${p.email_kerja || '-'}</p>
    `;
}

function renderPekerjaanEdit(user) {
    const pekerjaan = user.pekerjaan || {};
    document.getElementById('pekerjaanDisplay').innerHTML = `
        <form id="pekerjaanForm">
            <div class="form-group">
                <label>Institusi</label><input type="text" id="pInstitusi" value="${pekerjaan.institusi || ''}">
            </div>
            <div class="form-group">
                <label>Jenis Industri</label><input type="text" id="pIndustri" value="${pekerjaan.jenis_industri || ''}">
            </div>
            <div class="form-group">
                <label>Jabatan</label><input type="text" id="pJabatan" value="${pekerjaan.jabatan || ''}">
            </div>
            <div class="form-group">
                <label>Alamat Kantor</label><input type="text" id="pAlamatKantor" value="${pekerjaan.alamat_kantor || ''}">
            </div>
            <div class="form-group">
                <label>Telp Kantor</label><input type="text" id="pTelpKantor" value="${pekerjaan.telp_kantor || ''}">
            </div>
            <div class="form-group">
                <label>Email Kerja</label><input type="email" id="pEmailKerja" value="${pekerjaan.email_kerja || ''}">
            </div>
            <button type="submit" class="btn btn-primary btn-block">Simpan Pekerjaan</button>
        </form>
    `;
    document.getElementById('pekerjaanForm').addEventListener('submit', handlePekerjaanSave);
}

async function handlePekerjaanSave(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const pekerjaan = {
        institusi: form.get('institusi'),
        jenis_industri: form.get('jenis_industri'),
        jabatan: form.get('jabatan'),
        alamat_kantor: form.get('alamat_kantor'),
        telp_kantor: form.get('telp_kantor'),
        email_kerja: form.get('email_kerja'),
    };
    const res = await apiFetch('/alumni/' + currentUser.id, {
        method: 'PUT',
        body: JSON.stringify({ pekerjaan })
    });
    if (res.ok) {
        showAlert('Pekerjaan berhasil disimpan', 'success');
        renderPekerjaanDisplay({ pekerjaan });
    }
}

async function handleProfileSave(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const update = {
        nama: form.get('nama'),
        no_telp: form.get('no_telp'),
        no_wa: form.get('no_wa'),
        telegram: form.get('telegram'),
        sosmed: {
            facebook: form.get('facebook'),
            instagram: form.get('instagram'),
            twitter: form.get('twitter'),
            threads: form.get('threads'),
        },
        foto_profil: form.get('foto_profil'),
        linkedin: form.get('linkedin'),
        wilayah: form.get('wilayah'),
    };
    const res = await apiFetch('/alumni/' + currentUser.id, { method: 'PUT', body: JSON.stringify(update) });
    if (res.ok) {
        showAlert('Profil berhasil disimpan', 'success');
        const meRes = await apiFetch('/me');
        currentUser = (await meRes.json()).data;
        renderProfile();
    } else {
        const data = await res.json();
        showAlert(data.error || 'Gagal menyimpan', 'error');
    }
}

// Directory
let dirWilayahFilter = '';
async function renderDirectory() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="section-header">
            <h3>Direktori Alumni</h3>
            <span style="color: #7f8c8d; font-size: 14px;">${dirWilayahFilter ? dirWilayahFilter + ' - ' : ''}Alumni Terverifikasi</span>
        </div>
        <div class="wilayah-filter" id="dirWilayahFilter"></div>
        <div id="dirList"></div>
    `;
    await loadDirectory();
    initWilayahFilter('dirWilayahFilter');
    updateNav();
}

function initWilayahFilter(containerId) {
    const container = document.getElementById(containerId);
    const wilayahs = ['Jakarta', 'Bandung', 'Surabaya', 'Medan', 'Yogyakarta', 'Semarang', 'Makassar', 'Denpasar', 'Balikpapan'];
    container.innerHTML = '<button data-wilayah="" class="' + (dirWilayahFilter === '' ? 'active' : '') + '">Semua</button>' +
        wilayahs.map(w => '<button data-wilayah="' + w + '" class="' + (dirWilayahFilter === w ? 'active' : '') + '">' + w + '</button>').join('');
    container.querySelectorAll('button').forEach(btn => {
        btn.onclick = () => {
            dirWilayahFilter = btn.dataset.wilayah;
            loadDirectory();
        };
    });
}

async function loadDirectory() {
    const listEl = document.getElementById('dirList');
    listEl.innerHTML = '<div class="empty-state">Memuat...</div>';

    const res = await apiFetch('/alumni');
    const data = await res.json();

    if (!res.ok || !data.data) {
        listEl.innerHTML = '<div class="empty-state">Gagal memuat direktori</div>';
        return;
    }

    let alumni = data.data.filter(a => !dirWilayahFilter || a.wilayah === dirWilayahFilter);

    if (alumni.length === 0) {
        listEl.innerHTML = '<div class="empty-state">Tidak ada alumni di wilayah ini</div>';
        return;
    }

    listEl.innerHTML = '';
    alumni.forEach(a => {
        const card = document.createElement('div');
        card.className = 'alumni-card';
        card.innerHTML = `
            <h4>${esc(a.nama) || '-'}</h4>
            <div class="wilayah">${a.wilayah || '-'}</div>
            <span class="badge badge-${a.role === 'member' ? 'member' : a.role}">${a.role || 'Member'}</span>
        `;
        card.querySelector('h4').onclick = () => {
            // Load detail
            apiFetch('/alumni/' + a.id).then(r => r.json()).then(d => {
                showAlumniDetail(d.data);
            });
        };
        card.querySelector('h4').style.cursor = 'pointer';
        card.querySelector('h4').style.textDecoration = 'underline';
        listEl.appendChild(card);
    });
}

function showAlumniDetail(alumni) {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">${alumni.nama || '-'}</h2>
            <div class="sosmed-list" style="margin: 12px 0;">
                ${alumni.email ? `<span>${alumni.email}</span>` : ''}
                ${alumni.no_wa ? `<span>${alumni.no_wa}</span>` : ''}
                ${alumni.telegram ? `<span>${alumni.telegram}</span>` : ''}
                ${alumni.linkedin ? `<a href="${alumni.linkedin}" target="_blank" style="color: #3498db;">LinkedIn</a>` : ''}
            </div>
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-top: 16px;">
                <div>
                    <p><strong>Wilayah:</strong> ${alumni.wilayah || '-'}</p>
                    <p><strong>Bidang Keahlian:</strong> ${(alumni.bidang_keahlian || []).join(', ') || '-'}</p>
                </div>
                <div>
                    <p><strong>Pekerjaan:</strong><br>${alumni.pekerjaan ? alumni.pekerjaan.jabatan + ' di ' + alumni.pekerjaan.institusi : '-'}</p>
                    <p><strong>Pendidikan:</strong><br>${alumni.pendidikan_ntust ? alumni.pendidikan_ntust.fakultas + ' / ' + alumni.pendidikan_ntust.jurusan : '-'}</p>
                </div>
            </div>
            <button class="btn btn-secondary" onclick="navigate('#directory')">Kembali</button>
        </div>
    `;
}

// Forum
let forumFilteredByVerified = false;
async function renderForum() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="section-header">
            <h3>Forum Diskusi</h3>
            <div id="createPostBtn"></div>
        </div>
        <div id="forumList"></div>
    `;
    await loadForum();
    initCreatePostButton();
}

function initCreatePostButton() {
    const btnContainer = document.getElementById('createPostBtn');
    if (currentUser && currentUser.status === 'verified') {
        btnContainer.innerHTML = `<button class="btn btn-primary" onclick="openCreatePostForm()">Buat Postingan Baru</button>`;
    }
}

function openCreatePostForm() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Buat Postingan</h2>
            <form id="forumPostForm">
                <div class="form-group">
                    <label>Judul</label>
                    <input type="text" id="postJudul" required placeholder="Judul Diskusi">
                </div>
                <div class="form-group">
                    <label>Isi</label>
                    <textarea id="postIsi" rows="5" required placeholder="Deskripsi Anda..."></textarea>
                </div>
                <button type="submit" class="btn btn-primary">Posting</button>
            </form>
        </div>
    `;
    document.getElementById('forumPostForm').addEventListener('submit', handleCreatePost);
}

async function loadForum() {
    const listEl = document.getElementById('forumList');
    listEl.innerHTML = '<div class="empty-state">Memuat...</div>';

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
                <span class="forum-post-author">${esc(p.penulis_nama) || '-'}</span>
                <span class="forum-post-date">${formatDate(p.dibuat_pada)}</span>
            </div>
            <h4 class="forum-post-title">${esc(p.judul) || '-'}</h4>
            <div class="forum-post-body">${esc(p.isi) || ''}</div>
        `;
        listEl.appendChild(postEl);
    });
}

async function handleCreatePost(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const res = await apiFetch('/forum', {
        method: 'POST',
        body: JSON.stringify({
            judul: form.get('judul'),
            isi: form.get('isi'),
        })
    });
    const data = await res.json();
    if (res.ok || res.status === 201) {
        showAlert('Postingan berhasil diposting', 'success');
        loadForum();
        document.getElementById('forumList').scrollTop = 0;
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

// Subgroup management (admin)
async function renderSubgroups() {
    if (!currentUser) { navigate('/login'); return; }
    if (currentUser.role !== 'super_admin' && currentUser.role !== 'regional_admin') {
        navigate('/profile');
        return;
    }
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="section-header">
            <h3>Kelola Subgroup Wilayah</h3>
            <button class="btn btn-primary" onclick="openCreateSubgroupForm()">Buat Subgroup Baru</button>
        </div>
        <div id="subgroupList"></div>
    `;
    await loadSubgroups();
}

function openCreateSubgroupForm() {
    const app = document.getElementById('app');
    app.innerHTML = `
        <div class="card">
            <h2 class="card-title">Buat Subgroup Baru</h2>
            <form id="subgroupForm">
                <div class="form-group">
                    <label>Nama Subgroup</label>
                    <input type="text" id="sgNama" required placeholder="IKATAN ALUMNI NTUST WILAYAH JAKARTA">
                </div>
                <div class="form-group">
                    <label>Wilayah</label>
                    <select id="sgWilayah">
                        <option value="">Pilih Wilayah</option>
                        <option>Jakarta</option><option>Bandung</option><option>Surabaya</option>
                        <option>Medan</option><option>Yogyakarta</option><option>Semarang</option>
                        <option>Makassar</option><option>Denpasar</option><option>Balikpapan</option>
                        <option>Lainnya</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Deskripsi</label>
                    <textarea id="sgDeskripsi" rows="3"></textarea>
                </div>
                <div class="form-group">
                    <label>Admin Wilayah (pilih dari wadah hubungi kami di forum)</label>
                    <input type="text" id="sgAdminId" placeholder="ID alumni (Diisi nanti setelah appoint)">
                </div>
                <button type="submit" class="btn btn-primary">Simpan Subgroup</button>
            </form>
        </div>
    `;
    document.getElementById('subgroupForm').addEventListener('submit', handleCreateSubgroup);
}

async function handleCreateSubgroup(e) {
    e.preventDefault();
    const form = new FormData(e.target);
    const res = await apiFetch('/subgroups', {
        method: 'POST',
        body: JSON.stringify({
            nama: form.get('nama'),
            wilayah: form.get('wilayah'),
            deskripsi: form.get('deskripsi'),
            admin_id: form.get('admin_id'),
        })
    });
    const data = await res.json();
    if (res.ok || res.status === 201) {
        showAlert('Subgroup berhasil dibuat', 'success');
        loadSubgroups();
        navigate('/subgroups');
    } else {
        showAlert(data.error || 'Gagal membuat subgroup', 'error');
    }
}

async function loadSubgroups() {
    const listEl = document.getElementById('subgroupList');
    listEl.innerHTML = '<div class="empty-state">Memuat...</div>';

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

    listEl.innerHTML = '';
    subs.forEach(s => {
        const card = document.createElement('div');
        card.className = 'alumni-card';
        card.innerHTML = `
            <h4>${esc(s.nama)}</h4>
            <div class="wilayah">${s.wilayah}</div>
            ${s.deskripsi ? `<p style="font-size: 13px; color: #555;">${s.deskripsi}</p>` : ''}
            ${s.admin_id ? `<span class="badge badge-admin">Admin: ${s.admin_id.substring(0, 8)}...</span>` : ''}
            <div style="margin-top: 12px;">
                <button class="btn btn-danger" onclick="deleteSubgroup('${s.id}')" style="font-size: 13px; padding: 6px 12px;">Hapus</button>
            </div>
        `;
        listEl.appendChild(card);
    });
}

async function deleteSubgroup(id) {
    if (!confirm('Hapus subgroup ini?')) return;
    const res = await apiFetch('/subgroups/' + id, { method: 'DELETE' });
    if (res.ok) {
        showAlert('Subgroup berhasil dihapus', 'success');
        loadSubgroups();
    } else {
        showAlert('Gagal menghapus', 'error');
    }
}

// Initialize
initUser();
