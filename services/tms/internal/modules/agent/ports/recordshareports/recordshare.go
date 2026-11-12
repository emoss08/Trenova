package recordshareports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RecordRequest names one record whose shares are read.
type RecordRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
}

type Repository interface {
	// Upsert records each share, counting and re-dating one that already
	// exists for the same record and recipient.
	Upsert(ctx context.Context, shares []*recordshare.Share) error
	// Get is one share of a record, standing or taken back.
	Get(ctx context.Context, req *ShareTarget) (*recordshare.Share, error)
	// Revoke takes a standing share back. It reports false when the share was
	// already taken back.
	Revoke(ctx context.Context, share *recordshare.Share) (bool, error)
	// ListByRecord is who the record is shared with now, newest first.
	ListByRecord(ctx context.Context, req *RecordRequest) ([]*recordshare.Share, error)
	// SharedWith is which of the records are shared with the person now, in
	// one read.
	SharedWith(ctx context.Context, req *SharedWithRequest) ([]pulid.ID, error)
}

// ShareTarget names one share of one record.
type ShareTarget struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
	ShareID    pulid.ID
}

// SharedWithRequest asks which of a page of records one person was sent.
type SharedWithRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	RecordIDs  []pulid.ID
}

// ShareRequest shares one record with teammates.
type ShareRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
	UserIDs    []pulid.ID
	Note       string
	// Path is where in the app the link opens: the record's page on the tab,
	// or the Desk page, the sharer was on.
	Path string
}

type CandidatesRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
	Query      string
	Limit      int
}

type EmailStatus string

const (
	EmailQueued        = EmailStatus("Queued")
	EmailPartial       = EmailStatus("Partial")
	EmailFailed        = EmailStatus("Failed")
	EmailNotConfigured = EmailStatus("NotConfigured")
)

type ShareResult struct {
	Shares         []*recordshare.Share
	RecipientCount int
	EmailsQueued   int
	EmailStatus    EmailStatus
}

// Record is what was shared, as the people told about it read it.
type Record struct {
	Kind     recordkind.Kind
	ID       pulid.ID
	Label    string
	Subtitle string
}

// Title names the record inside a sentence: its kind, then its label
// ("invoice INV-2026-1042").
func (r *Record) Title() string {
	return r.Kind.Noun() + " " + r.Label
}

type RecipientPreview struct {
	UserID       pulid.ID
	Name         string
	EmailAddress string
	Emailed      bool
	Subject      string
	Body         string
}

// Preview is what a share would send, before anything is sent.
type Preview struct {
	Record          *Record
	SharedByName    string
	Note            string
	Path            string
	EmailConfigured bool
	Recipients      []RecipientPreview
}

type Service interface {
	List(
		ctx context.Context,
		req *RecordRequest,
		actor *permissionports.RequestActor,
	) ([]*recordshare.Share, error)
	Share(
		ctx context.Context,
		req *ShareRequest,
		actor *permissionports.RequestActor,
	) (*ShareResult, error)
	// Unshare takes a share back: its recipient no longer reads the record's
	// case through it. The person who shared it may, and so may whoever owns
	// the case or may give cases to people. It returns who the record is
	// still shared with.
	Unshare(
		ctx context.Context,
		req *ShareTarget,
		actor *permissionports.RequestActor,
	) ([]*recordshare.Share, error)
	// PreviewShare plans a share and renders what each recipient would get,
	// without recording or sending anything.
	PreviewShare(
		ctx context.Context,
		req *ShareRequest,
		actor *permissionports.RequestActor,
	) (*Preview, error)
	// ListCandidates are the teammates the record can be shared with: active
	// members, other than the sharer, whose role already lets them open it.
	ListCandidates(
		ctx context.Context,
		req *CandidatesRequest,
		actor *permissionports.RequestActor,
	) ([]*tenant.User, error)
}
