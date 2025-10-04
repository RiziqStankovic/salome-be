# Webhook Testing Examples - Salome BE

## 🧪 **Contoh Curl untuk Testing Webhook Salome BE**

### **1. Test Payload Lengkap (Success Case)**

```bash
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-123456",
    "transaction_status": "settlement",
    "gross_amount": "50000.00",
    "payment_type": "credit_card",
    "fraud_status": "accept",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567890",
    "bank": "bca",
    "channel": "credit_card",
    "approval_code": "123456",
    "currency": "IDR"
  }'
```

**Expected Response:**

```json
{
  "success": true,
  "message": "Webhook processed successfully",
  "order_id": "SALO-TOPUP-123456",
  "status": "success",
  "payment_method": "credit_card"
}
```

### **2. Test Group Payment (Success Case)**

```bash
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-GRP-789012",
    "transaction_status": "settlement",
    "gross_amount": "25000.00",
    "payment_type": "bank_transfer",
    "fraud_status": "accept",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567891",
    "bank": "bca",
    "channel": "bank_transfer",
    "approval_code": "123457",
    "currency": "IDR"
  }'
```

**Expected Response:**

```json
{
  "success": true,
  "message": "Webhook processed successfully",
  "order_id": "SALO-GRP-789012",
  "status": "success",
  "payment_method": "bank_transfer"
}
```

### **3. Test Pending Status**

```bash
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-345678",
    "transaction_status": "pending",
    "gross_amount": "100000.00",
    "payment_type": "credit_card",
    "fraud_status": "accept",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567892",
    "bank": "bca",
    "channel": "credit_card",
    "approval_code": "123458",
    "currency": "IDR"
  }'
```

**Expected Response:**

```json
{
  "success": true,
  "message": "Webhook processed successfully",
  "order_id": "SALO-TOPUP-345678",
  "status": "pending",
  "payment_method": "credit_card"
}
```

### **4. Test Failed Status**

```bash
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-456789",
    "transaction_status": "cancel",
    "gross_amount": "75000.00",
    "payment_type": "credit_card",
    "fraud_status": "deny",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567893",
    "bank": "bca",
    "channel": "credit_card",
    "approval_code": "123459",
    "currency": "IDR"
  }'
```

**Expected Response:**

```json
{
  "success": true,
  "message": "Webhook processed successfully",
  "order_id": "SALO-TOPUP-456789",
  "status": "failed",
  "payment_method": "credit_card"
}
```

## 📊 **Status Mapping**

| Midtrans Status | Salome Status | Description        |
| --------------- | ------------- | ------------------ |
| `settlement`    | `success`     | Payment berhasil   |
| `capture`       | `success`     | Payment berhasil   |
| `pending`       | `pending`     | Payment menunggu   |
| `cancel`        | `failed`      | Payment dibatalkan |
| `deny`          | `failed`      | Payment ditolak    |
| `expire`        | `failed`      | Payment expired    |

## 🔍 **Expected Log Output**

### **Success Case:**

```
[SALOME BE] ===== WEBHOOK RECEIVED FROM CLOUDFREN CORE =====
[SALOME BE] Order ID: SALO-TOPUP-123456
[SALOME BE] Transaction Status: settlement
[SALOME BE] Payment Type: credit_card
[SALOME BE] Gross Amount: 50000.00
[SALOME BE] Fraud Status: accept
[SALOME BE] Signature Key: test_signature_key_here
[SALOME BE] Status Message: midtrans payment notification
[SALOME BE] Merchant ID: G123456789
[SALOME BE] Transaction Time: 2024-01-01 12:00:00
[SALOME BE] Transaction ID: 1234567890
[SALOME BE] Bank: bca
[SALOME BE] Channel: credit_card
[SALOME BE] Approval Code: 123456
[SALOME BE] Currency: IDR
[SALOME BE] ================================================
[SALOME BE] Mapped transaction status: settlement -> success
[SALOME BE] Updated transaction status: SALO-TOPUP-123456 -> success, payment_method: credit_card
[SALOME BE] Current balance before update: UserID=user123, CurrentBalance=100000, AmountToAdd=50000
[SALOME BE] Updated user balance: UserID=user123, OldBalance=100000, AmountAdded=50000, NewBalance=150000
[SALOME BE] ===== RESPONSE TO CLOUDFREN CORE =====
[SALOME BE] Status Code: 200 OK
[SALOME BE] Response: map[message:Webhook processed successfully order_id:SALO-TOPUP-123456 status:success payment_method:credit_card success:true]
[SALOME BE] ======================================
```

### **Pending Case:**

```
[SALOME BE] ===== WEBHOOK RECEIVED FROM CLOUDFREN CORE =====
[SALOME BE] Order ID: SALO-TOPUP-345678
[SALOME BE] Transaction Status: pending
[SALOME BE] Payment Type: credit_card
[SALOME BE] Gross Amount: 100000.00
[SALOME BE] Fraud Status: accept
[SALOME BE] Signature Key: test_signature_key_here
[SALOME BE] Status Message: midtrans payment notification
[SALOME BE] Merchant ID: G123456789
[SALOME BE] Transaction Time: 2024-01-01 12:00:00
[SALOME BE] Transaction ID: 1234567892
[SALOME BE] Bank: bca
[SALOME BE] Channel: credit_card
[SALOME BE] Approval Code: 123458
[SALOME BE] Currency: IDR
[SALOME BE] ================================================
[SALOME BE] Mapped transaction status: pending -> pending
[SALOME BE] Updated transaction status: SALO-TOPUP-345678 -> pending, payment_method: credit_card
[SALOME BE] ===== RESPONSE TO CLOUDFREN CORE =====
[SALOME BE] Status Code: 200 OK
[SALOME BE] Response: map[message:Webhook processed successfully order_id:SALO-TOPUP-345678 status:pending payment_method:credit_card success:true]
[SALOME BE] ======================================
```

## 🧪 **Testing Script**

Buat file `test_webhook_success.sh`:

```bash
#!/bin/bash

echo "Testing Salome BE Webhook - Success Cases..."

# Test 1: Top-up success
echo "Test 1: Top-up success (settlement -> success)"
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-123456",
    "transaction_status": "settlement",
    "gross_amount": "50000.00",
    "payment_type": "credit_card",
    "fraud_status": "accept",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567890",
    "bank": "bca",
    "channel": "credit_card",
    "approval_code": "123456",
    "currency": "IDR"
  }'

echo -e "\n\n"

# Test 2: Group payment success
echo "Test 2: Group payment success (settlement -> success)"
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-GRP-789012",
    "transaction_status": "settlement",
    "gross_amount": "25000.00",
    "payment_type": "bank_transfer",
    "fraud_status": "accept",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567891",
    "bank": "bca",
    "channel": "bank_transfer",
    "approval_code": "123457",
    "currency": "IDR"
  }'

echo -e "\n\n"

# Test 3: Pending status
echo "Test 3: Pending status (pending -> pending)"
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-345678",
    "transaction_status": "pending",
    "gross_amount": "100000.00",
    "payment_type": "credit_card",
    "fraud_status": "accept",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567892",
    "bank": "bca",
    "channel": "credit_card",
    "approval_code": "123458",
    "currency": "IDR"
  }'

echo -e "\n\n"

# Test 4: Failed status
echo "Test 4: Failed status (cancel -> failed)"
curl -X POST http://localhost:8080/webhook/cloudfren \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "SALO-TOPUP-456789",
    "transaction_status": "cancel",
    "gross_amount": "75000.00",
    "payment_type": "credit_card",
    "fraud_status": "deny",
    "signature_key": "test_signature_key_here",
    "status_message": "midtrans payment notification",
    "merchant_id": "G123456789",
    "transaction_time": "2024-01-01 12:00:00",
    "transaction_id": "1234567893",
    "bank": "bca",
    "channel": "credit_card",
    "approval_code": "123459",
    "currency": "IDR"
  }'

echo -e "\n\nTest completed!"
```

## 🔧 **Database Updates**

### **Transaction Table Updates:**

- `status` → `success` (bukan `completed`)
- `payment_method` → diisi dengan `payment_type` dari webhook
- `updated_at` → timestamp saat ini

### **User Balance Updates (untuk top-up):**

- Hanya terjadi jika `status = "success"` dan `order_id` dimulai dengan `"SALO-TOPUP-"`
- Mengambil `amount` dari kolom `amount` di tabel `transactions` berdasarkan `payment_reference`
- Menjumlahkan ke kolom `balance` di tabel `users` dengan `WHERE user_id = transaction.user_id`
- SQL: `UPDATE users SET balance = balance + $1 WHERE id = $2`
- Logging: Menampilkan balance sebelum dan sesudah update

### **Group Member Updates (untuk group payment):**

- Hanya terjadi jika `status = "success"` dan `order_id` dimulai dengan `"SALO-GRP-"`
- Mengambil `user_id` dan `group_id` dari tabel `transactions` berdasarkan `payment_reference`
- `user_status` → `'paid'`
- `paid_at` → timestamp saat ini

## 📝 **Summary Changes**

1. ✅ **Status Mapping**: `settlement`/`capture` → `success` (bukan `completed`)
2. ✅ **Payment Method**: Update `payment_method` dengan `payment_type` dari webhook
3. ✅ **Response**: Tambahkan `payment_method` di response
4. ✅ **Logging**: Update log untuk menampilkan payment method
5. ✅ **Balance Update**: Hanya untuk `status = "success"`
6. ✅ **Group Update**: Hanya untuk `status = "success"`
