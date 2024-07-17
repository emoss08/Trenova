// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package services

import (
	"fmt"
	"net/smtp"
	"strconv"

	"kafka/internal"

	"github.com/jordan-wright/email"
)

// EmailService provides functionalities to send emails using SMTP.
type EmailService struct {
	From     string // From address for the emails.
	Host     string // SMTP host.
	Port     int    // SMTP port.
	Username string // Username for SMTP authentication.
	Password string // Password for SMTP authentication.
}

// NewEmailService creates a new EmailService instance configured with environment variables.
//
// Returns:
//
//	*EmailService - a new EmailService instance
func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(internal.EnvVar("EMAIL_PORT"))
	return &EmailService{
		From:     internal.EnvVar("EMAIL_FROM"),
		Host:     internal.EnvVar("EMAIL_HOST"),
		Port:     port,
		Username: internal.EnvVar("EMAIL_USERNAME"),
		Password: internal.EnvVar("EMAIL_PASSWORD"),
	}
}

// Ping checks if the email service is available.
func (s *EmailService) Ping() error {
	smtpAddr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	c, err := smtp.Dial(smtpAddr)
	if err != nil {
		return err
	}

	defer c.Close()

	return c.Auth(auth)
}

// Send sends an email to a single recipient with the specified subject and body.
//
// Parameters:
//
//	to - recipient email address
//	subject - email subject
//	body - email body in HTML format
//
// Returns:
//
//	error - an error if sending the email fails
func (s *EmailService) Send(to, subject, body string) error {
	e := email.NewEmail()
	e.From = s.From
	e.To = []string{to}
	e.Subject = subject
	e.HTML = []byte(body)

	smtpAddr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	return e.Send(smtpAddr, auth)
}

// SendBulk sends an email to multiple recipients with the specified subject and body.
//
// Parameters:
//
//	to - slice of recipient email addresses
//	subject - email subject
//	body - email body in HTML format
//
// Returns:
//
//	error - an error if sending the email fails
func (s *EmailService) SendBulk(to []string, subject, body string) error {
	e := email.NewEmail()
	e.From = s.From
	e.To = to
	e.Subject = subject
	e.HTML = []byte(body)

	smtpAddr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	return e.Send(smtpAddr, auth)
}
