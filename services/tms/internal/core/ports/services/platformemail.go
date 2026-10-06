package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/shared/i18n"
)

var ErrPlatformEmailNotConfigured = errors.New(
	"platform email is not configured: set platform.cloud.systemEmail.apiKey",
)

type PlatformEmailMessage struct {
	Kind           string
	To             string
	Subject        string
	HTML           string
	Text           string
	IdempotencyKey string
}

type SignupVerificationEmail struct {
	To          string
	Locale      i18n.Locale
	Name        string
	CompanyName string
	Token       string
	ExpiresAt   int64
}

type SignupExistingAccountEmail struct {
	To     string
	Locale i18n.Locale
	Name   string
}

type WelcomeEmail struct {
	To          string
	Locale      i18n.Locale
	Name        string
	CompanyName string
	TrialEndsAt int64
	Timezone    string
}

type TrialEndedEmail struct {
	To            string
	Locale        i18n.Locale
	Name          string
	CompanyName   string
	ReadOnlyUntil int64
	Timezone      string
}

type AccountPurgedEmail struct {
	To          string
	Locale      i18n.Locale
	Name        string
	CompanyName string
}

type PasswordResetEmail struct {
	To               string
	Locale           i18n.Locale
	Name             string
	CompanyName      string
	ResetURL         string
	ExpiresInMinutes int
	ExpiresAt        int64
	Timezone         string
	IdempotencyKey   string
}

type PlatformEmailService interface {
	SendSignupVerification(ctx context.Context, msg *SignupVerificationEmail) error
	SendSignupExistingAccount(ctx context.Context, msg *SignupExistingAccountEmail) error
	SendWelcome(ctx context.Context, msg *WelcomeEmail) error
	SendTrialEnded(ctx context.Context, msg *TrialEndedEmail) error
	SendAccountPurged(ctx context.Context, msg *AccountPurgedEmail) error
	SendPasswordReset(ctx context.Context, msg *PasswordResetEmail) error
}
