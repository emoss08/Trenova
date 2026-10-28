package ediconnectiondraftports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CredentialKind string

const (
	CredentialKindSecret  = CredentialKind("Secret")
	CredentialKindSetting = CredentialKind("Setting")
)

type CredentialRequirement struct {
	Keys     []string       `json:"keys"`
	Label    string         `json:"label"`
	Kind     CredentialKind `json:"kind"`
	Required bool           `json:"required"`
	Missing  bool           `json:"missing"`
}

type RequirementUsage string

const (
	RequirementUsageMandatory   = RequirementUsage("Mandatory")
	RequirementUsageOptional    = RequirementUsage("Optional")
	RequirementUsageConditional = RequirementUsage("Conditional")
	RequirementUsageNotUsed     = RequirementUsage("NotUsed")
)

type GapKind string

const (
	GapKindUnsupported = GapKind("Unsupported")
	GapKindAmbiguous   = GapKind("Ambiguous")
)

type ResolutionSource string

const (
	ResolutionSourceCandidate = ResolutionSource("Candidate")
	ResolutionSourcePerson    = ResolutionSource("Person")
)

type ValueStatus string

const (
	ValueStatusAccepted   = ValueStatus("Accepted")
	ValueStatusDefaulted  = ValueStatus("Defaulted")
	ValueStatusKept       = ValueStatus("Kept")
	ValueStatusHeld       = ValueStatus("Held")
	ValueStatusUnverified = ValueStatus("Unverified")
	ValueStatusInvalid    = ValueStatus("Invalid")
	ValueStatusMissing    = ValueStatus("Missing")
)

func (s ValueStatus) Usable() bool {
	switch s {
	case ValueStatusAccepted, ValueStatusDefaulted, ValueStatusKept:
		return true
	case ValueStatusHeld, ValueStatusUnverified, ValueStatusInvalid, ValueStatusMissing:
		return false
	default:
		return false
	}
}

type ChangeKind string

const (
	ChangeKindAdd    = ChangeKind("Add")
	ChangeKindChange = ChangeKind("Change")
	ChangeKindSame   = ChangeKind("Same")
	ChangeKindNone   = ChangeKind("None")
)

func (c ChangeKind) Writes() bool {
	return c == ChangeKindAdd || c == ChangeKindChange
}

type SourceKind string

const (
	SourceKindDocument      = SourceKind("Document")
	SourceKindSample        = SourceKind("Sample")
	SourceKindPerson        = SourceKind("Person")
	SourceKindConfiguration = SourceKind("Configuration")
	SourceKindDefault       = SourceKind("Default")
	SourceKindAgent         = SourceKind("Agent")
)

type PartnerMatchKind string

const (
	PartnerMatchNew       = PartnerMatchKind("New")
	PartnerMatchExisting  = PartnerMatchKind("Existing")
	PartnerMatchAmbiguous = PartnerMatchKind("Ambiguous")
)

type SpecPage struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type SpecDocument struct {
	ID       pulid.ID   `json:"id"`
	FileName string     `json:"fileName"`
	Pages    []SpecPage `json:"pages"`
}

type DraftCitation struct {
	Page    int    `json:"page"`
	Quote   string `json:"quote"`
	Section string `json:"section,omitempty"`
}

type DraftValueInput struct {
	Field      string        `json:"field"`
	Value      string        `json:"value"`
	Confidence float64       `json:"confidence"`
	Citation   DraftCitation `json:"citation"`
}

type DraftRequirementInput struct {
	TransactionSet  edi.TransactionSet    `json:"transactionSet"`
	Direction       edi.DocumentDirection `json:"direction"`
	SegmentID       string                `json:"segment"`
	ElementPosition int                   `json:"elementPosition,omitempty"`
	Usage           RequirementUsage      `json:"usage"`
	Codes           []string              `json:"codes,omitempty"`
	Citation        DraftCitation         `json:"citation"`
}

type DraftMappingInput struct {
	EntityType  edi.MappingEntityType `json:"entityType"`
	SourceValue string                `json:"sourceValue"`
	SourceLabel string                `json:"sourceLabel,omitempty"`
	TargetID    pulid.ID              `json:"targetId"`
	Confidence  float64               `json:"confidence"`
	Citation    DraftCitation         `json:"citation"`
}

type DraftGapInput struct {
	Kind     GapKind       `json:"kind"`
	Summary  string        `json:"summary"`
	Citation DraftCitation `json:"citation"`
}

type DraftResolutionInput struct {
	Field  string           `json:"field"`
	Value  string           `json:"value"`
	Source ResolutionSource `json:"source"`
	Note   string           `json:"note,omitempty"`
}

type ConnectionDraftRequest struct {
	TenantInfo   pagination.TenantInfo   `json:"-"`
	Document     SpecDocument            `json:"-"`
	PartnerID    pulid.ID                `json:"partnerId,omitempty"`
	CustomerID   pulid.ID                `json:"customerId,omitempty"`
	Values       []DraftValueInput       `json:"values"`
	Requirements []DraftRequirementInput `json:"requirements,omitempty"`
	Mappings     []DraftMappingInput     `json:"mappings,omitempty"`
	Gaps         []DraftGapInput         `json:"gaps,omitempty"`
	Resolutions  []DraftResolutionInput  `json:"resolutions,omitempty"`
}

type DraftSource struct {
	Kind    SourceKind `json:"kind"`
	Page    int        `json:"page,omitempty"`
	Quote   string     `json:"quote,omitempty"`
	Section string     `json:"section,omitempty"`
	Note    string     `json:"note,omitempty"`
}

type DraftCandidate struct {
	Value      string      `json:"value"`
	Confidence float64     `json:"confidence"`
	Source     DraftSource `json:"source"`
	Problem    string      `json:"problem,omitempty"`
}

type DraftValue struct {
	Field      string           `json:"field"`
	Label      string           `json:"label"`
	Value      string           `json:"value"`
	Current    string           `json:"current,omitempty"`
	Change     ChangeKind       `json:"change"`
	Status     ValueStatus      `json:"status"`
	Required   bool             `json:"required"`
	Confidence float64          `json:"confidence,omitempty"`
	Source     DraftSource      `json:"source"`
	Candidates []DraftCandidate `json:"candidates,omitempty"`
	Reason     string           `json:"reason,omitempty"`
}

type PartnerCandidate struct {
	PartnerID pulid.ID `json:"partnerId"`
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Reasons   []string `json:"reasons"`
}

type DraftPartner struct {
	Match      PartnerMatchKind   `json:"match"`
	PartnerID  pulid.ID           `json:"partnerId,omitempty"`
	Version    int64              `json:"version,omitempty"`
	Status     domaintypes.Status `json:"status"`
	Reasons    []string           `json:"reasons,omitempty"`
	Candidates []PartnerCandidate `json:"candidates,omitempty"`
	CustomerID pulid.ID           `json:"customerId,omitempty"`
	Change     ChangeKind         `json:"change"`
	Fields     []DraftValue       `json:"fields"`
}

type DraftSetDirection struct {
	TransactionSet edi.TransactionSet    `json:"transactionSet"`
	Direction      edi.DocumentDirection `json:"direction"`
}

type DraftConnection struct {
	Method   edi.ConnectionMethod `json:"method,omitempty"`
	Inbound  bool                 `json:"inbound"`
	Outbound bool                 `json:"outbound"`
	Sets     []DraftSetDirection  `json:"sets"`
}

type DraftTransport struct {
	ProfileID   pulid.ID                `json:"profileId,omitempty"`
	Version     int64                   `json:"version,omitempty"`
	Name        string                  `json:"name"`
	Method      edi.ConnectionMethod    `json:"method"`
	Status      domaintypes.Status      `json:"status"`
	Change      ChangeKind              `json:"change"`
	Fields      []DraftValue            `json:"fields"`
	Credentials []CredentialRequirement `json:"credentials"`
}

type DraftRequirementCheck struct {
	SegmentID       string           `json:"segmentId"`
	ElementPosition int              `json:"elementPosition,omitempty"`
	Usage           RequirementUsage `json:"usage"`
	Codes           []string         `json:"codes,omitempty"`
	Problem         string           `json:"problem"`
	Source          DraftSource      `json:"source"`
}

type DraftControlNumber struct {
	Kind    edi.ControlNumberKind `json:"kind"`
	Start   int64                 `json:"start"`
	Current int64                 `json:"current,omitempty"`
	Change  ChangeKind            `json:"change"`
	Source  DraftSource           `json:"source"`
	Problem string                `json:"problem,omitempty"`
}

type DraftSet struct {
	TransactionSet edi.TransactionSet      `json:"transactionSet"`
	Direction      edi.DocumentDirection   `json:"direction"`
	DocumentTypeID pulid.ID                `json:"documentTypeId,omitempty"`
	Change         ChangeKind              `json:"change"`
	Fields         []DraftValue            `json:"fields"`
	Requirements   int                     `json:"requirements"`
	Unmet          []DraftRequirementCheck `json:"unmet"`
	Unverified     []DraftRequirementCheck `json:"unverified,omitempty"`
	ControlNumbers []DraftControlNumber    `json:"controlNumbers,omitempty"`
}

type DraftProjectFile struct {
	Path   string     `json:"path"`
	Kind   string     `json:"kind"`
	Change ChangeKind `json:"change"`
}

type DraftDiagnostic struct {
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

type DraftProject struct {
	ProjectID   pulid.ID           `json:"projectId,omitempty"`
	Role        string             `json:"role"`
	Change      ChangeKind         `json:"change"`
	Files       []DraftProjectFile `json:"files"`
	Errors      int                `json:"errors"`
	Warnings    int                `json:"warnings"`
	Diagnostics []DraftDiagnostic  `json:"diagnostics"`
}

type DraftMapping struct {
	EntityType      edi.MappingEntityType `json:"entityType"`
	SourceID        pulid.ID              `json:"sourceId"`
	SourceLabel     string                `json:"sourceLabel,omitempty"`
	TargetID        pulid.ID              `json:"targetId,omitempty"`
	TargetLabel     string                `json:"targetLabel,omitempty"`
	CurrentTargetID pulid.ID              `json:"currentTargetId,omitempty"`
	Change          ChangeKind            `json:"change"`
	Status          ValueStatus           `json:"status"`
	Confidence      float64               `json:"confidence,omitempty"`
	Source          DraftSource           `json:"source"`
	Reason          string                `json:"reason,omitempty"`
}

type DraftSampleCheck struct {
	Field    string `json:"field"`
	Label    string `json:"label"`
	Expected string `json:"expected"`
	Found    string `json:"found"`
	Matches  bool   `json:"matches"`
}

type DraftUnmappedCode struct {
	EntityType  edi.MappingEntityType `json:"entityType"`
	SourceID    pulid.ID              `json:"sourceId"`
	SourceLabel string                `json:"sourceLabel,omitempty"`
}

type DraftSample struct {
	Page           int                            `json:"page"`
	TransactionSet edi.TransactionSet             `json:"transactionSet,omitempty"`
	Direction      edi.DocumentDirection          `json:"direction,omitempty"`
	Parsed         bool                           `json:"parsed"`
	Error          string                         `json:"error,omitempty"`
	Overview       *edix12inspect.InspectOverview `json:"overview,omitempty"`
	Checks         []DraftSampleCheck             `json:"checks,omitempty"`
	Unmapped       []DraftUnmappedCode            `json:"unmapped,omitempty"`
}

type DraftGap struct {
	Kind    GapKind     `json:"kind"`
	Summary string      `json:"summary"`
	Source  DraftSource `json:"source"`
	Problem string      `json:"problem,omitempty"`
}

type DraftFailure struct {
	Record  string `json:"record"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type DraftKept struct {
	Record string `json:"record"`
	Label  string `json:"label"`
}

type ConnectionDraft struct {
	DocumentID     pulid.ID        `json:"documentId"`
	FileName       string          `json:"fileName"`
	PageCount      int             `json:"pageCount"`
	DocumentDigest string          `json:"documentDigest"`
	Partner        DraftPartner    `json:"partner"`
	Connection     DraftConnection `json:"connection"`
	Interchange    []DraftValue    `json:"interchange"`
	Transport      *DraftTransport `json:"transport,omitempty"`
	Sets           []DraftSet      `json:"sets"`
	Project        *DraftProject   `json:"project,omitempty"`
	Mappings       []DraftMapping  `json:"mappings"`
	Kept           []DraftKept     `json:"kept,omitempty"`
	Samples        []DraftSample   `json:"samples"`
	Gaps           []DraftGap      `json:"gaps"`
	Decisions      []DraftValue    `json:"decisions"`
	Failures       []DraftFailure  `json:"failures"`
	Blockers       []string        `json:"blockers"`
	Changes        int             `json:"changes"`
	Ready          bool            `json:"ready"`
}

func (d *ConnectionDraft) CredentialsToEnter() []string {
	if d == nil || d.Transport == nil {
		return nil
	}
	labels := make([]string, 0, len(d.Transport.Credentials))
	for _, credential := range d.Transport.Credentials {
		if credential.Required && credential.Missing {
			labels = append(labels, credential.Label)
		}
	}
	return labels
}

type ApplyConnectionDraftRequest struct {
	Draft                  *ConnectionDraftRequest       `json:"draft"`
	ExpectedDocumentDigest string                        `json:"expectedDocumentDigest"`
	Actor                  *permissionports.RequestActor `json:"-"`
}

type DraftWrite struct {
	Record string   `json:"record"`
	ID     pulid.ID `json:"id"`
	Action string   `json:"action"`
	Label  string   `json:"label"`
}

type AppliedConnectionDraft struct {
	PartnerID pulid.ID         `json:"partnerId"`
	Created   bool             `json:"created"`
	Writes    []DraftWrite     `json:"writes"`
	Draft     *ConnectionDraft `json:"draft"`
}

type PlannedPartner struct {
	Before *edi.EDIPartner `json:"before,omitempty"`
	After  *edi.EDIPartner `json:"after"`
	Change ChangeKind      `json:"change"`
}

type PlannedTransport struct {
	Before *edi.EDICommunicationProfile `json:"before,omitempty"`
	After  *edi.EDICommunicationProfile `json:"after"`
	Change ChangeKind                   `json:"change"`
}

type PlannedProject struct {
	Project *edi.LadingProject `json:"project,omitempty"`
	Before  *edi.LadingVersion `json:"before,omitempty"`
	Files   []lading.File      `json:"files"`
	Samples []lading.Sample    `json:"samples"`
	Change  ChangeKind         `json:"change"`
}

type PlannedMapping struct {
	Before *edi.EDIMappingProfileItem `json:"before,omitempty"`
	After  *edi.EDIMappingProfileItem `json:"after"`
	Change ChangeKind                 `json:"change"`
}

type PlannedControlNumber struct {
	Before *edi.EDIControlNumberSequence `json:"before,omitempty"`
	After  *edi.EDIControlNumberSequence `json:"after"`
	Change ChangeKind                    `json:"change"`
}

type ConnectionDraftPlan struct {
	Draft          *ConnectionDraft       `json:"draft"`
	Partner        PlannedPartner         `json:"partner"`
	Transport      *PlannedTransport      `json:"transport,omitempty"`
	Project        *PlannedProject        `json:"project,omitempty"`
	Mappings       []PlannedMapping       `json:"mappings"`
	ControlNumbers []PlannedControlNumber `json:"controlNumbers"`
}

type EDIConnectionDraftService interface {
	PlanConnectionDraft(
		ctx context.Context,
		req *ConnectionDraftRequest,
	) (*ConnectionDraft, error)
	PreviewConnectionDraft(
		ctx context.Context,
		req *ConnectionDraftRequest,
	) (*ConnectionDraftPlan, error)
	ApplyConnectionDraft(
		ctx context.Context,
		req *ApplyConnectionDraftRequest,
	) (*AppliedConnectionDraft, error)
}

type FindEDIPartnerMatchesRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	Code         string                `json:"code"`
	Name         string                `json:"name"`
	ISAQualifier string                `json:"isaQualifier"`
	ISAID        string                `json:"isaId"`
	AS2ID        string                `json:"as2Id"`
	CustomerID   pulid.ID              `json:"customerId"`
}

type EDIPartnerMatch struct {
	Partner       *edi.EDIPartner `json:"partner"`
	ByCode        bool            `json:"byCode"`
	ByName        bool            `json:"byName"`
	ByInterchange bool            `json:"byInterchange"`
	ByAS2         bool            `json:"byAs2"`
	ByCustomer    bool            `json:"byCustomer"`
}

type GetEDIPartnerSetupRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	PartnerID  pulid.ID              `json:"partnerId"`
}

type EDIPartnerSetup struct {
	Partner               *edi.EDIPartner                 `json:"partner"`
	CommunicationProfiles []*edi.EDICommunicationProfile  `json:"communicationProfiles"`
	Project               *edi.LadingProject              `json:"project,omitempty"`
	LatestVersion         *edi.LadingVersion              `json:"latestVersion,omitempty"`
	MappingItems          []*edi.EDIMappingProfileItem    `json:"mappingItems"`
	ControlNumbers        []*edi.EDIControlNumberSequence `json:"controlNumbers"`
}

type InterchangeIdentity struct {
	Qualifier     string `json:"qualifier"`
	ID            string `json:"id"`
	ApplicationID string `json:"applicationId"`
	Partners      int    `json:"partners"`
}

type ResolveEDIMappingTargetsRequest struct {
	TenantInfo pagination.TenantInfo                `json:"-"`
	Targets    map[edi.MappingEntityType][]pulid.ID `json:"targets"`
}

type EDIConnectionDraftRepository interface {
	FindPartnerMatches(
		ctx context.Context,
		req *FindEDIPartnerMatchesRequest,
	) ([]*EDIPartnerMatch, error)
	GetPartnerSetup(ctx context.Context, req *GetEDIPartnerSetupRequest) (*EDIPartnerSetup, error)
	GetInterchangeIdentity(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*InterchangeIdentity, error)
	ResolveMappingTargets(
		ctx context.Context,
		req *ResolveEDIMappingTargetsRequest,
	) (map[edi.MappingEntityType]map[pulid.ID]string, error)
}
