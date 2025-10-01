package service

import (
	"context"
	"fmt"
	"log"
)

// EmailProvider interface for different email services
type EmailProvider interface {
	SendOTPEmail(ctx context.Context, data OTPEmailData) error
	SendWelcomeEmail(ctx context.Context, email, name string) error
}

// MultiProviderEmailService handles multiple email providers with fallback
type MultiProviderEmailService struct {
	providers []EmailProvider
	primary   EmailProvider
}

// NewMultiProviderEmailService creates a new multi-provider email service
func NewMultiProviderEmailService(providers []EmailProvider) *MultiProviderEmailService {
	if len(providers) == 0 {
		return &MultiProviderEmailService{}
	}

	return &MultiProviderEmailService{
		providers: providers,
		primary:   providers[0], // First provider is primary
	}
}

// SendOTPEmail tries to send OTP email using available providers
func (m *MultiProviderEmailService) SendOTPEmail(ctx context.Context, data OTPEmailData) error {
	log.Printf("MultiProviderEmailService: Starting OTP email send to %s", data.Email)
	log.Printf("MultiProviderEmailService: Available providers: %d", len(m.providers))

	if len(m.providers) == 0 {
		log.Printf("MultiProviderEmailService: No email providers configured")
		return fmt.Errorf("no email providers configured")
	}

	var lastErr error
	for i, provider := range m.providers {
		log.Printf("MultiProviderEmailService: Attempting to send OTP email via provider %d", i+1)

		err := provider.SendOTPEmail(ctx, data)
		if err == nil {
			log.Printf("MultiProviderEmailService: OTP email sent successfully via provider %d", i+1)
			return nil
		}

		log.Printf("MultiProviderEmailService: Provider %d failed: %v", i+1, err)
		lastErr = err
	}

	// All providers failed
	log.Printf("MultiProviderEmailService: All providers failed. Last error: %v", lastErr)
	return fmt.Errorf("all email providers failed. Last error: %w", lastErr)
}

// SendWelcomeEmail tries to send welcome email using available providers
func (m *MultiProviderEmailService) SendWelcomeEmail(ctx context.Context, email, name string) error {
	if len(m.providers) == 0 {
		return fmt.Errorf("no email providers configured")
	}

	var lastErr error
	for i, provider := range m.providers {
		log.Printf("Attempting to send welcome email via provider %d", i+1)

		err := provider.SendWelcomeEmail(ctx, email, name)
		if err == nil {
			log.Printf("Welcome email sent successfully via provider %d", i+1)
			return nil
		}

		log.Printf("Provider %d failed: %v", i+1, err)
		lastErr = err
	}

	// All providers failed
	return fmt.Errorf("all email providers failed. Last error: %w", lastErr)
}

// GetProviderCount returns the number of configured providers
func (m *MultiProviderEmailService) GetProviderCount() int {
	return len(m.providers)
}

// GetPrimaryProvider returns the primary provider
func (m *MultiProviderEmailService) GetPrimaryProvider() EmailProvider {
	return m.primary
}
