package recordshare

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxRecipients = 25
	MaxNoteLength = 1000
	MaxPathLength = 1000
)

// Share is one record sent to one teammate: who sent it, with what note, and
// where in the app the link opens. Sharing it again with the same person
// counts and re-dates the same row. A share lets its recipient read the
// record's case and nothing more, until it is taken back: a revoked share is
// kept, with who took it back and when, and shows its recipient nothing.
type Share struct {
	keel.Table `keel:"record_shares"`

	ID             pulid.ID        `json:"id"             keel:"id,pk,autoid:rsh_"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	RecordKind     recordkind.Kind `json:"recordKind"     keel:"record_kind"`
	RecordID       pulid.ID        `json:"recordId"       keel:"record_id"`
	SharedWithID   pulid.ID        `json:"sharedWithId"   keel:"shared_with_id"`
	SharedByID     pulid.ID        `json:"sharedById"     keel:"shared_by_id"`
	Note           string          `json:"note"           keel:"note,nullzero"`
	Path           string          `json:"path"           keel:"path"`
	ShareCount     int             `json:"shareCount"     keel:"share_count"`
	FirstSharedAt  int64           `json:"firstSharedAt"  keel:"first_shared_at"`
	LastSharedAt   int64           `json:"lastSharedAt"   keel:"last_shared_at"`
	RevokedAt      *int64          `json:"revokedAt"      keel:"revoked_at"`
	RevokedByID    pulid.ID        `json:"revokedById"    keel:"revoked_by_id,nullzero"`
	CreatedAt      int64           `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64           `json:"updatedAt"      keel:"updated_at,autoupdate"`

	SharedWith *tenant.User `json:"sharedWith,omitempty" keel:"belongs:shared_with_id"`
	SharedBy   *tenant.User `json:"sharedBy,omitempty"   keel:"belongs:shared_by_id"`

	Removable bool `json:"removable" keel:"-"`
}

func (s *Share) Revoked() bool { return s.RevokedAt != nil }

func (s *Share) Revoke(by pulid.ID, now int64) {
	s.RevokedAt = &now
	s.RevokedByID = by
}

func (s *Share) GetID() pulid.ID { return s.ID }

func (s *Share) GetTableName() string { return "record_shares" }

// ValidPath reports whether a share's link is an address inside the app. The
// link is joined to the app's own origin in an email, so one that could leave
// it (another origin, a scheme) is never stored.
func ValidPath(path string) bool {
	return strings.HasPrefix(path, "/") &&
		!strings.HasPrefix(path, "//") &&
		!strings.ContainsAny(path, "\\\r\n\t ") &&
		utf8.RuneCountInString(path) <= MaxPathLength
}

func (s *Share) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.RecordID,
			validation.Required.Error("Record is required"),
		),
		validation.Field(&s.SharedWithID,
			validation.Required.Error("Recipient is required"),
		),
		validation.Field(&s.SharedByID,
			validation.Required.Error("Sharer is required"),
		),
	))

	if !s.RecordKind.Known() {
		multiErr.Add("kind", errortypes.ErrInvalid, "That kind of record cannot be shared")
	}
	if !ValidPath(s.Path) {
		multiErr.Add("path", errortypes.ErrInvalid, "The link must be an address inside the app")
	}
	if utf8.RuneCountInString(s.Note) > MaxNoteLength {
		multiErr.Add(
			"note",
			errortypes.ErrInvalid,
			"Note must be {0} characters or fewer",
			MaxNoteLength,
		)
	}
	if s.SharedWithID.IsNotNil() && s.SharedWithID == s.SharedByID {
		multiErr.Add("userIds", errortypes.ErrInvalid, "You cannot share something with yourself")
	}
}
