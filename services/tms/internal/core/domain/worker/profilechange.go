package worker

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var ErrInvalidProfileChangeStatus = errors.New("invalid profile change status")

// ProfileChangeStatus is how far a driver's request to change their own
// record has got.
type ProfileChangeStatus string

const (
	ProfileChangePending   = ProfileChangeStatus("Pending")
	ProfileChangeApproved  = ProfileChangeStatus("Approved")
	ProfileChangeRejected  = ProfileChangeStatus("Rejected")
	ProfileChangeWithdrawn = ProfileChangeStatus("Withdrawn")
)

func (s ProfileChangeStatus) String() string { return string(s) }

func (s ProfileChangeStatus) IsValid() bool {
	switch s {
	case ProfileChangePending, ProfileChangeApproved, ProfileChangeRejected, ProfileChangeWithdrawn:
		return true
	default:
		return false
	}
}

// IsOpen reports whether the request is still waiting on somebody.
func (s ProfileChangeStatus) IsOpen() bool { return s == ProfileChangePending }

// CanTransitionTo is the state machine. Every decision is final: a change
// somebody approved has already been written onto the record, and one they
// rejected is asked for again rather than reopened.
func (s ProfileChangeStatus) CanTransitionTo(next ProfileChangeStatus) bool {
	if s != ProfileChangePending {
		return false
	}
	return next == ProfileChangeApproved ||
		next == ProfileChangeRejected ||
		next == ProfileChangeWithdrawn
}

// ProfileField is a field a worker may ask to change about themselves. The
// list is closed: compliance fields — licence, medical card, endorsements —
// stay carrier-controlled, because a driver who could edit their own
// qualifications could also edit them into compliance.
type ProfileField string

const (
	ProfileFieldPhoneNumber           = ProfileField("phoneNumber")
	ProfileFieldAddressLine1          = ProfileField("addressLine1")
	ProfileFieldAddressLine2          = ProfileField("addressLine2")
	ProfileFieldCity                  = ProfileField("city")
	ProfileFieldPostalCode            = ProfileField("postalCode")
	ProfileFieldEmergencyContactName  = ProfileField("emergencyContactName")
	ProfileFieldEmergencyContactPhone = ProfileField("emergencyContactPhone")
)

// profileFieldOrder is the order changes are listed in, which is the order a
// person reads an address.
var profileFieldOrder = []ProfileField{
	ProfileFieldPhoneNumber,
	ProfileFieldAddressLine1,
	ProfileFieldAddressLine2,
	ProfileFieldCity,
	ProfileFieldPostalCode,
	ProfileFieldEmergencyContactName,
	ProfileFieldEmergencyContactPhone,
}

func (f ProfileField) IsValid() bool {
	for _, known := range profileFieldOrder {
		if known == f {
			return true
		}
	}
	return false
}

// Label is how the field reads to a person deciding the request.
func (f ProfileField) Label() string {
	switch f {
	case ProfileFieldPhoneNumber:
		return "Phone"
	case ProfileFieldAddressLine1:
		return "Address line 1"
	case ProfileFieldAddressLine2:
		return "Address line 2"
	case ProfileFieldCity:
		return "City"
	case ProfileFieldPostalCode:
		return "Postal code"
	case ProfileFieldEmergencyContactName:
		return "Emergency contact"
	case ProfileFieldEmergencyContactPhone:
		return "Emergency contact phone"
	default:
		return string(f)
	}
}

// ContactSnapshot is the part of a worker's record they may ask to change.
type ContactSnapshot struct {
	PhoneNumber           string
	AddressLine1          string
	AddressLine2          string
	City                  string
	PostalCode            string
	EmergencyContactName  string
	EmergencyContactPhone string
}

// ContactSnapshotOf reads the changeable fields off a worker.
func ContactSnapshotOf(w *Worker) ContactSnapshot {
	return ContactSnapshot{
		PhoneNumber:           w.PhoneNumber,
		AddressLine1:          w.AddressLine1,
		AddressLine2:          w.AddressLine2,
		City:                  w.City,
		PostalCode:            w.PostalCode,
		EmergencyContactName:  w.EmergencyContactName,
		EmergencyContactPhone: w.EmergencyContactPhone,
	}
}

func (c ContactSnapshot) value(field ProfileField) string {
	switch field {
	case ProfileFieldPhoneNumber:
		return c.PhoneNumber
	case ProfileFieldAddressLine1:
		return c.AddressLine1
	case ProfileFieldAddressLine2:
		return c.AddressLine2
	case ProfileFieldCity:
		return c.City
	case ProfileFieldPostalCode:
		return c.PostalCode
	case ProfileFieldEmergencyContactName:
		return c.EmergencyContactName
	case ProfileFieldEmergencyContactPhone:
		return c.EmergencyContactPhone
	default:
		return ""
	}
}

// FieldChange is one field's before and after.
type FieldChange struct {
	Field ProfileField `json:"field"`
	From  string       `json:"from"`
	To    string       `json:"to"`
}

// DiffContact is what a request would change. Fields that would not move are
// left out, so the person deciding sees only what is actually being asked.
func DiffContact(current, wanted ContactSnapshot) []FieldChange {
	changes := make([]FieldChange, 0, len(profileFieldOrder))
	for _, field := range profileFieldOrder {
		from := strings.TrimSpace(current.value(field))
		to := strings.TrimSpace(wanted.value(field))
		if from == to {
			continue
		}
		changes = append(changes, FieldChange{Field: field, From: from, To: to})
	}
	return changes
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerProfileChangeRequest)(nil)
	_ validationframework.TenantedEntity = (*WorkerProfileChangeRequest)(nil)
)

// WorkerProfileChangeRequest is a driver asking to change their own record,
// waiting on somebody in the office. The changes are stored as before-and-
// after pairs rather than as a copy of the wanted record, so a manager sees
// exactly what is being asked and an approval applies exactly that — not
// whatever else happened to be on the form.
type WorkerProfileChangeRequest struct {
	bun.BaseModel `bun:"table:worker_profile_change_requests,alias:wpcr" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	Status       ProfileChangeStatus `json:"status"       bun:"status,type:profile_change_status_enum,notnull,default:'Pending'"`
	Changes      []FieldChange       `json:"changes"      bun:"changes,type:JSONB,notnull"`
	Note         string              `json:"note"         bun:"note,type:VARCHAR(500),nullzero"`
	SubmittedAt  int64               `json:"submittedAt"  bun:"submitted_at,type:BIGINT,notnull"`
	DecidedAt    *int64              `json:"decidedAt"    bun:"decided_at,type:BIGINT,nullzero"`
	DecidedByID  pulid.ID            `json:"decidedById"  bun:"decided_by_id,type:VARCHAR(100),nullzero"`
	DecisionNote string              `json:"decisionNote" bun:"decision_note,type:VARCHAR(500),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker *Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (r *WorkerProfileChangeRequest) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ProfileChangeStatus]("Status is not valid"),
		),
		validation.Field(&r.SubmittedAt, validation.Required.Error("A submission date is required")),
		validation.Field(&r.Note,
			validation.Length(0, 500).Error("Note cannot exceed 500 characters"),
		),
		validation.Field(&r.DecisionNote,
			validation.Length(0, 500).Error("Note cannot exceed 500 characters"),
		),
	))

	// A request that changes nothing is a request nobody can decide.
	if len(r.Changes) == 0 {
		multiErr.Add("changes", errortypes.ErrRequired, "Nothing would change")
	}
	for i, change := range r.Changes {
		if !change.Field.IsValid() {
			multiErr.Add(
				"changes",
				errortypes.ErrInvalid,
				"That is not something you can change from the app",
			)
			break
		}
		if i > 0 && r.Changes[i-1].Field == change.Field {
			multiErr.Add("changes", errortypes.ErrInvalid, "A field is listed twice")
			break
		}
	}
	if (r.Status == ProfileChangeApproved || r.Status == ProfileChangeRejected) &&
		(r.DecidedAt == nil || *r.DecidedAt <= 0) {
		multiErr.Add("decidedAt", errortypes.ErrRequired, "A decision needs the date it was made")
	}
	// A rejection with no reason leaves a driver with a record that did not
	// change and no idea why.
	if r.Status == ProfileChangeRejected && strings.TrimSpace(r.DecisionNote) == "" {
		multiErr.Add("decisionNote", errortypes.ErrRequired, "Turning a request down needs a reason")
	}
}

// ApplyTo writes the approved changes onto the worker. Only the closed list of
// fields is ever touched, whatever the stored changes say.
func (r *WorkerProfileChangeRequest) ApplyTo(w *Worker) {
	for _, change := range r.Changes {
		switch change.Field {
		case ProfileFieldPhoneNumber:
			w.PhoneNumber = change.To
		case ProfileFieldAddressLine1:
			w.AddressLine1 = change.To
		case ProfileFieldAddressLine2:
			w.AddressLine2 = change.To
		case ProfileFieldCity:
			w.City = change.To
		case ProfileFieldPostalCode:
			w.PostalCode = change.To
		case ProfileFieldEmergencyContactName:
			w.EmergencyContactName = change.To
		case ProfileFieldEmergencyContactPhone:
			w.EmergencyContactPhone = change.To
		}
	}
}

// Fields is the fields a request touches, in reading order.
func (r *WorkerProfileChangeRequest) Fields() []ProfileField {
	out := make([]ProfileField, 0, len(r.Changes))
	for _, change := range r.Changes {
		out = append(out, change.Field)
	}
	sort.Slice(out, func(i, j int) bool {
		return fieldRank(out[i]) < fieldRank(out[j])
	})
	return out
}

func fieldRank(field ProfileField) int {
	for i, known := range profileFieldOrder {
		if known == field {
			return i
		}
	}
	return len(profileFieldOrder)
}

func (r *WorkerProfileChangeRequest) GetID() pulid.ID { return r.ID }

func (r *WorkerProfileChangeRequest) GetCreatedAt() int64 { return r.CreatedAt }

func (r *WorkerProfileChangeRequest) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *WorkerProfileChangeRequest) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *WorkerProfileChangeRequest) GetTableName() string {
	return "worker_profile_change_requests"
}

func (r *WorkerProfileChangeRequest) GetResourceType() string { return "profile_change_request" }

func (r *WorkerProfileChangeRequest) GetResourceID() string { return r.ID.String() }

func (r *WorkerProfileChangeRequest) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("wpcr_")
		}
		if r.Status == "" {
			r.Status = ProfileChangePending
		}
		if r.SubmittedAt <= 0 {
			r.SubmittedAt = now
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}
