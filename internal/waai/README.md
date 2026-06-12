# WhatsApp AI Assistant (WAAI) Module

Module independen untuk integrasi WhatsApp AI Assistant dengan sistem Travego ERP menggunakan Wagy API dan Anthropic Claude.

## Fitur Utama

- **Webhook Integration**: Menerima dan memproses pesan WhatsApp dari Wagy
- **Conversation Management**: Menyimpan dan mengelola riwayat percakapan per pengguna di Redis
- **AI Processing**: Menggunakan Claude API dengan function calling untuk memberikan respons cerdas
- **Multi-tenant Support**: Mendukung multiple organisasi dengan mapping nomor WhatsApp ke tenant
- **Tool Integration**: Terintegrasi dengan tools ERP (fleet availability, bookings, revenue, dll)

## Struktur Folder

```
internal/waai/
├── handler.go       # Fiber route handlers
├── wagy.go          # Wagy API client
├── session.go       # Redis session management
├── ai.go            # Anthropic AI integration
├── tenant.go        # Database tenant lookup
├── tools.go         # Tool definitions & execution
├── webhook.go       # Webhook verification & parsing
└── config.go        # Configuration management
```

## Setup & Konfigurasi

### 1. Environment Variables

Tambahkan ke file `.env`:

```bash
# Wagy WhatsApp Integration
WAGY_DEVICE_ID=OFFICE-01
WAGY_TOKEN=your-wagy-token
WAGY_WEBHOOK_SECRET=your-webhook-secret

# Anthropic API
ANTHROPIC_API_KEY=your-anthropic-api-key

# Redis (sudah ada, pastikan konfigurasi benar)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_URL=redis://localhost:6379
```

### 2. Database Schema

Pastikan tabel `wa_contacts` dan `organizations` sudah ada:

```sql
CREATE TABLE IF NOT EXISTS wa_contacts (
    id BIGINT PRIMARY KEY,
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100),
    role VARCHAR(50),
    organization_id BIGINT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS organizations (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 3. Registrasi Routes

Routes sudah otomatis terdaftar di `routes/routes.go`:

```go
// Setup WhatsApp AI Assistant module (WAAI)
rdb := helper.GetRedisClient()
if rdb != nil {
    waaiCfg := waai.LoadConfig()
    if err := waai.RegisterRoutes(app, waaiCfg, db, cfg.Database.Driver, rdb); err != nil {
        log.Printf("Warning: Failed to register WAAI routes: %v", err)
    }
}
```

## API Endpoints

### Webhook Verification (GET)
```
GET /waai/webhook?challenge={challenge}
```
Digunakan Wagy untuk verifikasi URL saat registrasi webhook.

### Incoming Messages (POST)
```
POST /waai/webhook
Header: X-Wagy-Signature: HMAC-SHA256(body, secret)
Body: JSON payload dari Wagy
```

Memproses pesan WhatsApp masuk, membalas dengan respons AI.

### Admin - Clear Session (DELETE)
```
DELETE /waai/admin/session/{phone}
```
Menghapus riwayat percakapan untuk nomor WhatsApp tertentu.

### Admin - Health Check (GET)
```
GET /waai/admin/health
```
Memeriksa status modul WAAI.

## Alur Kerja

### 1. Pesan Masuk
```
Wagy → POST /waai/webhook
  ↓
Verifikasi HMAC-SHA256 signature
  ↓
Parse WebhookPayload
  ↓
Ekstrak nomor pengirim & pesan
  ↓
Lookup tenant di DB
  ↓
Return 200 immediately to Wagy
  ↓ (background goroutine)
Load session history dari Redis
  ↓
Send ke Anthropic API dengan context
  ↓
Proses tool_use jika ada (max 5 iterasi)
  ↓
Save updated history ke Redis
  ↓
Send respons ke Wagy
```

### 2. Tool Execution

Saat Anthropic mengembalikan `tool_use`, modul mengeksekusi tool dan mengirim hasil kembali:

- **get_business_snapshot**: Ringkasan bisnis hari ini
- **get_fleet_availability**: Ketersediaan armada
- **get_booking_list**: Daftar booking
- **get_revenue_summary**: Ringkasan revenue

## Struktur Data

### ConversationMessage
```json
{
  "role": "user|assistant",
  "content": "text atau array of objects dengan tool use/result"
}
```

### WebhookPayload
```json
{
  "event": "message.received",
  "source": "whatsapp",
  "data": {
    "id": 1001,
    "device_id": "OFFICE-01",
    "owner_jid": "62812345678@s.whatsapp.net",
    "content": {
      "pn_jid": "628999888777@s.whatsapp.net",
      "content": "Halo, cek ketersediaan bus",
      "message_id": "ABC123XYZ",
      "timestamp": "2026-05-13T07:05:00Z"
    }
  }
}
```

## Session Management

- **Key Format**: `waai:session:{phone}`
- **TTL**: 24 jam (refresh otomatis setiap pesan baru)
- **Value**: JSON array of ConversationMessage
- **Storage**: Redis

Contoh session:
```json
[
  {"role": "user", "content": "Halo, cek ketersediaan bus"},
  {"role": "assistant", "content": "Tentu, saya akan cek ketersediaan..."},
  {"role": "user", "content": "Untuk tanggal 15-20 Juni"}
]
```

## Error Handling

### Tenant Not Found
```
Response: "Maaf, nomor Anda belum terdaftar dalam sistem. Hubungi administrator untuk pendaftaran."
```

### Invalid Signature
```
Status: 401 Unauthorized
Response: {"error": "Invalid signature"}
```

### AI Processing Error
```
Response: "Maaf, terjadi kesalahan saat memproses permintaan Anda. Silakan coba lagi."
```

## Performance & Optimization

1. **Async Processing**: AI processing dilakukan di background goroutine dengan timeout 30 detik
2. **Quick Response**: Webhook returns 200 dalam < 1 detik (sebelum AI processing selesai)
3. **Session Caching**: Riwayat percakapan disimpan di Redis untuk performa query yang cepat
4. **Tool Use Loop Limit**: Max 5 iterasi untuk prevent infinite loops

## Security

1. **HMAC-SHA256 Verification**: Setiap webhook diverifikasi dengan signature
2. **Tenant Isolation**: Setiap user hanya bisa akses data organisasi mereka sendiri
3. **Context Injection**: System prompt includes org name & user role untuk context awareness
4. **No Shared State**: Module sepenuhnya independen dari kode ERP lain

## Logging

Semua event dicatat dengan prefix `[WAAI]`:
```
[WAAI] Incoming message from 628123456789: Halo
[WAAI] Tenant not found for phone: 628111111111
[WAAI] Message sent to 628123456789
[WAAI] Error processing message from 628123456789: ...
```

## Testing

### 1. Test Webhook Verification
```bash
curl "http://localhost:3100/waai/webhook?challenge=test123"
# Response: test123
```

### 2. Test Invalid Signature
```bash
curl -X POST http://localhost:3100/waai/webhook \
  -H "X-Wagy-Signature: invalid" \
  -d '{"event": "message.received"}'
# Response: 401 Unauthorized
```

### 3. Clear Session
```bash
curl -X DELETE http://localhost:3100/waai/admin/session/628123456789
# Response: {"status": "session cleared"}
```

## Dependencies

```go
github.com/redis/go-redis/v9  // Session storage
github.com/gofiber/fiber/v2   // HTTP framework
```

Anthropic API diakses via HTTP (no SDK diperlukan di sini karena custom implementation).

## Maintenance

### Regular Tasks
- Monitor Redis usage untuk session storage
- Check log untuk error patterns
- Review tool execution untuk optimize performa

### Database Cleanup
```sql
-- Check inactive contacts
SELECT * FROM wa_contacts WHERE is_active = false AND created_at < NOW() - INTERVAL '90 days';

-- Archive old contacts jika diperlukan
UPDATE wa_contacts SET is_active = false WHERE last_message < NOW() - INTERVAL '180 days';
```

## Kontribusi & Extension

Untuk menambah tool baru:
1. Tambah definition di `tools.go` (GetToolDefinitions)
2. Tambah case di `ai.go` (executeTool)
3. Implementasi DB query yang sesuai
4. Test dengan berbagai input

## Support

Untuk bug report atau feature request:
1. Check log dengan prefix `[WAAI]`
2. Verify config di `.env`
3. Ensure database tables exist
4. Check Redis connection
