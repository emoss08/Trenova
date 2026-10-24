package factoring

import (
	"context"
	"path"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	MaxTransportRecipients   = 10
	MaxSftpHostLength        = 255
	MaxSftpUsernameLength    = 255
	MaxSftpDirectoryLength   = 500
	MaxSftpKnownHostKeyBytes = 8192
	DefaultSftpPort          = 22
	DefaultSftpDirectory     = "/"
)

type Transport struct {
	keel.Table `keel:"factoring_transports"`

	ID                 pulid.ID        `json:"id" keel:"id,pk,autoid:factrn_"`
	OrganizationID     pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	FactoringCompanyID pulid.ID        `json:"factoringCompanyId" keel:"factoring_company_id"`
	Method             TransportMethod `json:"method" keel:"method"`
	EmailRecipients    []string        `json:"emailRecipients" keel:"email_recipients,nullzero"`
	AttachPackets      bool            `json:"attachPackets" keel:"attach_packets"`
	SftpHost           string          `json:"sftpHost" keel:"sftp_host,nullzero"`
	SftpPort           int             `json:"sftpPort" keel:"sftp_port,nullzero"`
	SftpUsername       string          `json:"sftpUsername" keel:"sftp_username,nullzero"`
	SftpAuthMethod     SftpAuthMethod  `json:"sftpAuthMethod" keel:"sftp_auth_method,nullzero"`
	SftpSecret         string          `json:"-" keel:"sftp_secret,nullzero"`
	SftpKnownHostKey   string          `json:"sftpKnownHostKey" keel:"sftp_known_host_key,nullzero"`
	SftpDirectory      string          `json:"sftpDirectory" keel:"sftp_directory,nullzero"`
	LastTestedAt       *int64          `json:"lastTestedAt" keel:"last_tested_at"`
	LastTestError      string          `json:"lastTestError" keel:"last_test_error,nullzero"`
	Version            int64           `json:"version" keel:"version,version"`
	CreatedAt          int64           `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64           `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (t *Transport) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		t,
		validation.Field(&t.OrganizationID, validation.Required),
		validation.Field(&t.BusinessUnitID, validation.Required),
		validation.Field(
			&t.FactoringCompanyID,
			validation.Required.Error("Factoring company is required"),
		),
		validation.Field(
			&t.Method,
			validation.Required.Error("Choose how schedules reach the factor"),
			domainvalidation.ValidEnum[TransportMethod]("Invalid transport"),
		),
	))

	switch t.Method {
	case TransportMethodEmail:
		t.validateEmail(multiErr)
	case TransportMethodSftp:
		t.validateSftp(multiErr)
	}
}

func (t *Transport) validateEmail(multiErr *errortypes.MultiError) {
	if len(t.EmailRecipients) == 0 {
		multiErr.Add(
			"emailRecipients",
			errortypes.ErrRequired,
			"Add the factor's schedule inbox",
		)
		return
	}
	if len(t.EmailRecipients) > MaxTransportRecipients {
		multiErr.Add(
			"emailRecipients",
			errortypes.ErrInvalid,
			"A schedule can go to at most 10 addresses",
		)
	}
	for _, recipient := range t.EmailRecipients {
		if err := is.EmailFormat.Validate(recipient); err != nil || recipient == "" {
			multiErr.Add(
				"emailRecipients",
				errortypes.ErrInvalid,
				"Enter valid email addresses",
			)
			return
		}
	}
}

func (t *Transport) validateSftp(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		t,
		validation.Field(
			&t.SftpHost,
			validation.Required.Error("SFTP host is required"),
			validation.Length(0, MaxSftpHostLength),
			is.Host.Error("Enter a host name or IP address, without a scheme or path"),
		),
		validation.Field(
			&t.SftpPort,
			validation.Required.Error("SFTP port is required"),
			validation.Min(1).Error("Port must be between 1 and 65535"),
			validation.Max(65535).Error("Port must be between 1 and 65535"),
		),
		validation.Field(
			&t.SftpUsername,
			validation.Required.Error("SFTP username is required"),
			validation.Length(0, MaxSftpUsernameLength),
		),
		validation.Field(
			&t.SftpAuthMethod,
			validation.Required.Error("Choose password or private key"),
			domainvalidation.ValidEnum[SftpAuthMethod]("Invalid SFTP sign-in method"),
		),
		validation.Field(
			&t.SftpKnownHostKey,
			validation.Required.Error(
				"Paste the server's host key so Trenova only ever talks to that server",
			),
			validation.Length(0, MaxSftpKnownHostKeyBytes),
		),
		validation.Field(
			&t.SftpDirectory,
			validation.Length(0, MaxSftpDirectoryLength),
		),
	))
	if strings.TrimSpace(t.SftpSecret) == "" {
		multiErr.Add(
			"sftpSecret",
			errortypes.ErrRequired,
			"Enter the SFTP password or private key",
		)
	}
	if t.SftpDirectory != "" && !path.IsAbs(t.SftpDirectory) {
		multiErr.Add(
			"sftpDirectory",
			errortypes.ErrInvalid,
			"Directory must start with /",
		)
	}
	if strings.Contains(t.SftpDirectory, "..") {
		multiErr.Add(
			"sftpDirectory",
			errortypes.ErrInvalid,
			"Directory cannot contain ..",
		)
	}
}

func (t *Transport) Normalize() {
	recipients := make([]string, 0, len(t.EmailRecipients))
	seen := make(map[string]struct{}, len(t.EmailRecipients))
	for _, recipient := range t.EmailRecipients {
		trimmed := strings.TrimSpace(recipient)
		key := strings.ToLower(trimmed)
		if trimmed == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		recipients = append(recipients, trimmed)
	}
	t.EmailRecipients = recipients
	t.SftpHost = strings.TrimSpace(t.SftpHost)
	t.SftpUsername = strings.TrimSpace(t.SftpUsername)
	t.SftpKnownHostKey = strings.TrimSpace(t.SftpKnownHostKey)
	t.SftpDirectory = strings.TrimSpace(t.SftpDirectory)
	if t.Method != TransportMethodSftp {
		t.SftpHost = ""
		t.SftpPort = 0
		t.SftpUsername = ""
		t.SftpAuthMethod = ""
		t.SftpSecret = ""
		t.SftpKnownHostKey = ""
		t.SftpDirectory = ""
		return
	}
	t.EmailRecipients = nil
	if t.SftpPort == 0 {
		t.SftpPort = DefaultSftpPort
	}
	if t.SftpDirectory == "" {
		t.SftpDirectory = DefaultSftpDirectory
	}
	t.SftpDirectory = path.Clean(t.SftpDirectory)
}

func (t *Transport) SecretSet() bool {
	return t != nil && t.SftpSecret != ""
}

func (t *Transport) Target() string {
	if t == nil {
		return ""
	}
	if t.Method == TransportMethodSftp {
		return t.SftpUsername + "@" + t.SftpHost + ":" + t.SftpDirectory
	}
	return strings.Join(t.EmailRecipients, ", ")
}

func (t *Transport) RecordTest(at int64, err error) {
	t.LastTestedAt = &at
	t.LastTestError = ""
	if err != nil {
		t.LastTestError = err.Error()
	}
}

func (t *Transport) BeforeCreate(ctx context.Context) error {
	if t.Method == "" {
		t.Method = TransportMethodEmail
	}

	return nil
}

func (t *Transport) GetID() pulid.ID             { return t.ID }
func (t *Transport) GetOrganizationID() pulid.ID { return t.OrganizationID }
func (t *Transport) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }
func (t *Transport) GetTableName() string        { return "factoring_transports" }
