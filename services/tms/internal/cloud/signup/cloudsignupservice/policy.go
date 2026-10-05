package cloudsignupservice

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/cloud/signup/signupport"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/passwordutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxNameLength        = 100
	maxCompanyNameLength = 100
	minPasswordLength    = 12
	maxPasswordBytes     = 72
	loginSlugBaseLength  = 60
)

type signupInput struct {
	name        string
	companyName string
	password    string
	email       emailutils.Address
}

func (s *Service) validateSignup(req *signupport.CloudSignupRequest) (*signupInput, error) {
	multiErr := errortypes.NewMultiError()

	name := strings.Join(strings.Fields(req.Name), " ")
	switch {
	case name == "":
		multiErr.Add("name", errortypes.ErrRequired, "Enter your name")
	case utf8.RuneCountInString(name) > maxNameLength:
		multiErr.Add("name", errortypes.ErrInvalidLength, "Name must be 100 characters or fewer")
	}

	companyName := strings.Join(strings.Fields(req.CompanyName), " ")
	switch {
	case companyName == "":
		multiErr.Add("companyName", errortypes.ErrRequired, "Enter your company name")
	case utf8.RuneCountInString(companyName) > maxCompanyNameLength:
		multiErr.Add(
			"companyName",
			errortypes.ErrInvalidLength,
			"Company name must be 100 characters or fewer",
		)
	}

	address, emailErr := s.checkEmail(req.EmailAddress)
	if emailErr != nil {
		multiErr.AddError(emailErr)
	}

	if !req.AcceptTerms {
		multiErr.Add(
			"acceptTerms",
			errortypes.ErrRequired,
			"Accept the terms of service and privacy policy to continue",
		)
	}

	if passwordErr := checkPassword(req.Password, address); passwordErr != nil {
		multiErr.AddError(passwordErr)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &signupInput{
		name:        name,
		companyName: companyName,
		password:    req.Password,
		email:       address,
	}, nil
}

func (s *Service) checkEmail(raw string) (emailutils.Address, *errortypes.Error) {
	address, err := emailutils.Parse(raw)
	if err != nil {
		if errors.Is(err, emailutils.ErrEmpty) {
			return emailutils.Address{}, errortypes.NewValidationError(
				"emailAddress",
				errortypes.ErrRequired,
				"Enter your work email",
			)
		}

		return emailutils.Address{}, errortypes.NewValidationError(
			"emailAddress",
			errortypes.ErrInvalidFormat,
			"Enter a valid email address",
		)
	}

	signup := s.cloud.Signup
	if signup.BlockDisposableEmail && emailutils.IsDisposableDomain(address.Domain) {
		return emailutils.Address{}, errortypes.NewValidationError(
			"emailAddress",
			errortypes.ErrInvalid,
			"Use a permanent work email address. Disposable addresses cannot sign up",
		)
	}

	if !emailutils.DomainAllowed(address.Domain, signup.GetAllowedEmailDomains()) {
		return emailutils.Address{}, errortypes.NewValidationError(
			"emailAddress",
			errortypes.ErrInvalid,
			"Signups are limited to approved email domains",
		)
	}

	return address, nil
}

func checkPassword(password string, address emailutils.Address) *errortypes.Error {
	switch {
	case utf8.RuneCountInString(password) < minPasswordLength:
		return errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalidLength,
			"Password must be at least 12 characters",
		)
	case len(password) > maxPasswordBytes:
		return errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalidLength,
			"Password must be 72 bytes or fewer",
		)
	case strings.TrimSpace(password) == "":
		return errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalid,
			"Password cannot be only spaces",
		)
	}

	if address.Lower != "" {
		lowered := strings.ToLower(strings.TrimSpace(password))
		if lowered == address.Lower ||
			passwordutils.ContainsIgnoringCase(password, address.LocalPart) {
			return errortypes.NewValidationError(
				"password",
				errortypes.ErrInvalid,
				"Password must not contain your email address",
			)
		}
	}

	if passwordutils.IsCommon(password) {
		return errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalid,
			"This password is too common. Choose something harder to guess",
		)
	}

	return nil
}

func loginSlugBase(companyName string) string {
	return stringutils.SlugifyASCII(companyName, loginSlugBaseLength)
}

func usernameBase(address emailutils.Address) string {
	var builder strings.Builder
	builder.Grow(len(address.LocalPart))
	for _, r := range address.LocalPart {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			builder.WriteRune(r)
		}
		if builder.Len() >= maxUsernameLength {
			break
		}
	}

	return strings.Trim(builder.String(), "._-")
}
