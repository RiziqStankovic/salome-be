# CORS Configuration

## Overview

SALOME backend sekarang mendukung multiple CORS origins untuk fleksibilitas deployment yang lebih baik.

## Configuration

### YAML Configuration (app.yaml)

```yaml
server:
  port: 8080
  host: localhost
  cors_origins:
    - http://localhost:3000
    - http://localhost:3001
    - https://salome.vercel.app
    - https://salome-dev.vercel.app
    - https://salome-staging.vercel.app
```

### Environment Variables

- `CORS_ORIGIN`: Mengembalikan origin pertama dari array CORS origins (untuk backward compatibility)

## Features

### 1. Multiple Origins Support

- Backend dapat menerima request dari multiple origins yang dikonfigurasi
- Setiap origin dalam array `cors_origins` akan diizinkan untuk mengakses API

### 2. Dynamic Origin Validation

- Middleware CORS akan memvalidasi request origin terhadap daftar origins yang diizinkan
- Jika origin tidak ditemukan dalam daftar, akan menggunakan origin pertama sebagai fallback

### 3. Backward Compatibility

- Konfigurasi lama dengan `cors_origin` (singular) masih didukung
- Environment variable `CORS_ORIGIN` tetap berfungsi

## Implementation Details

### Config Structure

```go
type ServerConfig struct {
    Port        int      `yaml:"port"`
    Host        string   `yaml:"host"`
    CORSOrigins []string `yaml:"cors_origins"`
}
```

### CORS Middleware

- Memvalidasi request origin terhadap allowed origins
- Mengatur header CORS yang sesuai
- Mendukung credentials dan multiple HTTP methods

## Usage Examples

### Development

```yaml
cors_origins:
  - http://localhost:3000
  - http://localhost:3001
  - http://127.0.0.1:3000
```

### Production

```yaml
cors_origins:
  - https://salome.vercel.app
  - https://salome-dev.vercel.app
  - https://salome-staging.vercel.app
```

### Mixed Environment

```yaml
cors_origins:
  - http://localhost:3000
  - https://salome.vercel.app
  - https://salome-dev.vercel.app
```

## Security Considerations

1. **Production**: Hanya gunakan HTTPS origins untuk production
2. **Development**: Localhost origins dapat digunakan untuk development
3. **Validation**: Selalu validasi origins yang diizinkan
4. **Credentials**: CORS credentials diizinkan untuk authenticated requests

## Migration Guide

### From Single Origin

Jika sebelumnya menggunakan:

```yaml
cors_origin: http://localhost:3000
```

Ubah menjadi:

```yaml
cors_origins:
  - http://localhost:3000
```

### Adding New Origins

Tambahkan origin baru ke array `cors_origins`:

```yaml
cors_origins:
  - http://localhost:3000
  - https://new-domain.com # Origin baru
```
