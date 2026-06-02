# Implementasi Fitur Bank Darah – Berdasarkan Prioritas PRD

## Latar Belakang
Setelah audit menyeluruh, ditemukan beberapa fitur yang belum lengkap. Implementasi dilakukan sesuai urutan prioritas PRD: **P0 → P1 → P2**.

---

## Fitur yang Akan Dikerjakan

### 🔴 P0 – Must Have

#### 1. FR-M02 – FCM Push Notification (End-to-End)
**Masalah:** Backend punya kode FCM tapi `FCM_PROJECT_ID` tidak dikonfigurasi, dan mobile tidak mengirim `deviceToken` saat register/login.

**Solusi:**
- **Mobile:** Integrasikan `firebase_messaging` package. Saat app pertama buka, minta izin notifikasi, ambil FCM token, lalu kirim ke backend sebagai bagian dari login/register payload.
- **Backend (Go):** Buat endpoint `PUT /mobile/device-token` untuk update token perangkat ke kolom `device_token` di tabel `users`.
- **FCM Config:** Tambahkan kolom `device_token` ke tabel (sudah ada di schema), update handler broadcast untuk menggunakan token yang tersimpan.
- **Konfigurasi:** Gunakan Firebase Admin SDK dengan service account JSON.

> [!IMPORTANT]
> FCM memerlukan Firebase project yang sudah dikonfigurasi. File `google-services.json` harus ada di `Mobile_tcc/android/app/`. Saat ini saya akan mengimplementasikan kode-nya dan menginstruksikan cara setup Firebase project.

---

### 🟠 P1 – Should Have (Target v1.0)

#### 2. FR-A09 – Edit Donor dari Web Admin (CRUD lengkap)
**Masalah:** API `PUT /donors/:id` sudah ada tapi tidak ada tombol/form Edit di `DonorDetailPage.tsx`.

**Solusi:** Tambahkan modal/form Edit di `DonorDetailPage.tsx` yang memanggil `bankDarahController.updateDonor()`.

#### 3. FR-A10 – Edit Rumah Sakit dari Web Admin (CRUD lengkap)
**Masalah:** API `PUT /hospitals/:id` sudah ada tapi tidak ada tombol/form Edit di `HospitalsPage.tsx`.

**Solusi:** Tambahkan modal Edit di `HospitalsPage.tsx`.

#### 4. FR-A05 – Tutup Broadcast dari Admin
**Masalah:** Tidak ada tombol "Tutup Broadcast" di `MonitorPage.tsx`. PRD (User Journey Langkah 7) mensyaratkan admin dapat menutup broadcast.

**Solusi:**
- Backend: Tambahkan endpoint `PUT /api/v1/emergency/requests/{id}/close` yang mengubah status ke `FULFILLED` dan menutup broadcast.
- Frontend: Tambahkan tombol "Tutup & Selesaikan" di `MonitorPage.tsx`.

#### 5. FR-M09 – Edit Profil Donor dari Mobile
**Masalah:** `EditProfilePage` tidak baca session dan tidak panggil API. Tidak ada endpoint `PUT /mobile/donor` di backend.

**Solusi:**
- Backend: Tambahkan endpoint `PUT /mobile/donor` (auth via qr_token) untuk update nama, email, telepon, alamat, lat/lng.
- Mobile: Sambungkan `EditProfilePage` ke `SessionManager` dan panggil API baru tersebut.

#### 6. Session Persistence Mobile (SharedPreferences)
**Masalah:** Session hanya in-memory. App di-kill → harus login ulang.

**Solusi:** Simpan data donor (JSON) dan qr_token ke `SharedPreferences`. Baca saat app pertama dibuka. Jika ada session tersimpan, langsung masuk ke `DashboardPage` (skip login).

#### 7. Logout Bersih
**Masalah:** Tombol Logout di `ProfilePage` tidak memanggil `clearSession()`.

**Solusi:** Satu baris fix — panggil `SessionManager().clearSession()` dan hapus data SharedPreferences sebelum push ke `LoginPage`.

#### 8. FR-A08 – Stok Otomatis Naik Setelah Check-in Berhasil
**Masalah:** PRD mensyaratkan stok bertambah otomatis setelah donasi berhasil, saat ini harus manual.

**Solusi:** Di handler `handleCheckin` backend (Go), setelah `INSERT donation_history` dengan status COMPLETED, panggil `store.UpdateStock()` dengan mode `add` quantity=1 untuk blood type donor tersebut.

#### 9. FR-M11 – Onboarding First-Run
**Masalah:** `OnboardingPage` ada tapi selalu ditampilkan (tidak hanya first-run).

**Solusi:** Gunakan `SharedPreferences` key `hasSeenOnboarding`. Cek di `main.dart`: jika false → tampilkan `OnboardingPage`, jika true → cek session → `DashboardPage` atau `LoginPage`.

---

### 🟡 P2 – Nice to Have

#### 10. FR-A11 – Export Laporan CSV
**Solusi:** Tambahkan tombol export di `ReportsPage.tsx` yang men-generate CSV dari data yang sudah di-fetch dan men-trigger download.

#### 11. FR-M06 – Cache Stok 30 Menit
**Solusi:** Simpan timestamp terakhir fetch stok di SharedPreferences. Skip fetch jika belum 30 menit.

---

## Urutan Pengerjaan

1. **[P0] Logout bersih** (5 menit) ← paling cepat, sangat fundamental
2. **[P1] Session persistence** (SharedPreferences) ← prerequisite untuk nomor 9
3. **[P1] Onboarding first-run** ← butuh SharedPreferences
4. **[P1] Backend: endpoint close broadcast + update profil mobile + auto-stok checkin**
5. **[P1] Web Admin: Edit Donor + Edit RS + Tutup Broadcast**
6. **[P1] Mobile: Edit profil tersambung ke API**
7. **[P0] FCM: device token collection di mobile + update backend**
8. **[P2] Export CSV laporan**
9. **[P2] Cache stok 30 menit**

---

## File yang Akan Dimodifikasi

### Mobile (Flutter)
- `lib/main.dart` — routing first-run logic
- `lib/core/utils/session_manager.dart` — tambah SharedPreferences persistence
- `lib/features/auth/controllers/auth_controller.dart` — simpan session ke SharedPreferences
- `lib/features/profile/pages/profile_page.dart` — fix logout
- `lib/features/profile/pages/edit_profile_page.dart` — baca session + call API
- `lib/core/constants/api_endpoints.dart` — tambah endpoint baru
- `pubspec.yaml` — tambah `shared_preferences`, `firebase_messaging`, `firebase_core`

### Backend (Go)
- `internal/app/server.go` — tambah route close broadcast + update mobile donor + device token
- `internal/app/store.go` — tambah store methods
- `internal/app/server.go` (handleCheckin) — auto-update stok

### Web Admin (React)
- `src/views/pages/DonorDetailPage.tsx` — tambah modal Edit Donor
- `src/views/pages/HospitalsPage.tsx` — tambah modal Edit RS
- `src/views/pages/MonitorPage.tsx` — tambah tombol Tutup Broadcast
- `src/models/apiClient.ts` — tambah endpoint closeRequest
- `src/views/pages/ReportsPage.tsx` — tambah export CSV

## Verification Plan
- Jalankan `flutter analyze` setelah perubahan mobile
- Test login → tutup app → buka lagi → harus langsung masuk dashboard
- Test edit profil → profil terupdate di session
- Test close broadcast → status berubah di DB
- Test checkin → stok bertambah otomatis
