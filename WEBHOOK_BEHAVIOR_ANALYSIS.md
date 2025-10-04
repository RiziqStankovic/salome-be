# Analisis Perilaku Webhook Salome BE

## Overview

Dokumentasi ini menjelaskan apa yang terjadi di sisi Salome BE ketika Cloudfren Core memanggil API webhook dengan berbagai payload.

## 🔍 **Skenario Testing**

### **1. Payload Lengkap (Normal Case)**

**Request:**

```json
{
  "order_id": "SALO-TOPUP-123456",
  "transaction_status": "settlement",
  "payment_type": "credit_card",
  "gross_amount": "50000.00",
  "fraud_status": "accept",
  "signature_key": "midtrans_signature_key_here",
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

**Response:**

```json
{
  "success": true,
  "message": "Webhook processed successfully",
  "order_id": "SALO-TOPUP-123456",
  "status": "completed"
}
```

**Yang Terjadi:**

1. ✅ **Parse JSON** - Berhasil parse semua field
2. ✅ **Validasi** - `order_id` dan `transaction_status` ada
3. ✅ **Cek Transaksi** - Cari di database berdasarkan `payment_reference`
4. ✅ **Update Status** - Update status transaksi menjadi `completed`
5. ✅ **Update Balance** - Tambah balance user (jika top-up)
6. ✅ **Update Group** - Update status grup member (jika group payment)

---

### **2. Payload Minimal (Hanya Field Wajib)**

**Request:**

```json
{
  "order_id": "SALO-TOPUP-789012",
  "transaction_status": "settlement"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Webhook processed successfully",
  "order_id": "SALO-TOPUP-789012",
  "status": "completed"
}
```

**Yang Terjadi:**

1. ✅ **Parse JSON** - Berhasil parse (field lain kosong)
2. ✅ **Validasi** - `order_id` dan `transaction_status` ada
3. ✅ **Cek Transaksi** - Cari di database berdasarkan `payment_reference`
4. ✅ **Update Status** - Update status transaksi menjadi `completed`
5. ✅ **Update Balance** - Tambah balance user (jika top-up)
6. ✅ **Update Group** - Update status grup member (jika group payment)

---

### **3. Payload Kurang (Missing Required Fields)**

**Request:**

```json
{
  "order_id": "SALO-TOPUP-123456"
  // Missing transaction_status
}
```

**Response:**

```json
{
  "error": "transaction_status is required"
}
```

**Yang Terjadi:**

1. ❌ **Parse JSON** - Berhasil parse
2. ❌ **Validasi** - `transaction_status` tidak ada
3. ❌ **Return Error** - HTTP 400 Bad Request
4. ❌ **Tidak Ada Update** - Database tidak diupdate

---

### **4. Payload Kosong (Empty JSON)**

**Request:**

```json
{}
```

**Response:**

```json
{
  "error": "Invalid webhook payload",
  "detail": "Key: 'WebhookRequest.OrderID' Error:Field validation for 'OrderID' failed on the 'required' tag"
}
```

**Yang Terjadi:**

1. ❌ **Parse JSON** - Berhasil parse
2. ❌ **Validasi** - `order_id` tidak ada
3. ❌ **Return Error** - HTTP 400 Bad Request
4. ❌ **Tidak Ada Update** - Database tidak diupdate

---

### **5. Order ID Tidak Ditemukan**

**Request:**

```json
{
  "order_id": "SALO-TOPUP-999999",
  "transaction_status": "settlement"
}
```

**Response:**

```json
{
  "error": "Transaction not found"
}
```

**Yang Terjadi:**

1. ✅ **Parse JSON** - Berhasil parse
2. ✅ **Validasi** - Field wajib ada
3. ❌ **Cek Transaksi** - Tidak ditemukan di database
4. ❌ **Return Error** - HTTP 404 Not Found
5. ❌ **Tidak Ada Update** - Database tidak diupdate

---

### **6. Payload Invalid JSON**

**Request:**

```json
{
  "order_id": "SALO-TOPUP-123456",
  "transaction_status": "settlement",
  // Missing closing brace
```

**Response:**

```json
{
  "error": "Invalid webhook payload",
  "detail": "invalid character '}' looking for beginning of object key string"
}
```

**Yang Terjadi:**

1. ❌ **Parse JSON** - Gagal parse JSON
2. ❌ **Return Error** - HTTP 400 Bad Request
3. ❌ **Tidak Ada Update** - Database tidak diupdate

---

## 🔄 **Status Mapping**

| Midtrans Status | Salome Status | Action                      |
| --------------- | ------------- | --------------------------- |
| `capture`       | `completed`   | Update balance/group status |
| `settlement`    | `completed`   | Update balance/group status |
| `cancel`        | `failed`      | Log failure                 |
| `deny`          | `failed`      | Log failure                 |
| `expire`        | `failed`      | Log failure                 |
| `pending`       | `pending`     | Keep pending                |
| `other`         | `failed`      | Default to failed           |

---

## 📊 **Log Output**

### **Success Case:**

```
Webhook received from Cloudfren Core: OrderID=SALO-TOPUP-123456, Status=settlement, Amount=50000.00
Mapped transaction status: settlement -> completed
Updated transaction status: SALO-TOPUP-123456 -> completed
Updated user balance: UserID=user123, Amount=50000
```

### **Error Case:**

```
Error parsing webhook payload: invalid character '}' looking for beginning of object key string
```

### **Transaction Not Found:**

```
Webhook received from Cloudfren Core: OrderID=SALO-TOPUP-999999, Status=settlement, Amount=
Mapped transaction status: settlement -> completed
Transaction not found: SALO-TOPUP-999999
```

---

## 🛡️ **Error Handling**

### **1. Validation Errors (HTTP 400)**

- Missing `order_id`
- Missing `transaction_status`
- Invalid JSON format

### **2. Not Found Errors (HTTP 404)**

- Transaction tidak ditemukan di database

### **3. Server Errors (HTTP 500)**

- Database connection error
- Database query error
- Internal server error

---

## 🔧 **Database Operations**

### **1. Check Transaction Exists**

```sql
SELECT EXISTS(SELECT 1 FROM transactions WHERE payment_reference = $1)
```

### **2. Update Transaction Status**

```sql
UPDATE transactions
SET status = $1, updated_at = $2
WHERE payment_reference = $3
```

### **3. Update User Balance (Top-up)**

```sql
UPDATE users
SET balance = balance + $1, updated_at = $2
WHERE id = $3
```

### **4. Update Group Member Status (Group Payment)**

```sql
UPDATE group_members
SET user_status = 'paid', paid_at = $1, updated_at = $2
WHERE group_id = $3 AND user_id = $4
```

---

## 🧪 **Testing Commands**

### **Test Payload Lengkap:**

```bash
curl -X POST http://localhost:3000/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-123456",
    "transaction_status": "settlement",
    "gross_amount": "50000.00"
  }'
```

### **Test Payload Minimal:**

```bash
curl -X POST http://localhost:3000/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-789012",
    "transaction_status": "settlement"
  }'
```

### **Test Payload Kurang:**

```bash
curl -X POST http://localhost:3000/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-123456"
  }'
```

### **Test Order Tidak Ditemukan:**

```bash
curl -X POST http://localhost:3000/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-999999",
    "transaction_status": "settlement"
  }'
```

---

## 📈 **Monitoring & Debugging**

### **1. Log Levels**

- **INFO**: Webhook received, status mapping, successful updates
- **ERROR**: Parse errors, validation errors, database errors
- **DEBUG**: Detailed transaction processing steps

### **2. Response Codes**

- **200**: Success
- **400**: Bad Request (validation error)
- **404**: Not Found (transaction not found)
- **500**: Internal Server Error

### **3. Health Check**

```bash
curl -X GET http://localhost:3000/webhook/health
```

**Response:**

```json
{
  "status": "healthy",
  "service": "salome-webhook",
  "timestamp": 1704067200
}
```
