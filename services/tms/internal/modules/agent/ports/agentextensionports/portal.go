package agentextensionports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var ErrPortalRunExists = errors.New("a portal run with this idempotency key already exists")

type PortalSiteRepository interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*agentextension.PortalSite, error)
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*agentextension.PortalSite, error)
	Create(ctx context.Context, site *agentextension.PortalSite) (*agentextension.PortalSite, error)
	Update(ctx context.Context, site *agentextension.PortalSite) (*agentextension.PortalSite, error)
	Delete(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error
}

type ListPortalRunsRequest struct {
	TenantInfo pagination.TenantInfo
	SiteID     pulid.ID
	Before     int64
	Limit      int
}

type PortalRunRepository interface {
	Create(ctx context.Context, run *agentextension.PortalRun) (*agentextension.PortalRun, error)
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*agentextension.PortalRun, error)
	GetByIdempotencyKey(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		key string,
	) (*agentextension.PortalRun, error)
	Update(ctx context.Context, run *agentextension.PortalRun) (*agentextension.PortalRun, error)
	List(ctx context.Context, req *ListPortalRunsRequest) ([]*agentextension.PortalRun, error)
}

type PortalTaskRequest struct {
	Settings       map[string]string
	Instruction    string
	Guidance       string
	StartURL       string
	AllowedDomains []string
	Secrets        map[string]string
	ProfileID      string
	Metadata       map[string]string

	// SessionID continues in a browser a task already opened, on the page it
	// stopped at, instead of opening a new one.
	SessionID string
}

type PortalTaskStarted struct {
	TaskID    string
	SessionID string
}

type PortalTaskPhase string

const (
	PortalTaskRunning  = PortalTaskPhase("Running")
	PortalTaskFinished = PortalTaskPhase("Finished")
	PortalTaskFailed   = PortalTaskPhase("Failed")
	PortalTaskStopped  = PortalTaskPhase("Stopped")
)

type PortalTaskState struct {
	Phase       PortalTaskPhase
	Answer      string
	Succeeded   *bool
	VisitedURLs []string
	Steps       int
	CostUSD     decimal.Decimal
	SessionID   string
}

type PortalRecording struct {
	RecordingURL string
	ShareURL     string
}

type BrowserProvider interface {
	StartTask(ctx context.Context, req *PortalTaskRequest) (*PortalTaskStarted, error)
	TaskState(
		ctx context.Context,
		settings map[string]string,
		taskID string,
	) (*PortalTaskState, error)
	StopTask(ctx context.Context, settings map[string]string, taskID string) error
	StopSession(ctx context.Context, settings map[string]string, sessionID string) error
	Recording(
		ctx context.Context,
		settings map[string]string,
		sessionID string,
	) (*PortalRecording, error)
}

type PortalSiteView struct {
	ID               pulid.ID `json:"id"`
	Name             string   `json:"name"`
	Purpose          string   `json:"purpose"`
	Domains          []string `json:"domains"`
	StartURL         string   `json:"startUrl"`
	BrowserProfileID string   `json:"browserProfileId"`
	AllowActions     bool     `json:"allowActions"`
	Enabled          bool     `json:"enabled"`
	HasSignIn        bool     `json:"hasSignIn"`
	HasAuthenticator bool     `json:"hasAuthenticator"`
	Version          int64    `json:"version"`
	CreatedAt        int64    `json:"createdAt"`
	UpdatedAt        int64    `json:"updatedAt"`
}

type SavePortalSiteRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	ID                 pulid.ID              `json:"-"`
	Name               string                `json:"name"`
	Purpose            string                `json:"purpose"`
	Domains            []string              `json:"domains"`
	StartURL           string                `json:"startUrl"`
	Username           string                `json:"username"`
	Password           string                `json:"password"`
	AuthenticatorKey   string                `json:"authenticatorKey"`
	ClearSignIn        bool                  `json:"clearSignIn"`
	ClearAuthenticator bool                  `json:"clearAuthenticator"`
	BrowserProfileID   string                `json:"browserProfileId"`
	AllowActions       bool                  `json:"allowActions"`
	Enabled            bool                  `json:"enabled"`
	Version            int64                 `json:"version"`
}

type PortalRunView struct {
	ID              pulid.ID                       `json:"id"`
	SiteID          pulid.ID                       `json:"siteId"`
	SiteName        string                         `json:"siteName"`
	ExtensionType   agentextension.Type            `json:"extensionType"`
	Mode            agentextension.PortalRunMode   `json:"mode"`
	Status          agentextension.PortalRunStatus `json:"status"`
	Instruction     string                         `json:"instruction"`
	Details         map[string]string              `json:"details"`
	UsedSignIn      bool                           `json:"usedSignIn"`
	Answer          string                         `json:"answer"`
	VisitedURLs     []string                       `json:"visitedUrls"`
	OffSiteURLs     []string                       `json:"offSiteUrls"`
	Steps           int                            `json:"steps"`
	CostUSD         decimal.Decimal                `json:"costUsd"`
	Failure         string                         `json:"failure"`
	AgentRunID      pulid.ID                       `json:"agentRunId"`
	ThreadID        pulid.ID                       `json:"threadId"`
	ProposalID      pulid.ID                       `json:"proposalId"`
	RequestedByID   pulid.ID                       `json:"requestedById"`
	HasRecording    bool                           `json:"hasRecording"`
	CodePrompt      string                         `json:"codePrompt"`
	CodeExpiresAt   *int64                         `json:"codeExpiresAt"`
	CodeAttempts    int                            `json:"codeAttempts"`
	StartedAt       int64                          `json:"startedAt"`
	FinishedAt      *int64                         `json:"finishedAt"`
	VendorSessionID string                         `json:"-"`
}

type PortalRunListResponse struct {
	Items      []PortalRunView `json:"items"`
	NextBefore int64           `json:"nextBefore"`
}

type PortalRecordingResponse struct {
	RecordingURL string `json:"recordingUrl"`
	ShareURL     string `json:"shareUrl"`
}

type PortalSites interface {
	ListSites(ctx context.Context, tenantInfo pagination.TenantInfo) ([]PortalSiteView, error)
	CreateSite(ctx context.Context, req *SavePortalSiteRequest) (*PortalSiteView, error)
	UpdateSite(ctx context.Context, req *SavePortalSiteRequest) (*PortalSiteView, error)
	DeleteSite(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error
	ListRuns(ctx context.Context, req *ListPortalRunsRequest) (*PortalRunListResponse, error)
	RunRecording(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*PortalRecordingResponse, error)
}

type PortalSiteSummary struct {
	ID           pulid.ID
	Name         string
	Purpose      string
	Domains      []string
	StartURL     string
	AllowActions bool
	HasSignIn    bool
}

type PortalTaskOwner struct {
	Actor        *permissionports.RequestActor
	DefinitionID pulid.ID
	RunID        pulid.ID
	ThreadID     pulid.ID
	ProposalID   pulid.ID
}

type PortalReadRequest struct {
	TenantInfo pagination.TenantInfo
	SiteID     pulid.ID
	Question   string
	Details    map[string]string
	Owner      PortalTaskOwner
}

type PortalReadOutcome struct {
	RunID       pulid.ID
	Site        PortalSiteSummary
	Status      agentextension.PortalRunStatus
	Answer      string
	VisitedURLs []string
	Steps       int
	RetrievedAt int64

	// CodePrompt is how the portal asked for the sign-in code it sent the
	// person, when the read is waiting on one; written by the portal.
	CodePrompt string
}

type PortalActionRequest struct {
	TenantInfo     pagination.TenantInfo
	SiteID         pulid.ID
	Instruction    string
	Details        map[string]string
	IdempotencyKey string
	Owner          PortalTaskOwner
}

type PortalActionPlan struct {
	Site              PortalSiteSummary
	Instruction       string
	Details           map[string]string
	SentTask          string
	UsesSignIn        bool
	UsesAuthenticator bool
	Vendor            string
}

type PortalActionStarted struct {
	RunID   pulid.ID
	Site    PortalSiteSummary
	Status  agentextension.PortalRunStatus
	Answer  string
	WaitSet bool
	Resumed bool
}

type PortalAutomator interface {
	ListPortalSites(ctx context.Context, tenantInfo pagination.TenantInfo) ([]PortalSiteSummary, error)
	ReadPortal(ctx context.Context, req *PortalReadRequest) (*PortalReadOutcome, error)
	PlanPortalAction(ctx context.Context, req *PortalActionRequest) (*PortalActionPlan, error)
	StartPortalAction(ctx context.Context, req *PortalActionRequest) (*PortalActionStarted, error)
	TrackPortalRun(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
		siteName string,
	) error
}

type PortalRunFinisher interface {
	PortalRun(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
	) (*agentextension.PortalRun, error)
	PollPortalRun(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
	) (*PortalPoll, error)
	StopPortalRun(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runID pulid.ID,
		status agentextension.PortalRunStatus,
		reason string,
	) (*agentextension.PortalRun, error)
}

// PortalPoll is a run as one poll left it, and whether that poll is the one
// that found the portal asking for a sign-in code.
type PortalPoll struct {
	Run           *agentextension.PortalRun
	CodeRequested bool
}

type PortalRunWaits interface {
	RegisterPortalRunWait(ctx context.Context, req *RegisterPortalRunWaitRequest) error
	NotifyPortalRun(ctx context.Context, tenantInfo pagination.TenantInfo, run *agentextension.PortalRun)
	// AnnouncePortalRun tells the people whose work waits on a run that it
	// changed without ending, so their screen shows what it now needs.
	AnnouncePortalRun(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		run *agentextension.PortalRun,
	)
}

type ProvidePortalCodeRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Code       string

	// Administers lets someone who oversees agents enter a code for any run;
	// anyone else only for a run that was asked for them.
	Administers bool
}

// PortalSignInCodes takes a sign-in code a person relays from their phone or
// inbox and continues the portal task with it, in the browser that asked.
type PortalSignInCodes interface {
	ProvidePortalCode(ctx context.Context, req *ProvidePortalCodeRequest) (*PortalRunView, error)
}

type RegisterPortalRunWaitRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	SiteName   string
	Owner      PortalTaskOwner
}
