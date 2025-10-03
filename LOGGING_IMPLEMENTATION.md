    # Implementasi Custom Logging untuk Salome Backend

## Overview

Implementasi custom logging untuk GIN framework dengan format yang menampilkan informasi user dari JWT tanpa melakukan query database berulang.

## Format Log

```
[GIN] 2025/10/02 - 04:28:42 | 401 |   1.2834ms |   127.0.0.1 | GET   /api/v1/auth/profile | user=anonymous
[GIN] 2025/10/02 - 04:28:42 | 200 |   1.2834ms |   127.0.0.1 | GET   /api/v1/auth/profile | user=user@example.com
```

## Komponen yang Diimplementasikan

### 1. Custom Logging Middleware (`internal/middleware/logging.go`)

#### `CustomLoggingMiddleware()`

- Middleware logging custom yang menampilkan informasi user dari context
- Format: `[GIN] timestamp | status | latency | client_ip | method path | user_info`

#### `UserExtractionMiddleware()`

- Middleware yang mengekstrak informasi user dari JWT tanpa query database
- Menyimpan `user_id`, `user_email`, dan `user_authenticated` di context
- Tidak melakukan validasi database, hanya parsing JWT

#### Middleware yang Dioptimasi

##### `OptimizedAuthRequired()`

- Validasi JWT tanpa query database
- Hanya melakukan parsing dan validasi JWT
- Menyimpan user info di context

##### `OptimizedAuthRequiredWithStatus(db)`

- Validasi JWT + cek status user dengan 1 query database
- Menggantikan `AuthRequiredWithStatus` yang melakukan 2 query

##### `OptimizedAdminRequired(db)`

- Validasi JWT + cek admin status dengan 1 query database
- Menggantikan `AdminRequired` yang melakukan 2 query

### 2. Update Main Application (`main.go`)

```go
// Initialize Gin router
r := gin.New()

// Custom logging middleware
r.Use(middleware.CustomLoggingMiddleware())
r.Use(gin.Recovery())

// User extraction middleware (extracts user info from JWT for logging)
r.Use(middleware.UserExtractionMiddleware())

// CORS middleware
r.Use(middleware.CORS())
```

### 3. Update Routes (`internal/routes/routes.go`)

Semua route yang menggunakan middleware autentikasi telah diupdate untuk menggunakan middleware yang dioptimasi:

- `AuthRequired()` → `OptimizedAuthRequired()`
- `AuthRequiredWithStatus(db)` → `OptimizedAuthRequiredWithStatus(db)`
- `AdminRequired(db)` → `OptimizedAdminRequired(db)`

## Analisis JWT

JWT di salome-be menyimpan:

- `user_id` (UUID)
- `email` (string)
- `RegisteredClaims` (exp, iat, dll)

## Optimasi yang Dicapai

### Sebelum Optimasi

- Setiap request dengan auth melakukan 1-2 query database
- Logging tidak menampilkan informasi user
- Query database berulang untuk informasi yang sama

### Setelah Optimasi

- User extraction dari JWT tanpa query database
- Logging menampilkan email user atau "anonymous"
- Query database hanya untuk validasi status/admin (1 query)
- Performa lebih baik karena mengurangi query database

## Cara Kerja

1. **UserExtractionMiddleware** berjalan pertama untuk mengekstrak user info dari JWT
2. **CustomLoggingMiddleware** menggunakan user info dari context untuk logging
3. **OptimizedAuthRequired** melakukan validasi JWT tanpa query database
4. **OptimizedAuthRequiredWithStatus** melakukan 1 query untuk cek status user
5. **OptimizedAdminRequired** melakukan 1 query untuk cek admin status

## Keuntungan

1. **Performansi**: Mengurangi query database berulang
2. **Logging**: Informasi user terlihat di log untuk debugging
3. **Efisiensi**: JWT parsing lebih cepat daripada database query
4. **Monitoring**: Mudah melacak aktivitas user dari log

## Testing

Untuk test implementasi:

1. Jalankan server: `go run main.go`
2. Test endpoint dengan/ tanpa auth
3. Periksa log output untuk format yang benar
4. Monitor performa database query

## Catatan

- Middleware lama masih tersedia untuk backward compatibility
- User extraction middleware berjalan untuk semua request
- Logging format sesuai dengan permintaan: `user=anonymous` atau `user=email`
