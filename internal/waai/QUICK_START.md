# WhatsApp AI Assistant (WAAI) - Quick Start Guide

Panduan cepat untuk memulai integrasi WhatsApp AI Assistant dengan Travego ERP.

## 30 Detik Setup

### 1. Copy environment variables ke .env
```bash
WAGY_DEVICE_ID=OFFICE-01
WAGY_TOKEN=your-wagy-token
WAGY_WEBHOOK_SECRET=your-webhook-secret
ANTHROPIC_API_KEY=sk-ant-v1-...
REDIS_URL=redis://localhost:6379
```

### 2. Run database migration
```bash
psql -U postgres -d traveGo -f database/migrations/001_create_wa_contacts.sql
```

### 3. Add test contact
```sql
INSERT INTO wa_contacts (phone, name, role, organization_id, is_active)
VALUES ('628123456789', 'Your Name', 'admin', 1, true);
```

### 4. Build & run
```bash
go build
./main.exe
```

Selesai! Server akan log: `[WAAI] Routes registered successfully`

## Quick Test

### Test webhook verification
```bash
curl "http://localhost:3100/waai/webhook?challenge=test123"
# Response: test123
```

### Test invalid signature
```bash
curl -X POST http://localhost:3100/waai/webhook \
  -H "X-Wagy-Signature: invalid" \
  -H "Content-Type: application/json" \
  -d '{"event": "message.received"}'
# Response: 401 Unauthorized
```

## Files Overview

| File | Purpose |
|------|---------|
| `config.go` | Load env vars |
| `handler.go` | Fiber routes (GET/POST webhook) |
| `wagy.go` | Send message via Wagy API |
| `ai.go` | Call Anthropic + tool handling |
| `session.go` | Redis conversation storage |
| `tenant.go` | DB lookup & business snapshot |
| `tools.go` | Tool definitions |
| `tools_executor.go` | Tool implementation & queries |
| `webhook.go` | HMAC verification & parsing |

## API Endpoints

```
GET  /waai/webhook?challenge={id}          # Wagy verification
POST /waai/webhook                          # Incoming messages
DELETE /waai/admin/session/{phone}         # Clear session
GET  /waai/admin/health                    # Health check
```

## Workflow

```
User sends WhatsApp message
          ↓
Wagy POST → /waai/webhook
          ↓
Verify HMAC signature
          ↓
Parse JSON payload
          ↓
Lookup tenant in DB
          ↓
Return 200 (async processing starts)
          ↓
Load conversation history from Redis
          ↓
Send to Anthropic with tools
          ↓
Execute tools if needed (loop max 5x)
          ↓
Save history to Redis
          ↓
Send response via Wagy
```

## Next Steps

1. **Register Webhook with Wagy**: See `WAGY_SETUP.md`
2. **Add More Contacts**: Insert into `wa_contacts` table
3. **Customize System Prompt**: Edit `ai.go` buildSystemPrompt()
4. **Add Custom Tools**: Follow `tools_executor.go` pattern

## Environment Variables Checklist

- [ ] `WAGY_DEVICE_ID` - Device ID from Wagy
- [ ] `WAGY_TOKEN` - API token from Wagy
- [ ] `WAGY_WEBHOOK_SECRET` - Random secret for HMAC
- [ ] `ANTHROPIC_API_KEY` - API key from Anthropic
- [ ] `REDIS_URL` - Redis connection string
- [ ] `REDIS_HOST` - Redis host (optional)
- [ ] `REDIS_PORT` - Redis port (optional)

## Troubleshooting

**Issue**: "Routes not registered"
- Check if Redis is connected
- Check env vars are set
- Check logs for specific errors

**Issue**: "Tenant not found"
- Verify phone in `wa_contacts`
- Format: `628123456789` (no @s.whatsapp.net)
- Check `is_active = true`

**Issue**: "Signature verification failed"
- Verify `WAGY_WEBHOOK_SECRET` matches Wagy settings
- Check raw body is not modified before verification

**Issue**: No response from AI
- Check `ANTHROPIC_API_KEY` is valid
- Check rate limits not exceeded
- Check timeout (30 seconds default)

## Example Conversation

```
User: "Cek ketersediaan bus untuk 15-20 Juni"

WAAI:
1. Load session history
2. Call Anthropic with context
3. Anthropic uses get_fleet_availability tool
4. Query database for available units
5. Return result to Anthropic
6. Anthropic generates response
7. Save to Redis
8. Send via Wagy

Response: "Tersedia 8 unit untuk tanggal 15-20 Juni"
```

## Performance Notes

- Webhook returns in < 1 sec (AI processing is async)
- Session TTL: 24 hours
- Max tool iterations: 5
- AI timeout: 30 seconds
- Response size: Recommend < 2000 chars for WhatsApp

## For Development

### Enable Debug Logging
Edit `handler.go`:
```go
log.Printf("[WAAI-DEBUG] Phone: %s", phone)
log.Printf("[WAAI-DEBUG] Tenant: %+v", tenant)
log.Printf("[WAAI-DEBUG] AI Response: %s", response)
```

### Test with Mock Data
Edit `ai.go` to use `MockToolExecutor`:
```go
// toolExec: NewMockToolExecutor()
```

### Monitor Sessions
```bash
redis-cli
KEYS "waai:session:*"
GET "waai:session:628123456789"
```

## File Structure

```
internal/waai/
├── config.go              # Config management
├── handler.go             # HTTP handlers
├── wagy.go                # Wagy API client
├── ai.go                  # Anthropic integration
├── session.go             # Redis session
├── tenant.go              # Tenant lookup
├── tools.go               # Tool definitions
├── tools_executor.go      # Tool implementation
├── webhook.go             # Webhook verification
├── README.md              # Detailed documentation
├── WAGY_SETUP.md          # Wagy configuration guide
└── QUICK_START.md         # This file
```

## Support

- Full docs: `README.md`
- Wagy setup: `WAGY_SETUP.md`
- Check logs: `grep WAAI server.log`
- Database issues: Check `wa_contacts` table
- API issues: Check `.env` variables
