package services

import (
	"context"
	"errors"
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
	Name        string
	CompanyName string
	Token       string
	ExpiresAt   int64
}

type SignupExistingAccountEmail struct {
	To   string
	Name string
}

type WelcomeEmail struct {
	To          string
	Name        string
	CompanyName string
	TrialEndsAt int64
	Timezone    string
}

type TrialEndedEmail struct {
	To            string
	Name          string
	CompanyName   string
	ReadOnlyUntil int64
	Timezone      string
}

type AccountPurgedEmail struct {
	To          string
	Name        string
	CompanyName string
}

type PlatformEmailService interface {
	SendSignupVerification(ctx context.Context, msg *SignupVerificationEmail) error
	SendSignupExistingAccount(ctx context.Context, msg *SignupExistingAccountEmail) error
	SendWelcome(ctx context.Context, msg *WelcomeEmail) error
	SendTrialEnded(ctx context.Context, msg *TrialEndedEmail) error
	SendAccountPurged(ctx context.Context, msg *AccountPurgedEmail) error
	SendRendered(ctx context.Context, msg *PlatformEmailMessage) error
}
