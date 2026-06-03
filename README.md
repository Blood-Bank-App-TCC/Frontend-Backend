# Aplikasi Bank Darah PMI

Web admin Bank Darah PMI untuk mengelola stok darah, permintaan darurat rumah sakit, broadcast donor eligible radius 10 KM, live response dashboard, QR check-in donor, data pendonor, rumah sakit mitra, dan laporan operasional.

Fokus repository saat ini adalah web admin, backend REST API, PostgreSQL/PostGIS, dan Docker Compose. Fitur mobile Flutter, Firebase, FCM, push notification, dan onboarding mobile belum menjadi prioritas pengerjaan lokal.

## Stack

- Frontend: React 18, Vite, TypeScript, React Router, Zustand, Axios, Nginx
- Backend: Go 1.22 REST API dengan `net/http`, `pgx`, JWT HMAC, AES-256-GCM untuk token QR
- Database: PostgreSQL 15 + PostGIS
- Local services: `frontend`, `backend`, `postgres`

## Menjalankan Lokal Dengan Docker

Pastikan Docker Desktop sudah aktif, lalu jalankan dari root project:

```bash
docker compose down
docker compose up --build
```

Jika Docker di mesin lokal memakai binary Compose lama, gunakan perintah ekuivalen `docker-compose down` dan `docker-compose up --build`.

URL lokal:

```text
Frontend web admin : http://localhost:5173
Backend health     : http://localhost:8080/api/v1/health
PostgreSQL         : localhost:5432
```

Login demo:

```text
Username: operator
Password: pmi123
```

Login demo donor mobile:

```text
Email donor seed : rian@example.test
Password         : pmi123
```

Token QR demo untuk check-in manual:

```text
QR-DEMO-001
```

## Service Docker

- `frontend`: Nginx static server, expose port `5173`, proxy `/api` ke service `backend:8080`
- `backend`: Go REST API, expose port `8080`, menunggu `postgres` healthy
- `postgres`: PostgreSQL/PostGIS, expose port `5432`, healthcheck `pg_isready`

Environment penting di `docker-compose.yml`:

```text
DATABASE_URL=postgres://bank_darah:bank_darah@postgres:5432/bank_darah?sslmode=disable
JWT_SECRET=dev-bank-darah-secret-change-me
QR_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
DEMO_MODE=true
```

`DEMO_MODE=true` membuat live response broadcast bergerak otomatis untuk demo web admin tanpa aplikasi mobile.

## Menjalankan Manual

Backend membutuhkan PostgreSQL/PostGIS dan environment backend. Contoh konfigurasi ada di `backend/.env.example`.

```bash
docker compose up postgres
cd backend
go run ./cmd/api
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Vite mem-proxy `/api` ke `http://127.0.0.1:8080`.

## Endpoint REST Utama

Base URL:

```text
/api/v1
```

Endpoint web admin:

```text
GET    /health
POST   /auth/admin/login
GET    /auth/me
GET    /stock
PUT    /stock/:bloodType/:productType
GET    /emergency/requests
POST   /emergency/requests
GET    /emergency/requests/:id/eligible-donors
POST   /emergency/requests/:id/broadcast
GET    /emergency/requests/:id/live-responses
PUT    /emergency/requests/:id/close
GET    /donors
POST   /donors
GET    /donors/:idOrQrToken
PUT    /donors/:id
PUT    /donors/:id/status
POST   /donations/checkin
GET    /hospitals
POST   /hospitals
PUT    /hospitals/:id
DELETE /hospitals/:id
```

Endpoint mobile donor:

```text
POST   /mobile/register
POST   /mobile/login
GET    /mobile/history
GET    /mobile/donor
GET    /mobile/stock
GET    /mobile/broadcast/active
POST   /mobile/respond
PUT    /mobile/device-token
PUT    /mobile/donor
```

Response sukses:

```json
{
  "success": true,
  "data": {}
}
```

Response error:

```json
{
  "success": false,
  "code": "ERROR_CODE",
  "message": "Pesan error"
}
```

## Tabel Database Utama

Migration idempotent berada di `backend/database/migrations/001_init.sql` dan membuat tabel:

- `admin_users`
- `users`
- `hospitals`
- `blood_requests`
- `donation_history`
- `blood_stock`
- `stock_transactions`
- `emergency_broadcasts`
- `live_responses`

Migration lanjutan `backend/database/migrations/002_auth_history_notifications.sql` menambahkan:

- `users.password_hash`
- `notification_logs`
- index riwayat donasi dan rumah sakit aktif

Seed awal mencakup admin `operator`, `superadmin`, stok darah awal, rumah sakit contoh, donor contoh, request darurat contoh, live responses, dan riwayat donasi.

## Fitur Web Admin

- Dashboard stok darah dan request terbaru
- Login admin demo
- CRUD dasar donor, detail donor, edit donor, dan status donor
- CRUD dasar rumah sakit
- Read/update stok darah
- Buat emergency request
- Review eligible donor dan broadcast request
- Monitor live response dan tutup request/broadcast
- QR/manual check-in donor dan auto tambah stok `WB`
- Laporan operasional dan export CSV

## Catatan Deployment

Struktur service sudah siap diarahkan ke Cloud Run dan Cloud SQL:

- Frontend dapat dibuild menjadi container Nginx atau hosting statis
- Backend menggunakan `DATABASE_URL`, `JWT_SECRET`, dan `QR_ENCRYPTION_KEY` dari environment
- FCM thank-you notification membutuhkan `FCM_PROJECT_ID` dan `FCM_SERVICE_ACCOUNT_PATH`
- Database lokal memakai PostgreSQL/PostGIS; untuk produksi gunakan Cloud SQL PostgreSQL dengan ekstensi PostGIS

Dokumentasi OpenAPI statis untuk endpoint baru tersedia di `backend/docs/openapi.yaml`.
