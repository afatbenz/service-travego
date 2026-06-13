# Wagy Integration Setup Guide

Panduan lengkap untuk setup integrasi Wagy WhatsApp API dengan modul WAAI.

## Prasyarat

1. **Wagy Account**: Daftar di [Wagy](https://wagy.web.id)
2. **Device WhatsApp**: Nomor WhatsApp yang sudah terdaftar di Wagy
3. **Anthropic API Key**: Daftar di [Anthropic](https://console.anthropic.com)
4. **Server dengan Public IP/Domain**: Untuk webhook callback dari Wagy

## Step 1: Setup Wagy Device

### Registrasi Device
1. Login ke dashboard Wagy
2. Buka menu "Devices"
3. Klik "Add Device"
4. Scan QR code dengan WhatsApp Business
5. Tunggu status menjadi "Connected"
6. Catat **Device ID** (contoh: `OFFICE-01`)

### Generate API Token
1. Di halaman Device, klik settings
2. Generate API Token baru
3. Catat token ini (gunakan untuk `WAGY_TOKEN`)

## Step 2: Setup Webhook

### 1. Tentukan Domain/IP Public
Wagy perlu mengirim callback ke server Anda. Pastikan:
- Server accessible dari internet
- Port 3100 (atau port yang digunakan) terbuka
- Gunakan HTTPS jika production

Webhook URL: `https://your-domain.com/waai/webhook`

### 2. Generate Webhook Secret
Buat random string yang aman (minimal 32 karakter):
```bash
# Linux/Mac
openssl rand -base64 32

# Windows PowerShell
[System.Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

### 3. Register Webhook di Wagy
1. Login ke Wagy Dashboard
2. Buka Device settings
3. Klik "Add Webhook"
4. URL: `https://your-domain.com/waai/webhook`
5. Secret: Paste webhook secret yang sudah dibuat
6. Events: Pilih `message.received` dan `message.sent`
7. Klik "Test URL" - seharusnya return status 200
8. Klik "Save"

## Step 3: Setup Database

### 1. Run Migration
```bash
# Execute SQL file
psql -U postgres -d traveGo -f database/migrations/001_create_wa_contacts.sql
```

### 2. Tambah Test Contact
```sql
INSERT INTO wa_contacts (phone, name, role, organization_id, is_active)
VALUES ('628123456789', 'John Doe', 'admin', 1, true);
```

Ganti:
- `628123456789` dengan nomor WhatsApp Anda
- `John Doe` dengan nama Anda
- `1` dengan organization_id yang valid

## Step 4: Environment Configuration

Tambah ke `.env`:

```bash
# Wagy Configuration
WAGY_DEVICE_ID=OFFICE-01
WAGY_TOKEN=your-wagy-api-token
WAGY_WEBHOOK_SECRET=your-webhook-secret-from-step-2

# Anthropic Configuration
ANTHROPIC_API_KEY=sk-ant-v1-your-anthropic-key

# Redis (pastikan sudah berjalan)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_URL=redis://localhost:6379
```

## Step 5: Testing

### 1. Test Local Setup
```bash
# Build
go build -o main.exe

# Run
./main.exe
```

Server akan log: `[WAAI] Routes registered successfully`

### 2. Test Webhook Verification
```bash
curl "http://localhost:3100/waai/webhook?challenge=test123"
# Response: test123
```

### 3. Test Manual Message (simulation)
```bash
# Buat file test.json
cat > test.json << 'EOF'
{
  "event": "message.received",
  "source": "whatsapp",
  "data": {
    "id": 1001,
    "device_id": "OFFICE-01",
    "owner_jid": "628123456789@s.whatsapp.net",
    "content": {
      "pn_jid": "628123456789@s.whatsapp.net",
      "content": "Halo, cek ketersediaan bus",
      "message_id": "ABC123XYZ",
      "timestamp": "2026-06-12T07:05:00Z"
    }
  }
}
EOF

# Generate HMAC signature
WEBHOOK_SECRET="your-webhook-secret"
SIGNATURE=$(echo -n "$(cat test.json)" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex | cut -d' ' -f2)

# Send test request
curl -X POST http://localhost:3100/waai/webhook \
  -H "X-Wagy-Signature: $SIGNATURE" \
  -H "Content-Type: application/json" \
  -d @test.json
```

### 4. Real WhatsApp Test
Kirim pesan WhatsApp ke nomor device Anda dari nomor yang sudah terdaftar di `wa_contacts`.

Periksa:
- Server logs untuk `[WAAI]` entries
- Redis session tersimpan
- Respons diterima di WhatsApp

## Step 6: Production Deployment

### 1. Ngrok (Testing saja)
Untuk test cepat tanpa domain:
```bash
ngrok http 3100
# Catat URL: https://xxxxx.ngrok.io
# Update webhook URL di Wagy ke: https://xxxxx.ngrok.io/waai/webhook
```

### 2. Production Server
- Deploy ke server dengan public domain
- Setup reverse proxy (nginx/apache)
- Enable HTTPS with SSL certificate
- Update webhook URL di Wagy

### 3. Environment Production
```bash
# .env production
APP_ENV=production
PORT=3100
DB_HOST=prod-db-host
WAGY_DEVICE_ID=OFFICE-PROD-01
WAGY_TOKEN=prod-token
WAGY_WEBHOOK_SECRET=prod-secret
ANTHROPIC_API_KEY=prod-key
REDIS_URL=redis://prod-redis:6379
```

## Troubleshooting

### 1. "Invalid Signature" Error
```
Kemungkinan:
- Webhook secret tidak cocok antara Wagy dan .env
- Raw body tidak diproses dengan benar
- Header X-Wagy-Signature tidak dikirim

Solusi:
- Double check WAGY_WEBHOOK_SECRET di .env
- Pastikan header X-Wagy-Signature ada di request
- Check server logs untuk detail error
```

### 2. Webhook URL tidak terverifikasi
```
Kemungkinan:
- URL tidak accessible dari internet
- Firewall memblokir port
- SSL certificate tidak valid

Solusi:
- Test: curl https://your-domain.com/waai/webhook?challenge=test
- Check firewall rules
- Verify SSL certificate
```

### 3. Pesan tidak dibalas
```
Kemungkinan:
- Nomor WhatsApp tidak terdaftar di wa_contacts
- Redis tidak connected
- Anthropic API error
- Timeout saat processing

Solusi:
- Check: SELECT * FROM wa_contacts WHERE phone = '628xxx'
- Check Redis: redis-cli ping
- Check logs untuk Anthropic errors
- Increase timeout di ai.go (default 30s)
```

### 4. Session tidak tersimpan
```
Kemungkinan:
- Redis not running
- REDIS_URL configuration salah
- Permissions issue

Solusi:
- redis-cli ping
- Check REDIS_URL format: redis://host:port
- Check Redis logs
```

## Monitoring

### Check Active Sessions
```bash
redis-cli
KEYS "waai:session:*"
GET "waai:session:628123456789"
```

### Monitor Logs
```bash
# Real-time logs
tail -f server.log | grep WAAI

# Count messages per phone
grep "Incoming message from" server.log | cut -d' ' -f7 | sort | uniq -c
```

### Health Check
```bash
curl http://localhost:3100/waai/admin/health
# Response: {"status":"ok","module":"waai"}
```

## Best Practices

1. **Test di Development Dulu**: Jangan langsung ke production
2. **Monitor Redis Usage**: Session bisa accumulate, setup cleanup jika perlu
3. **Log Review Regular**: Check error patterns
4. **Backup Database**: wa_contacts bisa menjadi critical data
5. **API Rate Limiting**: Anthropic memiliki rate limit, implement retry logic
6. **Webhook Timeout**: Server harus respond < 3 detik

## API Limits & Quotas

### Wagy
- Message rate: Check dokumentasi Wagy
- Device limit: Sesuai plan subscription
- Webhook delivery: Retry up to 3 times

### Anthropic
- Rate limits: Sesuai plan
- Token limits: claude-sonnet-4-6 supports 200k tokens
- Retry strategy: Built-in dengan exponential backoff

## Contoh Use Cases

### 1. Check Ketersediaan Bus
```
User: "Cek ketersediaan bus untuk 15-20 Juni"
WAAI: Calls get_fleet_availability
Response: "Tersedia 8 unit untuk tanggal tersebut"
```

### 2. Lihat Booking
```
User: "Berapa banyak booking hari ini?"
WAAI: Calls get_business_snapshot
Response: "Hari ini ada 3 booking confirmed"
```

### 3. Revenue Report
```
User: "Revenue minggu ini berapa?"
WAAI: Calls get_revenue_summary with period=weekly
Response: "Revenue minggu ini: Rp 12.5 juta"
```

## Support & Debugging

Untuk debugging lebih detail, edit `internal/waai/handler.go`:

```go
// Add verbose logging
log.Printf("[WAAI-DEBUG] Payload: %+v", payload)
log.Printf("[WAAI-DEBUG] Tenant: %+v", tenant)
log.Printf("[WAAI-DEBUG] AI Response: %s", response)
```

Rebuild dan run kembali:
```bash
go build -o main.exe
./main.exe
```
