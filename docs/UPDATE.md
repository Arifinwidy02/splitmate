# SplitMate — Update Log

Catatan perubahan per tanggal. Format: tanggal, ringkasan, detail penting.

---

## 2026-08-15 — Hotfix: Google Sign-In gagal untuk user baru (production)

- Gejala: login Google di production menampilkan "Google sign in failed.",
  log backend: `cannot scan NULL into *string` (kolom `password_hash`).
- Penyebab: `CreateWithOAuth` membuat user tanpa `password_hash` (NULL), tetapi
  `scanUser` di `internal/user/repository.go` meng-scan kolom tersebut ke `string`
  biasa → error saat membuat user OAuth baru.
- Perbaikan:
  - `scanUser` memakai `pgtype.Text` untuk `password_hash` (NULL → string kosong).
  - `Login` menolak user dengan hash kosong (user OAuth-only tidak bisa login
    pakai password) dengan `ErrInvalidCredentials`.
- Tes: `TestOAuthFindOrCreate` (integrasi, DB asli) — user OAuth baru, re-login,
  linking email, dan penolakan password login. `go test ./...` hijau.
- UI: tombol "Continue with Google" dipindah ke bawah form (form → divider "or" →
  tombol Google) di halaman login & register.
- Deployment: hanya backend yang berubah → redeploy Zeabur cukup.

---

## 2026-08-15 — Google Sign In + Production (branch `staging`)

### Google Sign In (OAuth 2.0)

- Backend:
  - `GET /api/v1/auth/google` — redirect ke Google consent screen dengan
    `oauth_state` cookie (HttpOnly, 10 menit) untuk proteksi CSRF.
  - `GET /api/v1/auth/google/callback` — validasi state, tukar `code` → token,
    ambil profil, lalu find-or-create user:
    1. `oauth_accounts` sudah ada → login sebagai user itu
    2. email sudah terdaftar (user password) → tautkan akun, login
    3. belum ada → buat user baru (tanpa password hash) + tautkan akun
  - Sukses: session cookie + redirect ke `/`. Gagal: redirect `/login?google=error`
    (detail dicatat di log server, tidak diekspos).
  - Config baru: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `OAUTH_REDIRECT_URL`,
    `APP_BASE_URL`. Tanpa konfigurasi, endpoint mengembalikan `503 GOOGLE_NOT_CONFIGURED`.
- Frontend:
  - `next.config.ts` rewrites: `/api/v1/:path*` → API. Ini penting: callback Google
    harus lewat origin frontend agar session cookie jatuh di domain frontend
    (domain API berbeda → browser tidak mengirim cookie-nya ke frontend).
  - Tombol "Continue with Google" di halaman login & register.
  - Banner error di `/login?google=error`.
- Tes: service (find-or-create, linking, kegagalan exchange) + handler (redirect,
  state cookie, callback sukses/gagal) — semua hijau. `go test ./...`, `tsc`,
  `eslint`, `build`, 41 vitest hijau.
- Verifikasi lokal: proxy rewrite → 302 ke Google; callback gagal → redirect error.

### Deployment Production (M10)

- Frontend live: `https://splitmate-phi.vercel.app` (Vercel, Root Directory
  `frontend`; `rootDirectory` di `vercel.json` tidak didukung lagi — diatur dari dashboard).
- API live: `https://splitmate.zeabur.app` (Zeabur, `backend/Dockerfile`,
  migrasi otomatis di startup).
- PostgreSQL production di Zeabur.
- Smoke test produksi: register → login → grup → expense → dashboard → hapus grup, lulus.

### Catatan

- Setelah perubahan `NEXT_PUBLIC_API_URL`, wajib redeploy (nilainya di-inline saat build).
- Modul Go masih `github.com/Arifinwidy02/splitmate-backend` (nama repo berubah jadi
  `splitmate`) — kosmetik, belum diubah.

### Yang harus dilakukan selanjutnya

1. Buat OAuth client di Google Cloud Console (Web application).
2. Daftarkan redirect URI:
   - `https://splitmate-phi.vercel.app/api/v1/auth/google/callback`
   - `http://localhost:3000/api/v1/auth/google/callback` (dev)
3. Set env di Zeabur: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`,
   `OAUTH_REDIRECT_URL=https://splitmate-phi.vercel.app/api/v1/auth/google/callback`,
   `APP_BASE_URL=https://splitmate-phi.vercel.app` → deploy ulang API.
4. Test manual: login via Google di production.