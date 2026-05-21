# Matriks Fitur (Feature IDs) - BayarAman Sistem

Dokumen ini mendata seluruh fitur (*Feature IDs*) yang ada pada sistem backend BayarAman. Penamaan ID ini berguna untuk dokumentasi, komunikasi tim, pelacakan perbaikan bug (*issue tracking*), dan perencanaan rilis.

## 1. Autentikasi & Otorisasi (AUTH)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **AUT-001** | Registrasi Pengguna | Pendaftaran akun baru dengan validasi email (RFC 5322) dan password (min 8 char, uppercase, lowercase, digit). |
| **AUT-002** | Login Pengguna | Autentikasi pengguna menggunakan email/password untuk mendapatkan JWT. |
| **AUT-003** | Verifikasi OTP | Pengiriman dan validasi kode OTP (via Email) untuk keamanan ganda. |
| **AUT-004** | Validasi Token & Akses | Middleware untuk memvalidasi akses endpoint berdasarkan *roles* JWT, termasuk pengecekan token blacklist. |
| **AUT-005** | Refresh Token Rotation | Mekanisme pembaruan token akses secara aman tanpa harus login ulang secara manual. |
| **AUT-006** | Device Session Management | Manajemen sesi login di berbagai perangkat (melihat dan mencabut sesi aktif). |
| **AUT-007** | Suspicious Login Detection | Mendeteksi dan memberi notifikasi aktivitas login dari lokasi/perangkat mencurigakan. |
| **AUT-008** | Access Token Blacklist | Saat logout, access token dimasukkan ke Redis blacklist (TTL 15 menit) sehingga tidak bisa digunakan kembali. |
| **AUT-009** | Input Validation | Validasi ketat format email dan kekuatan password pada saat registrasi. |

## 2. Manajemen Pengguna (USER)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **USR-001** | Profil Pengguna | Mendapatkan detail profil dan saldo akun pengguna saat ini. |
| **USR-002** | Update Profil | Memperbarui detail profil pengguna (Nama, Foto Profil). |
| **USR-003** | Verifikasi KYC | Verifikasi identitas pengguna (KTP/Selfie) untuk membuka limit transaksi atau status *verified*. |

## 3. Sistem Escrow / Rekber (ESC)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **ESC-001** | Buat Transaksi Escrow | Pembuatan *room* transaksi baru oleh Buyer atau Seller. |
| **ESC-002** | Pendanaan (Funding) | Integrasi dengan Xendit untuk mendapatkan Virtual Account / link pembayaran. |
| **ESC-003** | Manajemen State FSM | Transisi state ketat (Pending -> Funded -> Shipped -> Delivered -> Completed). |
| **ESC-004** | Upload Resi Pengiriman | API bagi Seller untuk mengunggah nomor resi fisik/digital dan foto bukti. |
| **ESC-005** | Selesai Transaksi | Buyer mengkonfirmasi barang tiba dan escrow memindahkan dana ke dompet penjual. |
| **ESC-006** | Expired Transaction | Pembatalan transaksi otomatis bila melewati batas waktu pembayaran (waktu kedaluwarsa diambil dari konfigurasi sistem, tidak di-*hardcode*). |
| **ESC-007** | Auto Release Escrow | Penyelesaian transaksi otomatis ke penjual (*completed*) jika pembeli tidak mengonfirmasi penerimaan dalam waktu 2x24 jam (dinamis) sejak barang *delivered*. |

## 4. Sistem Dompet Internal (WAL)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **WAL-001** | Ledger & Saldo | Penyimpanan informasi Saldo Utama (*Balance*) dan Saldo Ditahan (*Held Balance*). |
| **WAL-002** | Penarikan Dana (Withdraw)| Fitur penarikan uang yang terintegrasi Xendit Disbursements. |
| **WAL-003** | Riwayat Transaksi Dompet| Mutasi dompet lengkap (Debit/Kredit/Hold) untuk transparansi transaksi. |

## 5. Live Chat Escrow (CHT)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **CHT-001** | Koneksi WebSocket | Inisiasi *real-time chat* dalam sebuah *room* escrow spesifik. |
| **CHT-002** | Pesan Teks | Mengirim dan menerima pesan chat secara *real-time*. |
| **CHT-003** | Lampiran Gambar | Fitur unggah foto ke Storage yang tautannya dikirim via *chat websocket*. |
| **CHT-004** | Riwayat Obrolan | Memuat *history* chat yang tersimpan di PostgreSQL ketika membuka *room*. |

## 6. Dashboard Operasional Admin (ADM)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **ADM-001** | Pembekuan Escrow | Admin dapat membekukan transaksi (`freeze`) jika ada indikasi penipuan. |
| **ADM-002** | Resolusi Sengketa | Penentuan pemenang dari *dispute* dan pencairan dana ke pihak yang benar. |
| **ADM-003** | Log Audit Escrow | Pengecekan *timeline* seluruh transaksi (*logs* untuk investigasi admin). |
| **ADM-004** | Retry Pencairan | Pengulangan *payout* yang gagal di sisi gateway secara manual oleh admin. |

## 7. Keamanan & Performa Sistem (SYS)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **SYS-001** | Idempotency Protection | Mencegah *double-process* (misal: tombol klik ganda) berbekal *Redis Cache*. |
| **SYS-002** | Rate Limiting | Membatasi jumlah permintaan API per IP untuk mencegah *DDoS* dan *Brute-force*. |
| **SYS-003** | Xendit Webhooks | Mendengarkan event asinkronus (Pembayaran sukses / Disbursement berhasil). |
| **SYS-004** | Auto Reconciliation | Pekerja latar belakang (Cron) untuk menghitung dan mencocokkan saldo lokal dengan *Gateway*. |

## 8. Bukti & Dokumentasi Transaksi (EVI)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **EVI-001** | Upload Video Packing | Seller mengunggah video proses packing barang sebagai bukti kondisi sebelum kirim. |
| **EVI-002** | Upload Video Unboxing | Buyer mengunggah video unboxing sebagai bukti kondisi barang saat diterima. |
| **EVI-003** | Upload Foto Packing | Seller mengunggah foto packing (maks 3 foto) sebagai dokumentasi tambahan. |
| **EVI-004** | Upload Foto Unboxing | Buyer mengunggah foto unboxing (maks 3 foto) sebagai dokumentasi tambahan. |
| **EVI-005** | Konfirmasi Delivered | Buyer mengonfirmasi barang telah sampai, mengubah status ke *delivered*. |

## 9. Concurrency & Data Integrity (CON)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **CON-001** | Wallet Row Locking | Penggunaan `SELECT FOR UPDATE` pada operasi saldo dompet untuk mencegah *lost update* pada transaksi konkuren. |
| **CON-002** | Atomic Wallet Transaction | Seluruh operasi dompet (debit/kredit + pencatatan mutasi + payout) dibungkus dalam satu DB transaction — rollback otomatis jika gagal. |
| **CON-003** | Escrow State Row Locking | Penggunaan `SELECT FOR UPDATE` pada transisi state escrow untuk mencegah dua proses mengubah status secara bersamaan. |
| **CON-004** | Atomic State Transition | Penggunaan `UPDATE ... WHERE status = expected` untuk transisi state yang tidak bisa di-race (webhook vs worker, user vs admin). |
| **CON-005** | Idempotency Atomic Lock | Penggunaan Redis `SetNX` sebagai atomic lock acquisition — hanya satu request per key yang diproses, duplikat menunggu lalu menerima cached response. |
| **CON-006** | Graceful Lock Release | Jika request gagal (5xx), lock idempotency dihapus agar client dapat melakukan retry dengan key yang sama. |

## 10. Konfigurasi Dinamis (CFG)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **CFG-001** | Dynamic Platform Fee | Persentase biaya platform disimpan di Redis, dapat diubah admin tanpa restart server. |
| **CFG-002** | Dynamic Escrow Expiry | Durasi kedaluwarsa escrow (jam) disimpan di Redis, dapat diubah admin tanpa restart server. |
| **CFG-003** | Admin Config CRUD | Endpoint admin untuk membaca dan memperbarui konfigurasi sistem secara *live*. |

## 11. Pengiriman & Tracking (SHP)
| ID Fitur | Nama Fitur | Deskripsi |
| :--- | :--- | :--- |
| **SHP-001** | Get Shipping Rates | Mengambil daftar ongkos kirim dari berbagai kurir berdasarkan alamat asal, tujuan, dan berat paket (via Biteship). |
| **SHP-002** | Track Shipment | Melacak status terkini dan riwayat pengiriman berdasarkan nomor resi yang terdaftar pada escrow. |
| **SHP-003** | Webhook Tracking Update | Menerima notifikasi otomatis dari aggregator (Biteship) saat ada perubahan status pengiriman. Jika status `delivered`, escrow otomatis transisi ke *delivered*. |
| **SHP-004** | Shipment Status Sync | Pekerja latar (cron job, interval 30 menit) untuk sinkronisasi status pengiriman yang belum diperbarui oleh webhook. |
| **SHP-005** | Auto Register Tracking | Saat seller upload resi, nomor resi otomatis didaftarkan ke aggregator untuk mengaktifkan webhook tracking. |
| **SHP-006** | Auto Deliver on Tracking | Ketika webhook aggregator melaporkan paket `delivered`, escrow otomatis berpindah dari `shipped` ke `delivered` (memulai timer Auto Release). |
