package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mailersend/mailersend-go"
)

type EmailService struct {
	client *mailersend.Mailersend
	from   mailersend.From
}

type OTPEmailData struct {
	Email     string
	Name      string
	OTPCode   string
	ExpiresIn int // in minutes
}

func NewEmailService(apiKey, fromEmail, fromName string) *EmailService {
	client := mailersend.NewMailersend(apiKey)

	from := mailersend.From{
		Name:  fromName,
		Email: fromEmail,
	}

	return &EmailService{
		client: client,
		from:   from,
	}
}

func (es *EmailService) SendOTPEmail(ctx context.Context, data OTPEmailData) error {
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

	recipients := []mailersend.Recipient{
		{
			Name:  data.Name,
			Email: data.Email,
		},
	}

	message := es.client.Email.NewMessage()
	message.SetFrom(es.from)
	message.SetRecipients(recipients)
	message.SetSubject(subject)
	message.SetHTML(html)
	message.SetText(text)

	// Set timeout context
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Send email
	res, err := es.client.Email.Send(ctx, message)
	if err != nil {
		log.Printf("Error sending OTP email to %s: %v", data.Email, err)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	log.Printf("OTP email sent successfully to %s. Message ID: %s", data.Email, res.Header.Get("X-Message-Id"))
	return nil
}

func (es *EmailService) SendWelcomeEmail(ctx context.Context, email, name string) error {
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

	recipients := []mailersend.Recipient{
		{
			Name:  name,
			Email: email,
		},
	}

	message := es.client.Email.NewMessage()
	message.SetFrom(es.from)
	message.SetRecipients(recipients)
	message.SetSubject(subject)
	message.SetHTML(html)
	message.SetText(text)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	res, err := es.client.Email.Send(ctx, message)
	if err != nil {
		log.Printf("Error sending welcome email to %s: %v", email, err)
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	log.Printf("Welcome email sent successfully to %s. Message ID: %s", email, res.Header.Get("X-Message-Id"))
	return nil
}
