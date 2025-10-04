# Webhook Integration dengan Cloudfren Core

## Overview

Sistem webhook terintegrasi antara Midtrans → Cloudfren Core → Salome untuk menangani notifikasi pembayaran.

## Alur Webhook

```
Midtrans → Cloudfren Core → Salome
```

### 1. Order ID Format

- **Cloudfren Core**: `ORDER-{user_id}-{timestamp}` atau format lain
- **Salome**: `salo-{type}-{id}`
  - Top-up: `salo-topup-{6digit_id}`
  - Group payment: `salo-grp-{6digit_id}`

### 2. Webhook Endpoints

#### Cloudfren Core (Webhook utama dari Midtrans)

```
POST /webhook/midtrans
```

#### Salome (Webhook dari Cloudfren Core)

```
POST /webhook/cloudfren
GET /webhook/health
```

## Implementasi di Cloudfren Core

### 1. Modifikasi Webhook Handler

```go
func (h *WebhookHandler) HandleMidtransWebhook(c *gin.Context) {
    var notification map[string]interface{}
    if err := c.ShouldBindJSON(&notification); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification"})
        return
    }

    orderID, ok := notification["order_id"].(string)
    if !ok {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
        return
    }

    // Check if this is a Salome order
    if strings.HasPrefix(orderID, "salo-") {
        // Forward to Salome webhook
        err := h.forwardToSalome(notification)
        if err != nil {
            log.Printf("Error forwarding to Salome: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward to Salome"})
            return
        }
    } else {
        // Handle Cloudfren Core order
        err := h.handleCloudfrenOrder(notification)
        if err != nil {
            log.Printf("Error handling Cloudfren order: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to handle order"})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *WebhookHandler) forwardToSalome(notification map[string]interface{}) error {
    salomeWebhookURL := "https://salome.cloudfren.id/webhook/cloudfren"

    jsonData, err := json.Marshal(notification)
    if err != nil {
        return err
    }

    resp, err := http.Post(salomeWebhookURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Salome webhook returned status: %d", resp.StatusCode)
    }

    return nil
}
```

### 2. Konfigurasi

Tambahkan URL Salome webhook di konfigurasi Cloudfren Core:

```yaml
webhooks:
  salome:
    url: "https://salome.cloudfren.id/webhook/cloudfren"
    timeout: 30s
    retry_attempts: 3
    retry_delay: 5s
```

## Payload Webhook

### Request ke Salome

```json
{
  "order_id": "salo-topup-123456",
  "transaction_status": "settlement",
  "payment_type": "credit_card",
  "gross_amount": "50000.00",
  "fraud_status": "accept",
  "signature_key": "...",
  "status_message": "midtrans payment notification",
  "merchant_id": "G123456789",
  "transaction_time": "2024-01-01 12:00:00",
  "transaction_id": "1234567890",
  "bank": "bca",
  "channel": "credit_card",
  "approval_code": "123456",
  "currency": "IDR"
}
```

### Response dari Salome

```json
{
  "success": true,
  "message": "Webhook processed successfully"
}
```

## Status Mapping

| Midtrans Status            | Salome Status | Action                      |
| -------------------------- | ------------- | --------------------------- |
| `capture`, `settlement`    | `completed`   | Update balance/group status |
| `cancel`, `deny`, `expire` | `failed`      | Log failure                 |
| `pending`                  | `pending`     | Keep pending                |

## Testing

### 1. Test Health Check

```bash
curl -X GET https://salome.cloudfren.id/webhook/health
```

### 2. Test Webhook

```bash
curl -X POST https://salome.cloudfren.id/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "salo-topup-123456",
    "transaction_status": "settlement",
    "gross_amount": "50000.00"
  }'
```

## Monitoring

### Logs

- Cloudfren Core: Log semua webhook yang diterima dan diteruskan
- Salome: Log webhook yang diterima dan status update

### Error Handling

- Jika Salome webhook gagal, Cloudfren Core akan retry sesuai konfigurasi
- Jika retry gagal, log error dan notifikasi admin

## Security

### 1. IP Whitelist

- Cloudfren Core: Hanya terima webhook dari Midtrans IP
- Salome: Hanya terima webhook dari Cloudfren Core IP

### 2. Signature Verification

- Cloudfren Core: Verifikasi signature Midtrans
- Salome: Verifikasi signature dari Cloudfren Core (opsional)

### 3. Rate Limiting

- Implementasi rate limiting untuk mencegah spam
- Cloudfren Core: 100 requests/minute
- Salome: 200 requests/minute
