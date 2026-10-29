package agentextensionports

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var ErrWebWatchFindingExists = errors.New("this run of the web watch is already recorded")

type ListWebWatchesRequest struct {
	TenantInfo   pagination.TenantInfo
	Status       extensionwebwatch.WebWatchStatus
	ThreadID     pulid.ID
	RequestedBy  pulid.ID
	DefinitionID pulid.ID
	Before       int64
	Limit        int
}

type WebWatchRepository interface {
	Create(
		ctx context.Context,
		watch *extensionwebwatch.WebWatch,
	) (*extensionwebwatch.WebWatch, error)
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*extensionwebwatch.WebWatch, error)
	Update(
		ctx context.Context,
		watch *extensionwebwatch.WebWatch,
	) (*extensionwebwatch.WebWatch, error)
	List(ctx context.Context, req *ListWebWatchesRequest) ([]*extensionwebwatch.WebWatch, error)
	CountActive(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error)
}

type WebWatchFindingRepository interface {
	// Create records a run's finding once; a run already recorded is
	// ErrWebWatchFindingExists.
	Create(
		ctx context.Context,
		finding *extensionwebwatch.WebWatchFinding,
	) (*extensionwebwatch.WebWatchFinding, error)
	ListByWatch(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		watchID pulid.ID,
		limit int,
	) ([]*extensionwebwatch.WebWatchFinding, error)
	ListUndelivered(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		watchID pulid.ID,
	) ([]*extensionwebwatch.WebWatchFinding, error)
	MarkDelivered(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
		at int64,
	) error
}

type WebWatchCreateRequest struct {
	Settings       map[string]string
	Name           string
	Query          string
	IncludeDomains []string
	ExcludeDomains []string
	Cadence        extensionwebwatch.WebWatchCadence
	WebhookURL     string
	Metadata       map[string]string
}

type WebWatchCreated struct {
	VendorID string
	Secret   string
}

type WebWatchRunPhase string

const (
	WebWatchRunPending   = WebWatchRunPhase("Pending")
	WebWatchRunCompleted = WebWatchRunPhase("Completed")
	WebWatchRunFailed    = WebWatchRunPhase("Failed")
	WebWatchRunCancelled = WebWatchRunPhase("Cancelled")
)

// Settled reports a run that will not change again.
func (p WebWatchRunPhase) Settled() bool {
	return p == WebWatchRunCompleted || p == WebWatchRunFailed || p == WebWatchRunCancelled
}

// WebWatchRun is one scheduled search the vendor ran, and what it found that
// earlier runs had not.
type WebWatchRun struct {
	ID        string
	Phase     WebWatchRunPhase
	Summary   string
	Results   []extensionwebwatch.WebWatchResult
	Failure   string
	CreatedAt int64
	SettledAt int64
	CostUSD   decimal.Decimal
}

type WebWatchRunsRequest struct {
	Settings map[string]string
	VendorID string
	// After is when the last run already counted was made; zero reads every
	// run in full.
	After int64
}

type WebWatchVendorState struct {
	// Disabled is a watch the vendor turned off on its own, such as after its
	// key kept failing.
	Disabled bool
}

type WebWatchProvider interface {
	CreateWatch(ctx context.Context, req *WebWatchCreateRequest) (*WebWatchCreated, error)
	DeleteWatch(ctx context.Context, settings map[string]string, vendorID string) error
	WatchState(
		ctx context.Context,
		settings map[string]string,
		vendorID string,
	) (*WebWatchVendorState, error)
	// WatchRuns are the vendor's latest runs, newest first, each with its
	// full output once it has completed. A run made at or before after was
	// already counted, so its output is not read again.
	WatchRuns(ctx context.Context, req *WebWatchRunsRequest) ([]WebWatchRun, error)
	// VerifyDelivery checks that a webhook delivery was signed with the
	// watch's secret.
	VerifyDelivery(delivery *WebWatchDelivery, secret string) error
}

// WebWatchDelivery is a vendor's webhook as it arrived: its headers, which
// carry the signature in the vendor's own header, its raw body and when it
// came.
type WebWatchDelivery struct {
	Header http.Header
	Body   []byte
	At     time.Time
}

type WebWatchRequest struct {
	TenantInfo pagination.TenantInfo
	Query      string
	Purpose    string
	Sites      []string
	Cadence    extensionwebwatch.WebWatchCadence
	Days       int
	Owner      PortalTaskOwner
}

// WebWatchPlan is what setting a watch would do, checked without reaching
// the vendor: the preview and the watch itself both read it.
type WebWatchPlan struct {
	Query          string
	Purpose        string
	Sites          []string
	Cadence        extensionwebwatch.WebWatchCadence
	Days           int
	EndsAt         int64
	Vendor         string
	CostPerCheck   decimal.Decimal
	EstimatedCost  decimal.Decimal
	ActiveWatches  int
	WatchesAllowed int
}

type WebWatchView struct {
	ID                pulid.ID                          `json:"id"`
	ExtensionType     agentextension.Type               `json:"extensionType"`
	Query             string                            `json:"query"`
	Purpose           string                            `json:"purpose"`
	Sites             []string                          `json:"sites"`
	Cadence           extensionwebwatch.WebWatchCadence `json:"cadence"`
	Status            extensionwebwatch.WebWatchStatus  `json:"status"`
	EndsAt            int64                             `json:"endsAt"`
	LastCheckedAt     *int64                            `json:"lastCheckedAt"`
	LastRunAt         *int64                            `json:"lastRunAt"`
	RunCount          int                               `json:"runCount"`
	FindingCount      int                               `json:"findingCount"`
	CostUSD           decimal.Decimal                   `json:"costUsd"`
	Failure           string                            `json:"failure"`
	AgentDefinitionID pulid.ID                          `json:"agentDefinitionId"`
	AgentRunID        pulid.ID                          `json:"agentRunId"`
	ThreadID          pulid.ID                          `json:"threadId"`
	RequestedByID     pulid.ID                          `json:"requestedById"`
	StoppedByID       pulid.ID                          `json:"stoppedById"`
	FinishedAt        *int64                            `json:"finishedAt"`
	CreatedAt         int64                             `json:"createdAt"`
}

type WebWatchListResponse struct {
	Items      []WebWatchView `json:"items"`
	NextBefore int64          `json:"nextBefore"`
}

type WebWatchFindingView struct {
	ID          pulid.ID                           `json:"id"`
	Summary     string                             `json:"summary"`
	Results     []extensionwebwatch.WebWatchResult `json:"results"`
	FoundAt     int64                              `json:"foundAt"`
	DeliveredAt *int64                             `json:"deliveredAt"`
}

type WebWatchFindingsResponse struct {
	Items []WebWatchFindingView `json:"items"`
}

type StopWebWatchRequest struct {
	TenantInfo pagination.TenantInfo
	WatchID    pulid.ID
	StoppedBy  pulid.ID

	// DefinitionID is the agent stopping a watch from a background run, which
	// may stop one set by its own background work.
	DefinitionID pulid.ID

	// Administers lets someone who oversees agents stop any watch; anyone
	// else only one set for them.
	Administers bool
}

// WebWatcher is what agents and people do with web watches.
type WebWatcher interface {
	PlanWebWatch(ctx context.Context, req *WebWatchRequest) (*WebWatchPlan, error)
	WatchWeb(ctx context.Context, req *WebWatchRequest) (*WebWatchView, error)
	ListWebWatches(ctx context.Context, req *ListWebWatchesRequest) (*WebWatchListResponse, error)
	StopWebWatch(ctx context.Context, req *StopWebWatchRequest) (*WebWatchView, error)
}

// WebWatchCheck is a watch as one check left it, and what it found that is
// not yet handed to the work waiting on it.
type WebWatchCheck struct {
	Watch       *extensionwebwatch.WebWatch
	Undelivered []*extensionwebwatch.WebWatchFinding
	Ended       bool
}

// WebWatchRunner is what the workflow that follows a watch asks of it.
type WebWatchRunner interface {
	CheckWebWatch(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		watchID pulid.ID,
	) (*WebWatchCheck, error)
	MarkWebWatchDelivered(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		findings []*extensionwebwatch.WebWatchFinding,
	) error
}

type RegisterWebWatchWaitRequest struct {
	TenantInfo pagination.TenantInfo
	Watch      *extensionwebwatch.WebWatch
	Owner      PortalTaskOwner
}

// WebWatchWaits parks the work that set a watch until the watch finds
// something, and picks it up with what it found.
type WebWatchWaits interface {
	RegisterWebWatchWait(ctx context.Context, req *RegisterWebWatchWaitRequest) error
	// NotifyWebWatch meets the waits on a watch with what it found, and sets
	// the next one while the watch goes on.
	NotifyWebWatch(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		watch *extensionwebwatch.WebWatch,
		findings []*extensionwebwatch.WebWatchFinding,
	) error
	// EndWebWatchWaits closes the waits on a watch that stopped before its
	// time: quietly for one a person or the agent stopped, with the reason
	// for one the vendor could no longer run.
	EndWebWatchWaits(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		watch *extensionwebwatch.WebWatch,
	)
}

// WebWatchDeliveries takes a vendor's webhook for a watch and checks the
// watch at once instead of at its next scheduled look.
type WebWatchDeliveries interface {
	ReceiveWebWatchDelivery(ctx context.Context, req *ReceiveWebWatchDeliveryRequest) error
}

type ReceiveWebWatchDeliveryRequest struct {
	TenantInfo pagination.TenantInfo
	WatchID    pulid.ID
	Delivery   WebWatchDelivery
}
