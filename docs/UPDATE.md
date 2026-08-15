# SplitMate — Update Log

Catatan perubahan per tanggal. Format: tanggal, ringkasan, detail penting.

---

## 2026-08-15 — Bulk Invite Anggota

### Undangan Massal (paste banyak email sekaligus)

- Backend:
  - Endpoint baru `POST /api/v1/groups/{groupId}/invitations/bulk` (admin only).
    Request `{"emails": [...]}` — maksimal 50 email per request; setiap email
    dinormalisasi (lowercase/trim) dan divalidasi format. Semua undangan dibuat
    dalam satu transaksi (`Repository.CreateInvitations`).
  - Pengecekan batch `MembersByEmails` / `PendingInvitationsByEmails`
    (query `email = ANY($2)`) — tanpa N+1.
  - Response `201` berisi `invitations[]` (email, status, expiresAt, token —
    token hanya dikembalikan sekali) dan `failed[]` dengan reason
    `MEMBER_EXISTS` / `INVITATION_EXISTS` / `DUPLICATE`. Format email invalid,
    list kosong, atau > 50 email → `422 VALIDATION_ERROR` (fail fast).
  - Endpoint single `POST /invitations` tetap ada (tidak diubah).
- Frontend:
  - `invite-form` diganti dari input email tunggal menjadi textarea (pisah
    dengan koma/baris baru) + hasil: daftar email → token, tombol "Salin
    semua" (`email: token` per baris), copy per baris, dan daftar email yang
    gagal dengan alasan (sudah anggota / sudah diundang / duplikat).
  - Action `inviteMember` diganti `inviteMembers` (bulk) di
    `app/actions/groups.ts`; tipe `Invitation` di `lib/api.ts` dihapus (tidak
    terpakai).
  - i18n: key `groups.inviteEmails*`, `copyAll`, `inviteFailure*` di id/en;
    teks `inviteCreated`/`tokenExpiry` disesuaikan untuk jamak.
- Tes: service (forbidden, validasi, batch campuran dengan skip member/duplikat,
  pending invitation), handler (403/422/201 + daftar skip), integrasi
  `TestBulkInvite` (campuran: user baru yang register setelah diundang + user
  yang sudah punya akun) — hijau.
- E2E: label form undangan berubah ke "Emails" di `main-flow.spec.ts`.
- Note: flow accept tidak berubah — orang yang sudah punya akun tinggal login
  dengan email yang sama lalu paste token; yang belum punya akun register
  dulu dengan email tersebut.

---

## 2026-08-15 — Dropdown Kategori + Logo Grup Opsional

### Kategori Expense Kembali ke Dropdown

- Picker kategori di `add-expense-form` dikembalikan dari radio grid berikon ke
  `<select>` biasa (sesuai permintaan user). Ikon kategori tetap dipakai di
  dashboard & group page (baris expense).

### Logo Grup Opsional (saat membuat grup)

- Backend:
  - Migrasi `000003_group_logo`: kolom `logo_image BYTEA` +
    `logo_content_type VARCHAR(100)` di tabel `groups` (pola sama dengan receipt).
  - `POST /api/v1/groups` menerima multipart (name, description, currency, logo)
    — JSON lama tetap didukung. Validasi logo: ≤ 5MB, whitelist
    jpeg/png/webp/gif ("Logo image is empty" / "... at most 5MB" /
    "Logo must be a JPEG, PNG, WebP or GIF image").
  - Endpoint baru `GET /api/v1/groups/{groupId}/logo` (auth + membership;
    404 `LOGO_NOT_FOUND` bila tanpa logo).
  - Response grup (`GET/POST/PATCH` + list) menambah `hasLogo`; dashboard
    `GroupOverview` juga mendapat `hasLogo`.
- Frontend:
  - `create-group-form`: upload logo opsional (input file selalu ter-mount
    `sr-only`, preview, tombol hapus, validasi klien) — pola sama dengan
    receipt (pelajaran dari bug FormData yang hilang).
  - `createGroup` action kini kirim FormData multipart.
  - Komponen `components/group-logo.tsx`: tampilkan logo (`next/image`
    `unoptimized` karena butuh cookie session — optimizer server tidak membawa
    cookie) atau badge inisial bila tidak ada. Dipakai di halaman grup
    (header), daftar grup, dan dashboard.
  - i18n: key `groups.logo*` di kamus id/en.
- Tes: service (create with logo, validasi, get logo + otorisasi) & handler
  (multipart create, logo invalid → 422, get logo handler) — hijau.
- E2E: upload logo saat buat grup + assert `<img src*="/logo">` terlihat di
  halaman grup; 2/2 deterministik.
- Verifikasi: `go build`/`vet`/`test` hijau (kecuali `TestOAuthFindOrCreate`
  pre-existing), `tsc`, `eslint`, vitest 54/54, `bun run build`, E2E 2/2.

---

## 2026-08-15 — Upload Receipt, Header Auth, Ikon Kategori, Toaster Sonner, Koma Ribuan

### Upload Receipt (opsional) saat Tambah Expense

- Backend:
  - Migrasi `000002_expense_receipt`: kolom `receipt_image BYTEA` +
    `receipt_content_type VARCHAR(100)` di tabel `expenses`.
  - Keputusan penyimpanan: BYTEA di PostgreSQL (bukan S3/CDN) — nol infrastruktur
    tambahan, kompatibel dengan Zeabur/Vercel. Byte hanya dimuat saat detail/GET
    receipt; daftar expense hanya mengembalikan `hasReceipt: bool`.
  - Validasi (service): ukuran ≤ 5MB (`maxReceiptBytes`), whitelist
    `image/jpeg|png|webp|gif`, limit parse multipart 10MB. Error:
    "Receipt image is empty" / "... at most 5MB" / "Receipt must be a JPEG, PNG,
    WebP or GIF image".
  - Endpoint baru `GET /api/v1/expenses/{expenseId}/receipt` — mengembalikan byte
    + `Content-Type` + `Cache-Control: private`; 404 `RECEIPT_NOT_FOUND` bila
    expense tidak punya receipt. Wajib autentikasi + membership.
  - `POST /api/v1/groups/{groupId}/expenses` (dan PUT update) menerima multipart
    `multipart/form-data` (field: description, amount, currency, paidBy, category,
    expenseDate, note, splitType, participant[], split.<userId>, receipt) —
    JSON lama tetap didukung.
- Frontend:
  - `app/actions/expenses.ts`: kirim FormData multipart (tanpa header
    `Content-Type` manual agar boundary multipart di-set browser).
  - `add-expense-form`: input file sr-only + label, preview gambar (object URL),
    tombol remove, validasi klien tipe/ukuran.
  - **Bug yang ditemukan saat verifikasi E2E**: input file awalnya di-unmount
    (diganti UI preview) sehingga file hilang dari FormData saat submit — fix:
    input selalu ter-mount (`sr-only`), hanya label yang di-hide; remove juga
    me-reset `input.value` via ref.
  - Group page: link "View receipt" (ikon Paperclip, `target="_blank"`) hanya
    dirender saat `hasReceipt` — lewat rewrite `/api/v1/:path*`, cookie session
    ikut terkirim sehingga gambar terbuka ter-autentikasi.
- Tes: service (create with receipt, validasi, get receipt) + handler (multipart
  decode dengan `textproto.MIMEHeader`, get receipt handler) — semua hijau.
  Smoke test: create multipart dengan PNG → `hasReceipt: true`, GET receipt byte
  identik.

### Header Halaman Login/Register

- `components/auth-header.tsx`: logo (`/splitmate_logo.png`) + teks SplitMate di
  kiri (link ke `/login`), `LanguageSwitcher` di kanan. Switcher dipindah dari
  dalam kartu auth ke header.

### Ikon Kategori Expense

- `components/category-icon.tsx`: peta ikon (lucide): Accommodation→BedDouble,
  Food & Drinks→UtensilsCrossed, Transportation→CarFront, Shopping→ShoppingBag,
  Entertainment→Clapperboard, Utilities→Lightbulb, Other→Tag (fallback Tag).
- Dipakai di dashboard, group page, dan picker kategori di add-expense-form
  (radio grid menggantikan `<select>`; radio sr-only tetap punya `name="category"`
  agar nilai ter-submit, fokus keyboard via `has-[:focus-visible]:ring-2`).

### Toaster Sonner (menggantikan toast custom)

- `sonner` di-install; `<Toaster position="bottom-right" richColors closeButton />`
  di root layout.
- `components/toast.tsx` kini bridge ke sonner (mengembalikan `null`): panggil
  `toast.success` sekali (guard ref), lalu strip `?success=` via `router.replace`.
- Redirect auth: login → `/?success=signed-in`; register → `/login?success=registered`;
  halaman login & dashboard merender `<Toast>`.

### Input Amount: Koma Ribuan

- `lib/amount.ts`: `formatAmountInput` kini memakai pemisah ribuan koma untuk
  KEDUA locale ("1,000,000.50") — param `locale` dihapus; desimal tetap titik.
  Placeholder ID berubah ke "0.00".
- `amount-input.tsx`: prop opsional `value`/`onChange`/`ariaLabel` (mode
  controlled) — dipakai untuk input split custom di expense form.

### Perbaikan E2E (flaky, bukan regresi fitur)

- Owner menunggu halaman yang sudah dirender sebelum friend join → daftar member
  stale ("1 members"). Test sebelumnya lolos bergantung timing toast lama yang
  melakukan `router.replace` ulang (+4.3s) — dengan toast sonner yang langsung
  bersih, kelemahan ini deterministik. Fix: `owner.reload()` setelah friend join
  (konvensi test sudah memakai reload setelah mutasi pihak lain).
- Fixture `tests/e2e/fixtures/receipt.png` (4×4 PNG) + langkah
  `setInputFiles("#receipt")` + asersi link "View receipt".

### Verifikasi

- Backend: `go build`/`go vet` hijau; `go test ./...` hijau kecuali
  `TestOAuthFindOrCreate` (kegagalan pre-existing, terverifikasi di tree bersih).
- Frontend: `tsc --noEmit`, `eslint`, `vitest` 54/54, `bun run build`, Playwright
  E2E 2/2 (dijalankan 2×, deterministik) — semua hijau.

---

## 2026-08-15 — i18n (ID/EN) + Input Amount Terformat

### i18n Bahasa Indonesia (default) & Inggris

- Mekanisme: cookie `lang` (`id`/`en`, default `id`) — tanpa segmen URL.
  - Server component: `getLocale()`/`getDict()` di `lib/i18n/index.ts` (server-only, baca `cookies()`).
  - Client component: terima `dict` sebagai prop dari halaman server (bukan context).
  - `LanguageSwitcher` (client) di header app layout & pojok kartu login/register: set cookie + `router.refresh()`.
  - `lib/i18n/tr.ts`: helper interpolasi placeholder `{name}` → `tr(dict.x.y, { name })`.
- Kamus: `lib/i18n/id.ts` (ID) & `en.ts` (EN), `Dict = Omit<typeof id, "locale"> & { locale: Locale }`.
  **Kendala Next.js**: objek dict TIDAK boleh berisi fungsi (ditolak saat di-pass ke client
  component — "Functions cannot be passed directly to Client Components"); semua interpolasi
  memakai template string + `tr()`.
- Cakupan: dashboard, groups, group detail, expense form, settle panel, auth (login/register),
  sidebar, not-found, error page, toast, server action fallback error, tombol aksi.
- Keputusan: `formatCurrency`/`formatDate` tetap id-ID (uang mengikuti mata uang, bukan bahasa);
  pesan error backend (Go `ApiError`) tetap Inggris.

### Input Amount Terformat

- `lib/amount.ts`: `parseAmountInput`, `formatAmountInput`, `nextAmountInputValue`.
  - Heuristik parse: separator TERAKHIR dengan ≤2 digit di belakangnya = desimal; lainnya = pemisah ribuan.
  - Format tampilan sesuai locale: `id` → "1.000,50" (titik ribuan), `en` → "1,000.50".
- `components/amount-input.tsx`: input visible terformat (tanpa `name`) + hidden input `name`
  berisi nilai mentah (mis. `"150000"`/`"1000.50"`) — backend tetap menerima angka polos.
  Dipakai di `add-expense-form` (amount utama) & `settle-panel` (quick settle).
- Split custom di expense form tetap input angka polos (belum terformat).

### Perbaikan bug yang ditemukan saat verifikasi

- `inviteMember` (server action) mengembalikan token di field `error` → tidak pernah dirender
  sebagai elemen `<code>`; dikembalikan ke `{ token }`.
- E2E: tombol `Create`/`Join` (dari dict), heading "Join a group", regex token `code` tidak
  anchored (JSX whitespace), helper `newContext` dengan cookie `lang=en`.
- `locale` pada kamus diberi `as const` agar tipe `Dict` valid.

### Verifikasi

- `tsc --noEmit`, `eslint`, `vitest` 55/55, `bun run build`, Playwright E2E 2/2 (main-flow)
  — semua hijau.

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
