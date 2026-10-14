package factoring

import (
	"context"
	"path"
	"strings"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
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

var _ bun.BeforeAppendModelHook = (*Transport)(nil)

type Transport struct {
	bun.BaseModel `bun:"table:factoring_transports,alias:factrn" json:"-"`

	ID                 pulid.ID        `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID        `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID        `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	FactoringCompanyID pulid.ID        `json:"factoringCompanyId" bun:"factoring_company_id,type:VARCHAR(100),notnull"`
	Method             TransportMethod `json:"method"             bun:"method,type:VARCHAR(20),notnull,default:'Email'"`
	EmailRecipients    []string        `json:"emailRecipients"    bun:"email_recipients,array,type:text[],nullzero"`
	AttachPackets      bool            `json:"attachPackets"      bun:"attach_packets,type:BOOLEAN,notnull,default:true"`
	SftpHost           string          `json:"sftpHost"           bun:"sftp_host,type:VARCHAR(255),nullzero"`
	SftpPort           int             `json:"sftpPort"           bun:"sftp_port,type:INTEGER,nullzero"`
	SftpUsername       string          `json:"sftpUsername"       bun:"sftp_username,type:VARCHAR(255),nullzero"`
	SftpAuthMethod     SftpAuthMethod  `json:"sftpAuthMethod"     bun:"sftp_auth_method,type:VARCHAR(20),nullzero"`
	SftpSecret         string          `json:"-"                  bun:"sftp_secret,type:TEXT,nullzero"`
	SftpKnownHostKey   string          `json:"sftpKnownHostKey"   bun:"sftp_known_host_key,type:TEXT,nullzero"`
	SftpDirectory      string          `json:"sftpDirectory"      bun:"sftp_directory,type:VARCHAR(500),nullzero"`
	LastTestedAt       *int64          `json:"lastTestedAt"       bun:"last_tested_at,type:BIGINT,nullzero"`
	LastTestError      string          `json:"lastTestError"      bun:"last_test_error,type:TEXT,nullzero"`
	Version            int64           `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64           `json:"createdAt"          bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt          int64           `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull"`
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

func (t *Transport) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("factrn_")
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}
	return nil
}

func (t *Transport) GetID() pulid.ID             { return t.ID }
func (t *Transport) GetOrganizationID() pulid.ID { return t.OrganizationID }
func (t *Transport) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }
func (t *Transport) GetTableName() string        { return "factoring_transports" }
