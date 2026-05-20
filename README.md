# BayarAman API - Enterprise Escrow Backend

BayarAman adalah sistem backend escrow (rekber) skala enterprise yang dirancang dengan keamanan tinggi, skalabilitas, dan ketahanan terhadap fraud. Sistem ini dibangun menggunakan arsitektur modular di Golang.

Daftar lengkap fitur beserta ID pelacakan tersedia di [FEATURE_IDS.md](./FEATURE_IDS.md).

---

## Fitur Utama

Sistem ini menggunakan pendekatan Finite State Machine untuk menangani siklus hidup transaksi escrow, ditambah dengan pencatatan ledger keuangan (Wallet), Live Chat via WebSocket, proteksi Idempotency, dan operasional Admin.

### Autentikasi & Keamanan
- Registrasi dengan validasi email (RFC 5322) dan password (min 8 char, uppercase, lowercase, digit) — [AUT-001, AUT-009](./FEATURE_IDS.md#1-autentikasi--otorisasi-auth)
- Login JWT dengan refresh token rotation dan deteksi token reuse — [AUT-002, AUT-005](./FEATURE_IDS.md#1-autentikasi--otorisasi-auth)
- Verifikasi OTP via email — [AUT-003](./FEATURE_IDS.md#1-autentikasi--otorisasi-auth)
- Access token blacklist saat logout (Redis, TTL 15 menit) — [AUT-008](./FEATURE_IDS.md#1-autentikasi--otorisasi-auth)
- Device session management dengan IP dan user agent tracking — [AUT-006](./FEATURE_IDS.md#1-autentikasi--otorisasi-auth)
- Rate limiting terintegrasi — [SYS-002](./FEATURE_IDS.md#7-keamanan--performa-sistem-sys)

### Sistem Escrow (Rekber)
- Finite State Machine ketat: `pending > funded > shipped > delivered > completed` — [ESC-003](./FEATURE_IDS.md#3-sistem-escrow--rekber-esc)
- Integrasi Xendit untuk pembayaran (invoice) dan pencairan (disbursement) — [ESC-002](./FEATURE_IDS.md#3-sistem-escrow--rekber-esc)
- Upload bukti: video packing/unboxing, foto (maks 3), resi pengiriman — [EVI-001 s/d EVI-005](./FEATURE_IDS.md#8-bukti--dokumentasi-transaksi-evi)
- Expired transaction otomatis (lazy expiration + background worker) — [ESC-006](./FEATURE_IDS.md#3-sistem-escrow--rekber-esc)
- Konfirmasi delivered oleh buyer — [EVI-005](./FEATURE_IDS.md#8-bukti--dokumentasi-transaksi-evi)

### Sistem Dompet (Wallet Ledger)
- Saldo utama (balance) dan saldo ditahan (held_balance) — [WAL-001](./FEATURE_IDS.md#4-sistem-dompet-internal-wal)
- Penarikan dana terintegrasi Xendit Disbursements — [WAL-002](./FEATURE_IDS.md#4-sistem-dompet-internal-wal)
- Riwayat mutasi lengkap (debit/kredit) — [WAL-003](./FEATURE_IDS.md#4-sistem-dompet-internal-wal)

### Concurrency & Data Integrity
- Row locking (SELECT FOR UPDATE) pada wallet dan escrow state — [CON-001, CON-003](./FEATURE_IDS.md#9-concurrency--data-integrity-con)
- Atomic DB transactions untuk semua operasi keuangan — [CON-002](./FEATURE_IDS.md#9-concurrency--data-integrity-con)
- Atomic state transition (UPDATE WHERE status = expected) mencegah race webhook vs worker — [CON-004](./FEATURE_IDS.md#9-concurrency--data-integrity-con)
- Idempotency protection dengan Redis SetNX atomic lock — [CON-005, CON-006](./FEATURE_IDS.md#9-concurrency--data-integrity-con)

### Live Chat (WebSocket)
- Real-time chat per room escrow — [CHT-001, CHT-002](./FEATURE_IDS.md#5-live-chat-escrow-cht)
- Lampiran gambar dan riwayat obrolan — [CHT-003, CHT-004](./FEATURE_IDS.md#5-live-chat-escrow-cht)

### Admin & Operasional
- Freeze escrow, override dispute, audit timeline — [ADM-001 s/d ADM-003](./FEATURE_IDS.md#6-dashboard-operasional-admin-adm)
- Retry payout gagal — [ADM-004](./FEATURE_IDS.md#6-dashboard-operasional-admin-adm)
- Konfigurasi dinamis (fee, expiry) tanpa restart — [CFG-001 s/d CFG-003](./FEATURE_IDS.md#10-konfigurasi-dinamis-cfg)
- Auto reconciliation (background worker) — [SYS-004](./FEATURE_IDS.md#7-keamanan--performa-sistem-sys)

---

## Stack Teknologi

| Komponen | Teknologi |
| :--- | :--- |
| Bahasa | Go (Golang) |
| Web Framework | Echo v4 |
| Database | PostgreSQL |
| ORM | GORM |
| Cache / KV Store | Redis |
| Real-time | WebSocket (gorilla) |
| Payment Gateway | Xendit |
| Password Hashing | Argon2id |
| Auth | JWT (HS256) + OTP |
| Config | Viper (.env) |

---

## Struktur Proyek

```
bayaraman/
├── cmd/
│   ├── api/             # Entry point utama (main.go)
│   ├── migrate/         # Database migration runner
│   └── seed/            # Data seeder
├── config/              # Konfigurasi app, database, redis
├── internal/
│   ├── handler/         # HTTP & WebSocket controllers
│   ├── middleware/      # Auth, Idempotency, Role
│   ├── model/           # Domain entities & state machine
│   ├── repository/      # Data access layer
│   ├── router/          # Route definitions
│   ├── service/         # Business logic & integrations
│   └── worker/          # Background workers (expiry, reconciliation, cleanup)
├── pkg/
│   ├── hash/            # Argon2id password hashing
│   └── jwt/             # JWT generation & parsing
├── FEATURE_IDS.md       # Daftar lengkap fitur dengan ID pelacakan
```

---

## Cara Menjalankan

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Siapkan environment:
   ```bash
   cp .env.example .env
   # Sesuaikan konfigurasi database, redis, dan API keys
   ```

3. Jalankan migrasi database:
   ```bash
   go run cmd/migrate/main.go
   ```

4. Jalankan aplikasi:
   ```bash
   go run cmd/api/main.go
   ```

Server berjalan di port yang dikonfigurasi di `.env` (default: 8080).

---

## API Endpoints

### Auth (Public)
| Method | Path | Deskripsi |
| :--- | :--- | :--- |
| POST | `/api/v1/auth/register` | Registrasi akun baru |
| POST | `/api/v1/auth/verify-email` | Verifikasi OTP email |
| POST | `/api/v1/auth/login` | Login, dapatkan JWT |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Logout + blacklist token |

### Escrow (Protected)
| Method | Path | Deskripsi |
| :--- | :--- | :--- |
| POST | `/api/v1/escrows/` | Buat transaksi escrow |
| GET | `/api/v1/escrows/` | List escrow saya |
| POST | `/api/v1/escrows/:id/fund` | Bayar (buat invoice Xendit) |
| POST | `/api/v1/escrows/:id/receipt` | Upload resi (seller) |
| POST | `/api/v1/escrows/:id/deliver` | Konfirmasi terima (buyer) |
| POST | `/api/v1/escrows/:id/complete` | Selesaikan & cairkan dana |
| POST | `/api/v1/escrows/:id/videos/packing` | Upload video packing |
| POST | `/api/v1/escrows/:id/videos/unboxing` | Upload video unboxing |
| POST | `/api/v1/escrows/:id/photos/packing` | Upload foto packing |
| POST | `/api/v1/escrows/:id/photos/unboxing` | Upload foto unboxing |

### Wallet (Protected)
| Method | Path | Deskripsi |
| :--- | :--- | :--- |
| GET | `/api/v1/wallets/me` | Lihat saldo & mutasi |
| POST | `/api/v1/wallets/withdraw` | Tarik dana ke rekening |

### Webhooks (Unprotected)
| Method | Path | Deskripsi |
| :--- | :--- | :--- |
| POST | `/webhooks/xendit` | Callback pembayaran Xendit |
| POST | `/webhooks/privy` | Callback KYC Privy |

### Admin (Protected, Role: admin)
| Method | Path | Deskripsi |
| :--- | :--- | :--- |
| GET | `/api/v1/admin/users` | List semua user |
| POST | `/api/v1/admin/users/:id/suspend` | Suspend user |
| POST | `/api/v1/admin/escrows/:id/freeze` | Bekukan escrow |
| POST | `/api/v1/admin/escrows/:id/disputes/override` | Override dispute |
| GET | `/api/v1/admin/escrows/:id/timeline` | Audit timeline |
| POST | `/api/v1/admin/payouts/:id/retry` | Retry payout gagal |
| GET | `/api/v1/admin/configs` | Lihat konfigurasi |
| PUT | `/api/v1/admin/configs` | Update konfigurasi |

---

Dibuat untuk memberikan rasa aman dalam setiap transaksi.
