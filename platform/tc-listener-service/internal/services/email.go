// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
