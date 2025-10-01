package service

import (
	"context"
	"fmt"
	"log"

	"github.com/resend/resend-go/v2"
)

type ResendService struct {
	client *resend.Client
	from   string
}

// ResendEmailData is an alias for OTPEmailData to maintain consistency
type ResendEmailData = OTPEmailData

func NewResendService(apiKey, fromEmail string) *ResendService {
	client := resend.NewClient(apiKey)

	return &ResendService{
		client: client,
		from:   fromEmail,
	}
}

func (rs *ResendService) SendOTPEmail(ctx context.Context, data ResendEmailData) error {
	log.Printf("ResendService: Attempting to send OTP email to %s", data.Email)
	subject := "Kode Verifikasi SALOME"

	// HTML template untuk OTP email
	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Kode Verifikasi SALOME</title>
		<style>
			body {
				font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
				line-height: 1.6;
				color: #333;
				max-width: 600px;
				margin: 0 auto;
				padding: 20px;
				background-color: #f8f9fa;
			}
			.container {
				background-color: white;
				border-radius: 10px;
				padding: 30px;
				box-shadow: 0 2px 10px rgba(0,0,0,0.1);
			}
			.header {
				text-align: center;
				margin-bottom: 30px;
			}
			.logo {
				font-size: 28px;
				font-weight: bold;
				color: #3b82f6;
				margin-bottom: 10px;
			}
			.title {
				font-size: 24px;
				color: #1f2937;
				margin-bottom: 20px;
			}
			.otp-code {
				background-color: #f3f4f6;
				border: 2px dashed #d1d5db;
				border-radius: 8px;
				padding: 20px;
				text-align: center;
				margin: 20px 0;
			}
			.otp-number {
				font-size: 32px;
				font-weight: bold;
				color: #3b82f6;
				letter-spacing: 5px;
				font-family: 'Courier New', monospace;
			}
			.warning {
				background-color: #fef3c7;
				border-left: 4px solid #f59e0b;
				padding: 15px;
				margin: 20px 0;
				border-radius: 4px;
			}
			.footer {
				text-align: center;
				margin-top: 30px;
				padding-top: 20px;
				border-top: 1px solid #e5e7eb;
				color: #6b7280;
				font-size: 14px;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<div class="logo">SALOME</div>
				<h1 class="title">Kode Verifikasi</h1>
			</div>
			
			<p>Halo <strong>%s</strong>,</p>
			
			<p>Terima kasih telah mendaftar di SALOME! Untuk menyelesaikan proses registrasi, silakan gunakan kode verifikasi berikut:</p>
			
			<div class="otp-code">
				<div class="otp-number">%s</div>
			</div>
			
			<div class="warning">
				<strong>⚠️ Penting:</strong>
				<ul>
					<li>Kode ini berlaku selama <strong>%d menit</strong></li>
					<li>Jangan bagikan kode ini kepada siapapun</li>
					<li>Jika Anda tidak meminta kode ini, abaikan email ini</li>
				</ul>
			</div>
			
			<p>Jika kode tidak berfungsi, silakan coba registrasi ulang atau hubungi tim support kami.</p>
			
			<div class="footer">
				<p>Email ini dikirim secara otomatis, mohon tidak membalas email ini.</p>
				<p>&copy; 2024 SALOME. All rights reserved.</p>
			</div>
		</div>
	</body>
	</html>
	`, data.Name, data.OTPCode, data.ExpiresIn)

	// Plain text version
	text := fmt.Sprintf(`
SALOME - Kode Verifikasi

Halo %s,

Terima kasih telah mendaftar di SALOME! 

Kode verifikasi Anda: %s

Kode ini berlaku selama %d menit.

Jangan bagikan kode ini kepada siapapun.

Jika Anda tidak meminta kode ini, abaikan email ini.

--
SALOME Team
	`, data.Name, data.OTPCode, data.ExpiresIn)

	// Send email
	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", "SALOME Platform", rs.from),
		To:      []string{data.Email},
		Subject: subject,
		Html:    html,
		Text:    text,
	}

	// Send email using SendWithContext
	log.Printf("ResendService: Sending OTP email to %s with params: From=%s, Subject=%s",
		data.Email, params.From, params.Subject)

	res, err := rs.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		log.Printf("ResendService: Error sending OTP email to %s: %v", data.Email, err)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	log.Printf("ResendService: OTP email sent successfully to %s. Message ID: %s", data.Email, res.Id)
	return nil
}

func (rs *ResendService) SendWelcomeEmail(ctx context.Context, email, name string) error {
	subject := "Selamat Datang di SALOME!"

	html := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Selamat Datang di SALOME</title>
		<style>
			body {
				font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
				line-height: 1.6;
				color: #333;
				max-width: 600px;
				margin: 0 auto;
				padding: 20px;
				background-color: #f8f9fa;
			}
			.container {
				background-color: white;
				border-radius: 10px;
				padding: 30px;
				box-shadow: 0 2px 10px rgba(0,0,0,0.1);
			}
			.header {
				text-align: center;
				margin-bottom: 30px;
			}
			.logo {
				font-size: 28px;
				font-weight: bold;
				color: #3b82f6;
				margin-bottom: 10px;
			}
			.title {
				font-size: 24px;
				color: #1f2937;
				margin-bottom: 20px;
			}
			.feature {
				background-color: #f8fafc;
				padding: 15px;
				margin: 10px 0;
				border-radius: 8px;
				border-left: 4px solid #3b82f6;
			}
			.footer {
				text-align: center;
				margin-top: 30px;
				padding-top: 20px;
				border-top: 1px solid #e5e7eb;
				color: #6b7280;
				font-size: 14px;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<div class="header">
				<div class="logo">SALOME</div>
				<h1 class="title">Selamat Datang!</h1>
			</div>
			
			<p>Halo <strong>%s</strong>,</p>
			
			<p>Selamat! Akun SALOME Anda telah berhasil dibuat dan diverifikasi. Sekarang Anda dapat menikmati semua fitur yang tersedia:</p>
			
			<div class="feature">
				<strong>🎯 Patungan Grup</strong><br>
				Bergabung dengan grup patungan untuk berbagi subscription aplikasi favorit Anda.
			</div>
			
			<div class="feature">
				<strong>💰 Hemat Biaya</strong><br>
				Bayar lebih sedikit dengan berbagi biaya subscription bersama teman-teman.
			</div>
			
			<div class="feature">
				<strong>🔒 Aman & Terpercaya</strong><br>
				Sistem pembayaran yang aman dan transparan untuk semua transaksi.
			</div>
			
			<p>Mulai jelajahi SALOME sekarang dan temukan grup patungan yang sesuai dengan kebutuhan Anda!</p>
			
			<div class="footer">
				<p>Terima kasih telah bergabung dengan SALOME!</p>
				<p>&copy; 2024 SALOME. All rights reserved.</p>
			</div>
		</div>
	</body>
	</html>
	`, name)

	text := fmt.Sprintf(`
SALOME - Selamat Datang!

Halo %s,

Selamat! Akun SALOME Anda telah berhasil dibuat dan diverifikasi.

Sekarang Anda dapat menikmati semua fitur yang tersedia:
- Patungan Grup: Bergabung dengan grup patungan untuk berbagi subscription
- Hemat Biaya: Bayar lebih sedikit dengan berbagi biaya subscription
- Aman & Terpercaya: Sistem pembayaran yang aman dan transparan

Mulai jelajahi SALOME sekarang dan temukan grup patungan yang sesuai dengan kebutuhan Anda!

Terima kasih telah bergabung dengan SALOME!

--
SALOME Team
	`, name)

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", "SALOME Platform", rs.from),
		To:      []string{email},
		Subject: subject,
		Html:    html,
		Text:    text,
	}

	// Send email using SendWithContext
	log.Printf("ResendService: Sending welcome email to %s with params: From=%s, Subject=%s",
		email, params.From, params.Subject)

	res, err := rs.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		log.Printf("ResendService: Error sending welcome email to %s: %v", email, err)
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	log.Printf("ResendService: Welcome email sent successfully to %s. Message ID: %s", email, res.Id)
	return nil
}
