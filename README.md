# BayarAman API - Enterprise Escrow Backend

BayarAman adalah sistem backend *escrow* (*rekber*) skala *enterprise* yang dirancang dengan keamanan tinggi, skalabilitas, dan ketahanan terhadap *fraud*. Sistem ini dibangun menggunakan arsitektur modular di Golang.

## 🚀 Fitur Utama
Sistem ini menggunakan pendekatan *State Machine* untuk menangani siklus hidup sebuah transaksi escrow, ditambah dengan pencatatan ledger keuangan (Wallet), *Live Chat* via WebSocket, proteksi Idempotency, dan operasional Admin.

- **Finite State Machine (FSM) Escrow**: Validasi ketat untuk status transaksi (`pending`, `funded`, `shipped`, `delivered`, `disputed`, `completed`, `cancelled`).
- **Sistem Dompet (Wallet Ledger)**: Sistem penyelesaian dana (*payout*) yang dikendalikan secara internal dengan fitur `held_balance` (dana ditahan) dan rekonsiliasi.
- **Integrasi Pihak Ketiga**:
  - **Pembayaran & Pencairan**: Xendit Payment Gateway & Disbursements.
  - **Penyimpanan**: Object Storage (AWS S3 / Supabase Storage).
- **Keamanan Skala Enterprise**:
  - Proteksi Idempotency Key (via Redis) mencegah transaksi ganda.
  - Autentikasi berbasis JWT dengan validasi OTP.
  - *Rate Limiting* terintegrasi.
- **Live Chat (WebSocket)**: Sistem obrolan *real-time* untuk pembeli dan penjual, termasuk pengiriman lampiran gambar.
- **Admin & Rekonsiliasi**: *Dashboard* internal untuk manajemen sengketa (*dispute*) dan pekerja latar belakang (*background worker*) untuk pengecekan selisih dana (*reconciliation*).

## 🛠️ Stack Teknologi
- **Bahasa Pemrograman**: Go (Golang)
- **Database**: PostgreSQL
- **Caching & KV Store**: Redis
- **Komunikasi Real-time**: WebSockets
- **Manajemen Storage**: Supabase / S3 (Implementasi dinamis)
- **Email Service**: SMTP Integrations
- **Payment Gateway**: Xendit

## 📂 Struktur Proyek
```text
bayaraman/
├── api/             # Konfigurasi atau spesifikasi OpenAPI/Swagger
├── cmd/
│   └── api/         # Titik masuk utama aplikasi (main.go)
├── config/          # Konfigurasi aplikasi & database
├── internal/        # Logika domain utama (tidak dapat di-import oleh module eksternal)
│   ├── handler/     # Layer HTTP & WebSocket Controller
│   ├── middleware/  # Auth, Idempotency, Logger, RateLimiter
│   ├── router/      # Pendefinisian rute API
│   └── service/     # Layer Logika Bisnis & Integrasi (Xendit, S3, SMTP)
├── migrate/         # File migrasi database
├── pkg/             # Utility yang bisa digunakan kembali (logger, utils)
└── seed/            # Seeder untuk mengisi data awal / tes
```

## ⚙️ Cara Menjalankan Aplikasi

1. **Clone & Install Dependencies**
   ```bash
   go mod tidy
   ```

2. **Persiapkan Environment**
   Salin atau sesuaikan isi file `.env`:
   ```bash
   cp .env.example .env
   ```

3. **Migrasi Database**
   Pastikan PostgreSQL dan Redis sedang berjalan. Jalankan migrasi:
   ```bash
   # (Tergantung setup migrasi yang Anda gunakan, misal menggunakan golang-migrate)
   migrate -path migrate/ -database "postgres://user:pass@localhost:5432/bayaraman?sslmode=disable" up
   ```

4. **Jalankan Aplikasi**
   ```bash
   go run cmd/api/main.go
   ```

Aplikasi akan berjalan sesuai port yang dikonfigurasikan di `.env`.

---
*Dibuat untuk memberikan rasa aman dalam setiap transaksi bayar.*
