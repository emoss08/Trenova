package systememail

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	CurrentID   = "current"
	InstanceTag = "trenova_instance"

	fingerprintVersion = "trenova.system-email.v1"
	minAPIKeyLength    = 20
	maxSecretLength    = 512
	maxFromAddress     = 254
	maxFromName        = 40
	maxDisplayName     = 100
	displayJoiner      = " via "
)

var ErrNoSender = errors.New("no system email sender is set on this instance")

type Sender struct {
	keel.Table `keel:"system_email_senders"`

	ID           string         `json:"-"           keel:"id,pk"`
	Provider     email.Provider `json:"provider"    keel:"provider"`
	FromAddress  string         `json:"fromAddress" keel:"from_address"`
	FromName     string         `json:"fromName"    keel:"from_name"`
	APIKeySealed string         `json:"-"           keel:"api_key_sealed"`
	Fingerprint  string         `json:"fingerprint" keel:"fingerprint"`
	ReceivedAt   int64          `json:"receivedAt"  keel:"received_at"`
}

type Credential struct {
	Provider    email.Provider
	APIKey      string
	FromAddress string
	FromName    string
}

func (c *Credential) String() string {
	if c == nil {
		return "<no system email credential>"
	}
	return "system email credential for " + string(c.Provider) + " from " + c.FromAddress
}

func (c *Credential) GoString() string {
	return c.String()
}

func (c *Credential) Validate() error {
	multiErr := errortypes.NewMultiError()
	if c == nil {
		multiErr.Add("", errortypes.ErrRequired, "A system email credential is required")
		return multiErr.Err()
	}
	if c.Provider != email.ProviderResend && c.Provider != email.ProviderPostmark {
		multiErr.Add("provider", errortypes.ErrInvalid, "Provider must be Resend or Postmark")
	}
	switch {
	case c.APIKey == "":
		multiErr.Add("apiKey", errortypes.ErrRequired, "API key is required")
	case !opaque(c.APIKey, minAPIKeyLength, maxSecretLength):
		multiErr.Add(
			"apiKey",
			errortypes.ErrInvalid,
			"API key must be 20 to 512 printable characters with no spaces",
		)
	}
	if !validAddress(c.FromAddress) {
		multiErr.Add(
			"fromAddress",
			errortypes.ErrInvalid,
			"From address must be a plain email address of at most 254 characters",
		)
	}
	if !validName(c.FromName) {
		multiErr.Add(
			"fromName",
			errortypes.ErrInvalid,
			"From name must be 1 to 40 printable characters with no leading or trailing space",
		)
	}
	return multiErr.Err()
}

func Fingerprint(c *Credential) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		fingerprintVersion,
		string(c.Provider),
		c.FromAddress,
		c.FromName,
		c.APIKey,
	}, "\n")))
	return hex.EncodeToString(sum[:])
}

func DisplayName(organizationName, fromName string) string {
	org := strings.TrimSpace(organizationName)
	if org == "" {
		return fromName
	}
	room := maxDisplayName - utf8.RuneCountInString(displayJoiner) -
		utf8.RuneCountInString(fromName)
	if runes := []rune(org); len(runes) > room {
		org = strings.TrimSpace(string(runes[:max(room, 0)]))
	}
	if org == "" {
		return fromName
	}
	return org + displayJoiner + fromName
}

type Status struct {
	Configured  bool
	Provider    email.Provider
	FromAddress string
	Fingerprint string
}

func opaque(v string, minLen, maxLen int) bool {
	if len(v) < minLen || len(v) > maxLen || !utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if r > unicode.MaxASCII || !unicode.IsPrint(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func validAddress(v string) bool {
	if v == "" || len(v) > maxFromAddress {
		return false
	}
	parsed, err := mail.ParseAddress(v)
	return err == nil && parsed.Address == v && parsed.Name == ""
}

func validName(v string) bool {
	if v == "" || utf8.RuneCountInString(v) > maxFromName || strings.TrimSpace(v) != v ||
		!utf8.ValidString(v) {
		return false
	}
	for _, r := range v {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
