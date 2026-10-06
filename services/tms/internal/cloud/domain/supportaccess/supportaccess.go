package supportaccess

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxNoteLength      = 500
	MaxReasonLength    = 500
	MaxTicketLength    = 100
	MinReasonLength    = 10
	MaxUserAgentLength = 512
	MaxClientIPLength  = 64
	MaxAddedByLength   = 255
)

var (
	_ bun.BeforeAppendModelHook = (*Grant)(nil)
	_ bun.BeforeAppendModelHook = (*Session)(nil)
	_ bun.BeforeAppendModelHook = (*StaffMember)(nil)
	_ bun.BeforeAppendModelHook = (*Principal)(nil)
)

type Grant struct {
	bun.BaseModel `bun:"table:support_access_grants,alias:sag" json:"-"`

	ID             pulid.ID   `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID   `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID   `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),notnull"`
	GrantedByID    pulid.ID   `json:"grantedById"           bun:"granted_by_id,type:VARCHAR(100),notnull"`
	AccessMode     AccessMode `json:"accessMode"            bun:"access_mode,type:VARCHAR(20),notnull"`
	Note           string     `json:"note,omitempty"        bun:"note,type:VARCHAR(500),nullzero"`
	StartsAt       int64      `json:"startsAt"              bun:"starts_at,type:BIGINT,notnull"`
	ExpiresAt      int64      `json:"expiresAt"             bun:"expires_at,type:BIGINT,notnull"`
	RevokedAt      *int64     `json:"revokedAt"             bun:"revoked_at,type:BIGINT,nullzero"`
	RevokedByID    pulid.ID   `json:"revokedById,omitempty" bun:"revoked_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt      int64      `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64      `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (g *Grant) IsActive(now int64) bool {
	return g != nil && g.RevokedAt == nil && now >= g.StartsAt && now < g.ExpiresAt
}

func (g *Grant) AllowsWrite(now int64) bool {
	return g.IsActive(now) && g.AccessMode.AllowsWrite()
}

func (g *Grant) Validate(multiErr *errortypes.MultiError, maxDuration time.Duration) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		g,
		validation.Field(&g.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&g.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&g.GrantedByID, validation.Required.Error("Granting user is required")),
		validation.Field(
			&g.AccessMode,
			validation.Required.Error("Access mode is required"),
			validation.By(func(any) error {
				if !g.AccessMode.IsValid() {
					return validation.NewError(
						"invalid",
						"Access mode must be read-only or read-write",
					)
				}
				return nil
			}),
		),
		validation.Field(
			&g.Note,
			validation.RuneLength(0, MaxNoteLength).Error("Note must be at most 500 characters"),
		),
		validation.Field(
			&g.ExpiresAt,
			validation.Required.Error("Expiry is required"),
			validation.By(func(any) error {
				if g.ExpiresAt <= g.StartsAt {
					return validation.NewError("invalid", "Access must end after it starts")
				}
				if time.Duration(g.ExpiresAt-g.StartsAt)*time.Second > maxDuration {
					return validation.NewError(
						"invalid",
						"Support access cannot be granted for longer than the allowed maximum",
					)
				}
				return nil
			}),
		),
	))
}

func (g *Grant) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if g.ID.IsNil() {
			g.ID = pulid.MustNew("sag_")
		}
		g.CreatedAt = now
		g.UpdatedAt = now
	case *bun.UpdateQuery:
		g.UpdatedAt = now
	}

	return nil
}

type Session struct {
	bun.BaseModel `bun:"table:support_sessions,alias:sps" json:"-"`

	ID                 pulid.ID  `json:"id"                         bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID  `json:"organizationId"             bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID  `json:"businessUnitId"             bun:"business_unit_id,type:VARCHAR(100),notnull"`
	GrantID            pulid.ID  `json:"grantId"                    bun:"grant_id,type:VARCHAR(100),notnull"`
	StaffUserID        pulid.ID  `json:"staffUserId"                bun:"staff_user_id,type:VARCHAR(100),notnull"`
	StaffName          string    `json:"staffName"                  bun:"staff_name,type:VARCHAR(255),notnull"`
	PrincipalUserID    pulid.ID  `json:"principalUserId"            bun:"principal_user_id,type:VARCHAR(100),notnull"`
	BaseSessionID      pulid.ID  `json:"-"                          bun:"base_session_id,type:VARCHAR(100),notnull"`
	SecretHash         string    `json:"-"                          bun:"secret_hash,type:VARCHAR(64),notnull"`
	Reason             string    `json:"reason"                     bun:"reason,type:VARCHAR(500),notnull"`
	TicketReference    string    `json:"ticketReference,omitempty"  bun:"ticket_reference,type:VARCHAR(100),nullzero"`
	StartedAt          int64     `json:"startedAt"                  bun:"started_at,type:BIGINT,notnull"`
	ExpiresAt          int64     `json:"expiresAt"                  bun:"expires_at,type:BIGINT,notnull"`
	ElevatedAt         *int64    `json:"elevatedAt"                 bun:"elevated_at,type:BIGINT,nullzero"`
	ElevatedUntil      *int64    `json:"elevatedUntil"              bun:"elevated_until,type:BIGINT,nullzero"`
	ElevationReason    string    `json:"elevationReason,omitempty"  bun:"elevation_reason,type:VARCHAR(500),nullzero"`
	ElevationTicket    string    `json:"elevationTicket,omitempty"  bun:"elevation_ticket,type:VARCHAR(100),nullzero"`
	ElevationCount     int       `json:"elevationCount"             bun:"elevation_count,type:INTEGER,notnull,default:0"`
	EndedAt            *int64    `json:"endedAt"                    bun:"ended_at,type:BIGINT,nullzero"`
	EndReason          EndReason `json:"endReason,omitempty"        bun:"end_reason,type:VARCHAR(30),nullzero"`
	LastSeenAt         int64     `json:"lastSeenAt"                 bun:"last_seen_at,type:BIGINT,notnull"`
	ClientIP           string    `json:"clientIp,omitempty"         bun:"client_ip,type:VARCHAR(64),nullzero"`
	UserAgent          string    `json:"userAgent,omitempty"        bun:"user_agent,type:VARCHAR(512),nullzero"`
	CreatedAt          int64     `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64     `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	OrganizationName   string    `json:"organizationName,omitempty" bun:"organization_name,scanonly"`
	GrantAccessMode    string    `json:"grantAccessMode,omitempty"  bun:"grant_access_mode,scanonly"`
	GrantExpiresAt     int64     `json:"grantExpiresAt,omitempty"   bun:"grant_expires_at,scanonly"`
	GrantRevokedAt     *int64    `json:"-"                          bun:"grant_revoked_at,scanonly"`
	GrantStartsAt      int64     `json:"-"                          bun:"grant_starts_at,scanonly"`
	StaffMemberActive  bool      `json:"-"                          bun:"staff_member_active,scanonly"`
	StaffMemberPresent bool      `json:"-"                          bun:"staff_member_present,scanonly"`
}

func (s *Session) IsOpen(now int64) bool {
	return s != nil && s.EndedAt == nil && now < s.ExpiresAt
}

func (s *Session) IsElevated(now int64) bool {
	return s.ElevatedUntil != nil && now < *s.ElevatedUntil
}

func (s *Session) Status(now int64) string {
	switch {
	case s.EndedAt != nil:
		return "ended"
	case now >= s.ExpiresAt:
		return "expired"
	default:
		return "active"
	}
}

func (s *Session) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		s,
		validation.Field(&s.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&s.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&s.GrantID, validation.Required.Error("Grant is required")),
		validation.Field(&s.StaffUserID, validation.Required.Error("Staff member is required")),
		validation.Field(
			&s.PrincipalUserID,
			validation.Required.Error("Support principal is required"),
		),
		validation.Field(&s.BaseSessionID, validation.Required.Error("Staff session is required")),
		validation.Field(&s.SecretHash, validation.Required.Error("Session secret is required")),
		validation.Field(&s.Reason, ReasonRules("Reason")...),
		validation.Field(
			&s.TicketReference,
			validation.RuneLength(0, MaxTicketLength).
				Error("Ticket reference must be at most 100 characters"),
		),
		validation.Field(
			&s.ExpiresAt,
			validation.By(func(any) error {
				if s.ExpiresAt <= s.StartedAt {
					return validation.NewError(
						"invalid",
						"A support session must end after it starts",
					)
				}
				return nil
			}),
		),
	))
}

func ReasonRules(label string) []validation.Rule {
	return []validation.Rule{
		validation.By(func(value any) error {
			text, _ := value.(string)
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				return validation.NewError("required", label+" is required")
			}
			if len([]rune(trimmed)) < MinReasonLength {
				return validation.NewError(
					"too_short",
					label+" must describe what you are doing in at least 10 characters",
				)
			}
			if len([]rune(trimmed)) > MaxReasonLength {
				return validation.NewError("too_long", label+" must be at most 500 characters")
			}
			return nil
		}),
	}
}

func (s *Session) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("sps_")
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

type StaffMember struct {
	bun.BaseModel `bun:"table:platform_staff_members,alias:psm" json:"-"`

	ID            pulid.ID  `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	UserID        pulid.ID  `json:"userId"                  bun:"user_id,type:VARCHAR(100),notnull"`
	Role          StaffRole `json:"role"                    bun:"role,type:VARCHAR(20),notnull"`
	Active        bool      `json:"active"                  bun:"active,type:BOOLEAN,notnull"`
	AddedBy       string    `json:"addedBy"                 bun:"added_by,type:VARCHAR(255),notnull"`
	DeactivatedBy string    `json:"deactivatedBy,omitempty" bun:"deactivated_by,type:VARCHAR(255),nullzero"`
	DeactivatedAt *int64    `json:"deactivatedAt"           bun:"deactivated_at,type:BIGINT,nullzero"`
	CreatedAt     int64     `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt     int64     `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UserName      string    `json:"userName,omitempty"      bun:"user_name,scanonly"`
	UserEmail     string    `json:"userEmail,omitempty"     bun:"user_email,scanonly"`
}

func (m *StaffMember) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		m,
		validation.Field(&m.UserID, validation.Required.Error("User is required")),
		validation.Field(
			&m.Role,
			validation.Required.Error("Role is required"),
			validation.By(func(any) error {
				if !m.Role.IsValid() {
					return validation.NewError("invalid", "Role must be support or engineer")
				}
				return nil
			}),
		),
		validation.Field(
			&m.AddedBy,
			validation.Required.Error("Who added the staff member is required"),
			validation.RuneLength(1, MaxAddedByLength).
				Error("Added by must be at most 255 characters"),
		),
	))
}

func (m *StaffMember) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("psm_")
		}
		m.CreatedAt = now
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}

type Principal struct {
	bun.BaseModel `bun:"table:support_principals,alias:spr" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	StaffUserID    pulid.ID `json:"staffUserId"    bun:"staff_user_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (p *Principal) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("spr_")
		}
		p.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
