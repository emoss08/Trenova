/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package provider

import (
	"context"

	"github.com/emoss08/trenova/microservices/email/internal/model"
	"github.com/rotisserie/eris"
)

// EmailProvider is the interface that all email providers must implement
type EmailProvider interface {
	// Send sends an email
	Send(ctx context.Context, email *model.Email) error

	// GetName returns the name of the provider
	GetName() string
}

// ProviderType is the type of email provider
type Type string

const (
	// ProviderTypeSMTP is the SMTP provider
	TypeSMTP Type = "smtp"
	// ProviderTypeSendGrid is the SendGrid provider
	TypeSendGrid Type = "sendgrid"
)

// Factory creates email providers
type Factory struct {
	smtpProvider     EmailProvider
	sendGridProvider EmailProvider
	defaultProvider  EmailProvider
}

// NewFactory creates a new provider factory
func NewFactory(smtp *SMTPProvider, sendGrid *SendGridProvider) *Factory {
	// Default to SMTP if configured, otherwise SendGrid
	var defaultProvider EmailProvider
	switch {
	case smtp.IsConfigured():
		defaultProvider = smtp
	case sendGrid.IsConfigured():
		defaultProvider = sendGrid
	default:
		defaultProvider = smtp
	}

	return &Factory{
		smtpProvider:     smtp,
		sendGridProvider: sendGrid,
		defaultProvider:  defaultProvider,
	}
}

// GetProvider returns the provider for the given type
func (f *Factory) GetProvider(providerType Type) (EmailProvider, error) {
	switch providerType {
	case TypeSMTP:
		return f.smtpProvider, nil
	case TypeSendGrid:
		return f.sendGridProvider, nil
	default:
		return f.defaultProvider, nil
	}
}

// GetDefaultProvider returns the default provider
func (f *Factory) GetDefaultProvider() EmailProvider {
	return f.defaultProvider
}

// ValidateConfig validates that at least one provider is configured
func (f *Factory) ValidateConfig() error {
	smtpProvider, ok := f.smtpProvider.(*SMTPProvider)
	if ok && smtpProvider.IsConfigured() {
		return nil
	}

	sendGridProvider, ok := f.sendGridProvider.(*SendGridProvider)
	if ok && sendGridProvider.IsConfigured() {
		return nil
	}

	return eris.New("no email provider is configured")
}
