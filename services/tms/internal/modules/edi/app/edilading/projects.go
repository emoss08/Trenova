package edilading

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	MaxFileBytes       = 256 << 10
	MaxProjectBytes    = 4 << 20
	MaxSampleBytes     = 512 << 10
	MaxSamplesBytes    = 32 << 20
	MaxPathLength      = 200
	MaxNotesLength     = 2_000
	MaxShadowDays      = 30
	MaxReplayDays      = 90
	maxSettingNameSize = 100
)

type ProjectsParams struct {
	fx.In

	Logger    *zap.Logger
	Runtime   *Runtime
	Projects  ediports.LadingProjectRepository
	Replays   ediports.LadingReplayRepository
	Partners  ediports.EDIPartnerRepository
	Audit     auditports.AuditService
	Workflows kernelports.WorkflowStarter `optional:"true"`
}

type Projects struct {
	l         *zap.Logger
	runtime   *Runtime
	projects  ediports.LadingProjectRepository
	replays   ediports.LadingReplayRepository
	partners  ediports.EDIPartnerRepository
	audit     auditports.AuditService
	workflows kernelports.WorkflowStarter
}

func NewProjects(p ProjectsParams) *Projects { //nolint:gocritic // fx params are passed by value
	return &Projects{
		l:         p.Logger.Named("edi.lading.projects"),
		runtime:   p.Runtime,
		projects:  p.Projects,
		replays:   p.Replays,
		partners:  p.Partners,
		audit:     p.Audit,
		workflows: p.Workflows,
	}
}

type CreateProjectRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	PartnerID   pulid.ID              `json:"partnerId"`
	Role        string                `json:"role"`
	Sets        []string              `json:"sets"`
	Them        *[2]string            `json:"them"`
	Us          *[2]string            `json:"us"`
	Group       *[2]string            `json:"group"`
	Separators  []string              `json:"separators"`
	Usage       string                `json:"usage"`
	Acknowledge string                `json:"acknowledge"`
	AckWithin   int                   `json:"ackWithin"`
}

// CreateProject starts a partner's project from the starter for its role: a draft that checks
// clean and whose tests pass, ready to be adapted to the partner's guide.
func (s *Projects) CreateProject(
	ctx context.Context,
	req *CreateProjectRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingProject, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}
	partner, err := s.partners.GetByID(ctx, ediports.GetEDIPartnerByIDRequest{
		ID:         req.PartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	starter, err := s.runtime.client.Starter(ctx, StarterFor(partner, req))
	if err != nil {
		return nil, err
	}
	draft, samples, err := s.checkedDraft(ctx, req.TenantInfo, starter.Files, starter.Samples)
	if err != nil {
		return nil, err
	}
	draft.OrganizationID = req.TenantInfo.OrgID
	draft.BusinessUnitID = req.TenantInfo.BuID
	draft.Status = edi.LadingVersionStatusDraft
	draft.CreatedByID = req.TenantInfo.UserID
	draft.Notes = "Started from Trenova's " + req.Role + " starter"
	project, err := s.projects.CreateProject(ctx, &ediports.CreateLadingProjectRequest{
		Project: &edi.LadingProject{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			EDIPartnerID:   partner.ID,
			Settings:       map[string]any{},
		},
		Draft:   draft,
		Samples: samples,
	})
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    project,
		Actor:     actor,
		Operation: permission.OpCreate,
		Current:   project,
		Comment:   "Lading project created for " + partner.Code,
	})
	return project, nil
}

// StarterFor is the starter request for a partner.
func StarterFor(partner *edi.EDIPartner, req *CreateProjectRequest) *lading.StarterRequest {
	return &lading.StarterRequest{
		Name:        partner.Code,
		Label:       partner.Name,
		Role:        req.Role,
		Them:        req.Them,
		Us:          req.Us,
		Zone:        partner.Timezone,
		Group:       req.Group,
		Separators:  req.Separators,
		Usage:       req.Usage,
		Acknowledge: req.Acknowledge,
		AckWithin:   req.AckWithin,
		Sets:        req.Sets,
	}
}

func validateCreate(req *CreateProjectRequest) error {
	multiErr := errortypes.NewMultiError()
	if req.PartnerID.IsNil() {
		multiErr.Add("partnerId", errortypes.ErrRequired, "Partner is required")
	}
	if req.Role != lading.RoleCarrier && req.Role != lading.RoleBroker {
		multiErr.Add("role", errortypes.ErrInvalid, "Role must be carrier or broker")
	}
	for index, set := range req.Sets {
		switch set {
		case "204", "990", "214", "210":
		default:
			multiErr.WithIndex("sets", index).Add(
				"",
				errortypes.ErrInvalid,
				"Set {0} is not one Trenova exchanges",
				set,
			)
		}
	}
	switch req.Acknowledge {
	case "", "997", "999", lading.AcknowledgeNone:
	default:
		multiErr.Add(
			"acknowledge",
			errortypes.ErrInvalid,
			"Acknowledgment must be 997, 999 or none",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Projects) GetProject(
	ctx context.Context,
	req *ediports.GetLadingProjectRequest,
) (*edi.LadingProject, error) {
	return s.projects.GetProject(ctx, *req)
}

func (s *Projects) ListProjects(
	ctx context.Context,
	req *ediports.ListLadingProjectsRequest,
) (*pagination.ListResult[*edi.LadingProject], error) {
	return s.projects.ListProjects(ctx, req)
}

func (s *Projects) ListVersions(
	ctx context.Context,
	req ediports.ListLadingVersionsRequest,
) ([]*edi.LadingVersion, error) {
	return s.projects.ListVersions(ctx, req)
}

type VersionDetail struct {
	Version *edi.LadingVersion `json:"version"`
	Samples []lading.Sample    `json:"samples"`
}

// GetVersion is a version with its samples' contents, as the studio opens it.
func (s *Projects) GetVersion(
	ctx context.Context,
	req ediports.GetLadingVersionRequest,
) (*VersionDetail, error) {
	version, err := s.projects.GetVersion(ctx, req)
	if err != nil {
		return nil, err
	}
	samples, err := s.Samples(ctx, req.TenantInfo, version)
	if err != nil {
		return nil, err
	}
	return &VersionDetail{Version: version, Samples: samples}, nil
}

func (s *Projects) Samples(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	version *edi.LadingVersion,
) ([]lading.Sample, error) {
	if len(version.Samples) == 0 {
		return []lading.Sample{}, nil
	}
	digests := make([]string, 0, len(version.Samples))
	seen := map[string]bool{}
	for _, ref := range version.Samples {
		if !seen[ref.Digest] {
			seen[ref.Digest] = true
			digests = append(digests, ref.Digest)
		}
	}
	stored, err := s.projects.GetSamples(ctx, ediports.GetLadingSamplesRequest{
		TenantInfo: tenantInfo,
		ProjectID:  version.ProjectID,
		Digests:    digests,
	})
	if err != nil {
		return nil, err
	}
	contents := make(map[string]string, len(stored))
	for _, sample := range stored {
		contents[sample.Digest] = sample.Content
	}
	samples := make([]lading.Sample, 0, len(version.Samples))
	for _, ref := range version.Samples {
		content, ok := contents[ref.Digest]
		if !ok {
			return nil, fmt.Errorf(
				"lading sample %s of version %s is missing",
				ref.Path,
				version.ID,
			)
		}
		samples = append(samples, lading.Sample{Path: ref.Path, Content: content})
	}
	return samples, nil
}

type SaveDraftRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	VersionID  pulid.ID              `json:"versionId"`
	Files      []lading.File         `json:"files"`
	Samples    []lading.Sample       `json:"samples"`
	Notes      string                `json:"notes"`
	Version    int64                 `json:"version"`
}

// SaveDraft replaces a draft's files and samples and checks them; a draft saves with errors.
func (s *Projects) SaveDraft(
	ctx context.Context,
	req *SaveDraftRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	if err := validateProject(req.Files, req.Samples); err != nil {
		return nil, err
	}
	if len(req.Notes) > MaxNotesLength {
		return nil, errortypes.NewValidationError(
			"notes",
			errortypes.ErrInvalid,
			"Notes cannot be longer than 2,000 characters",
		)
	}
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	if !version.Editable() {
		return nil, errortypes.NewBusinessError(
			"Version {0} is {1}; only a draft can change",
			version.Number,
			string(version.Status),
		)
	}
	previous := *version
	checked, samples, err := s.checkedDraft(ctx, req.TenantInfo, req.Files, req.Samples)
	if err != nil {
		return nil, err
	}
	version.Files = checked.Files
	version.Samples = checked.Samples
	version.Manifest = checked.Manifest
	version.ErrorCount = checked.ErrorCount
	version.WarningCount = checked.WarningCount
	version.Readiness = nil
	version.Notes = req.Notes
	version.Version = req.Version
	saved, err := s.projects.SaveDraft(ctx, version, samples)
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    saved,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Previous:  versionSummary(&previous),
		Current:   versionSummary(saved),
		Comment:   fmt.Sprintf("Lading draft %d saved", saved.Number),
	})
	return saved, nil
}

func versionSummary(version *edi.LadingVersion) map[string]any {
	paths := make([]string, 0, len(version.Files))
	for _, file := range version.Files {
		paths = append(paths, file.Path)
	}
	return map[string]any{
		"id":       version.ID,
		"number":   version.Number,
		"status":   version.Status,
		"files":    paths,
		"samples":  len(version.Samples),
		"errors":   version.ErrorCount,
		"warnings": version.WarningCount,
	}
}

// checkedDraft checks files and samples and returns the version fields and the samples to
// store.
func (s *Projects) checkedDraft(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	files []lading.File,
	samples []lading.Sample,
) (*edi.LadingVersion, []*edi.LadingSample, error) {
	ladingContext, err := s.runtime.Context(ctx, tenantInfo)
	if err != nil {
		return nil, nil, err
	}
	encoded, err := lading.Encode(&lading.Project{Files: files}, ladingContext)
	if err != nil {
		return nil, nil, err
	}
	check, err := s.runtime.client.Check(ctx, encoded)
	if err != nil {
		return nil, nil, err
	}
	refs := make([]edi.LadingSampleRef, 0, len(samples))
	stored := make([]*edi.LadingSample, 0, len(samples))
	seen := map[string]bool{}
	for _, sample := range samples {
		digest := lading.Digest(sample.Content)
		refs = append(refs, edi.LadingSampleRef{
			Path:   sample.Path,
			Digest: digest,
			Size:   len(sample.Content),
		})
		if seen[digest] {
			continue
		}
		seen[digest] = true
		stored = append(stored, &edi.LadingSample{
			Digest:  digest,
			Size:    len(sample.Content),
			Content: sample.Content,
		})
	}
	return &edi.LadingVersion{
		Files:        files,
		Samples:      refs,
		Manifest:     *check,
		ErrorCount:   check.Errors,
		WarningCount: check.Warnings,
	}, stored, nil
}

func validateProject(files []lading.File, samples []lading.Sample) error {
	multiErr := errortypes.NewMultiError()
	if len(files) == 0 {
		multiErr.Add("files", errortypes.ErrRequired, "A project needs at least its partner file")
	}
	if len(files) > lading.MaxFiles {
		multiErr.Add("files", errortypes.ErrInvalid, "A project holds at most 200 files")
	}
	if len(samples) > lading.MaxSamples {
		multiErr.Add("samples", errortypes.ErrInvalid, "A project holds at most 2,000 samples")
	}
	paths := map[string]bool{}
	total := 0
	for index, file := range files {
		field := multiErr.WithIndex("files", index)
		checkPath(field, file.Path, paths, ".lading")
		if len(file.Source) > MaxFileBytes {
			field.Add("source", errortypes.ErrInvalid, "A file holds at most 256 KiB")
		}
		total += len(file.Source)
	}
	if total > MaxProjectBytes {
		multiErr.Add("files", errortypes.ErrInvalid, "A project's files hold at most 4 MiB")
	}
	total = 0
	for index, sample := range samples {
		field := multiErr.WithIndex("samples", index)
		checkPath(field, sample.Path, paths, "")
		if len(sample.Content) > MaxSampleBytes {
			field.Add("content", errortypes.ErrInvalid, "A sample holds at most 512 KiB")
		}
		total += len(sample.Content)
	}
	if total > MaxSamplesBytes {
		multiErr.Add("samples", errortypes.ErrInvalid, "A project's samples hold at most 32 MiB")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func checkPath(multiErr *errortypes.MultiError, name string, seen map[string]bool, suffix string) {
	switch {
	case name == "":
		multiErr.Add("path", errortypes.ErrRequired, "Path is required")
	case len(name) > MaxPathLength:
		multiErr.Add("path", errortypes.ErrInvalid, "A path is at most 200 characters")
	case strings.HasPrefix(name, "/") || strings.Contains(name, "..") ||
		path.Clean(name) != name || strings.ContainsAny(name, "\\\x00"):
		multiErr.Add("path", errortypes.ErrInvalid, "Path {0} is not a relative path", name)
	case suffix != "" && !strings.HasSuffix(name, suffix):
		multiErr.Add("path", errortypes.ErrInvalid, "A source file's path ends in {0}", suffix)
	case seen[name]:
		multiErr.Add("path", errortypes.ErrDuplicate, "Path {0} is used twice", name)
	}
	seen[name] = true
}

type ImportRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	PartnerID  pulid.ID              `json:"partnerId"`
	Files      []lading.File         `json:"files"`
	Samples    []lading.Sample       `json:"samples"`
	Notes      string                `json:"notes"`
}

// Import writes files into a partner's project as a draft: a new project when the partner has
// none, its open draft when it has one, otherwise a new draft from its latest version. The
// draft keeps the samples it had, and the import's samples replace those at the same paths.
func (s *Projects) Import(
	ctx context.Context,
	req *ImportRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	project, err := s.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
		TenantInfo: req.TenantInfo,
		PartnerID:  req.PartnerID,
	})
	if errortypes.IsNotFoundError(err) {
		return s.importProject(ctx, req, actor)
	}
	if err != nil {
		return nil, err
	}
	versions, err := s.projects.ListVersions(ctx, ediports.ListLadingVersionsRequest{
		TenantInfo: req.TenantInfo,
		ProjectID:  project.ID,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("lading project %s has no versions", project.ID)
	}
	draft := versions[0]
	if !draft.Editable() {
		if draft, err = s.NewDraft(ctx, &NewDraftRequest{
			TenantInfo:    req.TenantInfo,
			ProjectID:     project.ID,
			FromVersionID: draft.ID,
		}, actor); err != nil {
			return nil, err
		}
	}
	kept, err := s.Samples(ctx, req.TenantInfo, draft)
	if err != nil {
		return nil, err
	}
	return s.SaveDraft(ctx, &SaveDraftRequest{
		TenantInfo: req.TenantInfo,
		VersionID:  draft.ID,
		Files:      req.Files,
		Samples:    MergeSamples(kept, req.Samples),
		Notes:      req.Notes,
		Version:    draft.Version,
	}, actor)
}

func (s *Projects) importProject(
	ctx context.Context,
	req *ImportRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	if err := validateProject(req.Files, req.Samples); err != nil {
		return nil, err
	}
	partner, err := s.partners.GetByID(ctx, ediports.GetEDIPartnerByIDRequest{
		ID:         req.PartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	draft, samples, err := s.checkedDraft(ctx, req.TenantInfo, req.Files, req.Samples)
	if err != nil {
		return nil, err
	}
	draft.OrganizationID = req.TenantInfo.OrgID
	draft.BusinessUnitID = req.TenantInfo.BuID
	draft.Status = edi.LadingVersionStatusDraft
	draft.CreatedByID = req.TenantInfo.UserID
	draft.Notes = req.Notes
	project, err := s.projects.CreateProject(ctx, &ediports.CreateLadingProjectRequest{
		Project: &edi.LadingProject{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			EDIPartnerID:   partner.ID,
			Settings:       map[string]any{},
		},
		Draft:   draft,
		Samples: samples,
	})
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    project,
		Actor:     actor,
		Operation: permission.OpCreate,
		Current:   project,
		Comment:   "Lading project created for " + partner.Code,
	})
	versions, err := s.projects.ListVersions(ctx, ediports.ListLadingVersionsRequest{
		TenantInfo: req.TenantInfo,
		ProjectID:  project.ID,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("lading project %s was created without its draft", project.ID)
	}
	return versions[0], nil
}

func MergeSamples(kept, added []lading.Sample) []lading.Sample {
	replaced := make(map[string]bool, len(added))
	for _, sample := range added {
		replaced[sample.Path] = true
	}
	merged := make([]lading.Sample, 0, len(kept)+len(added))
	for _, sample := range kept {
		if !replaced[sample.Path] {
			merged = append(merged, sample)
		}
	}
	return append(merged, added...)
}

type NewDraftRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	ProjectID     pulid.ID              `json:"projectId"`
	FromVersionID pulid.ID              `json:"fromVersionId"`
}

// NewDraft starts a draft from a version: its files and samples, checked again.
func (s *Projects) NewDraft(
	ctx context.Context,
	req *NewDraftRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	from, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.FromVersionID,
	})
	if err != nil {
		return nil, err
	}
	if from.ProjectID != req.ProjectID {
		return nil, errortypes.NewValidationError(
			"fromVersionId",
			errortypes.ErrInvalid,
			"The version belongs to another project",
		)
	}
	ladingContext, err := s.runtime.Context(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	encoded, err := lading.Encode(&lading.Project{Files: from.Files}, ladingContext)
	if err != nil {
		return nil, err
	}
	check, err := s.runtime.client.Check(ctx, encoded)
	if err != nil {
		return nil, err
	}
	created, err := s.projects.CreateVersion(ctx, &edi.LadingVersion{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		ProjectID:       req.ProjectID,
		Status:          edi.LadingVersionStatusDraft,
		ParentVersionID: from.ID,
		Files:           from.Files,
		Samples:         from.Samples,
		Manifest:        *check,
		ErrorCount:      check.Errors,
		WarningCount:    check.Warnings,
		CreatedByID:     req.TenantInfo.UserID,
		Notes:           fmt.Sprintf("Started from version %d", from.Number),
	}, nil)
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    created,
		Actor:     actor,
		Operation: permission.OpCreate,
		Current:   versionSummary(created),
		Comment: fmt.Sprintf(
			"Lading draft %d started from version %d",
			created.Number,
			from.Number,
		),
	})
	return created, nil
}

type RunVersionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	VersionID  pulid.ID              `json:"versionId"`
	Map        string                `json:"map"`
	Input      lading.RunInput       `json:"input"`
	Rules      bool                  `json:"rules"`
}

// Run runs one map of a version on an input, its lookups answered from the partner's mappings
// as a live message's are.
func (s *Projects) Run(ctx context.Context, req *RunVersionRequest) (*lading.RunResult, error) {
	if err := validateRunInput(&req.Input); err != nil {
		return nil, err
	}
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	project, err := s.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
		TenantInfo: req.TenantInfo,
		ID:         version.ProjectID,
	})
	if err != nil {
		return nil, err
	}
	program, err := s.runtime.Version(ctx, req.TenantInfo, project, version)
	if err != nil {
		return nil, err
	}
	if program.Manifest.Map(req.Map) == nil {
		return nil, errortypes.NewValidationError(
			"map",
			errortypes.ErrInvalid,
			"Version {0} has no map {1}",
			version.Number,
			req.Map,
		)
	}
	return s.runtime.Run(ctx, &RunRequest{
		TenantInfo: req.TenantInfo,
		PartnerID:  project.EDIPartnerID,
		Program:    program,
		Map:        req.Map,
		Input:      req.Input,
		Rules:      req.Rules,
		Now:        time.Now(),
	})
}

func validateRunInput(input *lading.RunInput) error {
	given := 0
	size := 0
	if input.X12 != nil {
		given++
		size += len(*input.X12)
	}
	if input.CSV != nil {
		given++
		size += len(*input.CSV)
	}
	if len(input.JSON) > 0 {
		given++
		size += len(input.JSON)
	}
	if len(input.Document) > 0 {
		given++
		size += len(input.Document)
	}
	switch {
	case given != 1:
		return errortypes.NewValidationError(
			"input",
			errortypes.ErrInvalid,
			"Give exactly one input: X12, CSV, JSON or a document",
		)
	case size > MaxSampleBytes:
		return errortypes.NewValidationError(
			"input",
			errortypes.ErrInvalid,
			"An input holds at most 512 KiB",
		)
	}
	return nil
}

type VersionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	VersionID  pulid.ID              `json:"versionId"`
}

// Test runs every test and property of a version against its samples and records its
// readiness.
func (s *Projects) Test(ctx context.Context, req *VersionRequest) (*lading.TestReport, error) {
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	report, err := s.test(ctx, req.TenantInfo, version)
	if err != nil {
		return nil, err
	}
	if version.Editable() {
		version.Readiness = readinessOf(report)
		if _, err = s.projects.UpdateVersionStatus(ctx, version); err != nil {
			return nil, err
		}
	}
	return report, nil
}

func (s *Projects) test(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	version *edi.LadingVersion,
) (*lading.TestReport, error) {
	samples, err := s.Samples(ctx, tenantInfo, version)
	if err != nil {
		return nil, err
	}
	ladingContext, err := s.runtime.Context(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	encoded, err := lading.Encode(
		&lading.Project{Files: version.Files, Samples: samples},
		ladingContext,
	)
	if err != nil {
		return nil, err
	}
	return s.runtime.client.Test(ctx, encoded)
}

func readinessOf(report *lading.TestReport) *edi.LadingReadiness {
	return &edi.LadingReadiness{
		Errors:               report.Readiness.Errors,
		Failing:              report.Readiness.Failing,
		Tests:                report.Readiness.Tests,
		UncoveredFlowClauses: report.Readiness.UncoveredFlowClauses,
		Ready:                report.Readiness.Ready,
		CheckedAt:            timeutils.NowUnix(),
	}
}

// Certify freezes a draft once it is ready: no errors, every test passing, every flow arm
// reached by a test, and, when another version is live, a replay against it of every inbound
// map that rejects nothing new.
func (s *Projects) Certify(
	ctx context.Context,
	req *VersionRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	if !version.Editable() {
		return nil, errortypes.NewBusinessError(
			"Version {0} is {1}; only a draft is certified",
			version.Number,
			string(version.Status),
		)
	}
	report, err := s.test(ctx, req.TenantInfo, version)
	if err != nil {
		return nil, err
	}
	readiness := readinessOf(report)
	if !readiness.Ready {
		version.Readiness = readiness
		if _, err = s.projects.UpdateVersionStatus(ctx, version); err != nil {
			return nil, err
		}
		return nil, notReady(readiness)
	}
	project, err := s.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
		TenantInfo: req.TenantInfo,
		ID:         version.ProjectID,
	})
	if err != nil {
		return nil, err
	}
	if project.ActiveVersionID.IsNotNil() {
		if err = s.requireReplays(ctx, req.TenantInfo, version); err != nil {
			return nil, err
		}
	}
	previous := versionSummary(version)
	now := timeutils.NowUnix()
	version.Status = edi.LadingVersionStatusCertified
	version.Readiness = readiness
	version.CertifiedAt = &now
	version.CertifiedByID = req.TenantInfo.UserID
	saved, err := s.projects.UpdateVersionStatus(ctx, version)
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    saved,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Previous:  previous,
		Current:   versionSummary(saved),
		Comment:   fmt.Sprintf("Lading version %d certified", saved.Number),
	})
	return saved, nil
}

func notReady(readiness *edi.LadingReadiness) error {
	multiErr := errortypes.NewMultiError()
	if readiness.Errors > 0 {
		multiErr.Add(
			"files",
			errortypes.ErrInvalid,
			"{0} errors remain in the files",
			readiness.Errors,
		)
	}
	if readiness.Failing > 0 {
		multiErr.Add("tests", errortypes.ErrInvalid, "{0} tests fail", readiness.Failing)
	}
	if readiness.Tests == 0 {
		multiErr.Add("tests", errortypes.ErrRequired, "The project has no tests")
	}
	if readiness.UncoveredFlowClauses > 0 {
		multiErr.Add(
			"tests",
			errortypes.ErrInvalid,
			"{0} flow clauses are not reached by any test",
			readiness.UncoveredFlowClauses,
		)
	}
	return multiErr
}

func (s *Projects) requireReplays(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	version *edi.LadingVersion,
) error {
	replays, err := s.replays.ListReplays(ctx, ediports.ListLadingReplaysRequest{
		TenantInfo: tenantInfo,
		VersionID:  version.ID,
	})
	if err != nil {
		return err
	}
	latest := map[string]*edi.LadingReplay{}
	for _, replay := range replays {
		if replay.Status != edi.LadingReplayStatusCompleted ||
			replay.CreatedAt < version.UpdatedAt {
			continue
		}
		if current, ok := latest[replay.Map]; !ok || replay.CreatedAt > current.CreatedAt {
			latest[replay.Map] = replay
		}
	}
	multiErr := errortypes.NewMultiError()
	for _, item := range inboundMaps(&version.Manifest) {
		replay, ok := latest[item]
		switch {
		case !ok:
			multiErr.Add(
				"replay",
				errortypes.ErrRequired,
				"Replay {0} against the active version before certifying",
				item,
			)
		case replay.Summary != nil && replay.Summary.NewlyRejected > 0:
			multiErr.Add(
				"replay",
				errortypes.ErrInvalid,
				"The replay of {0} rejects {1} messages the active version accepts",
				item,
				replay.Summary.NewlyRejected,
			)
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func inboundMaps(manifest *lading.Check) []string {
	names := make([]string, 0, len(manifest.Maps))
	for index := range manifest.Maps {
		item := &manifest.Maps[index]
		if item.Direction == lading.DirectionInbound && item.Source == "x12" {
			names = append(names, item.Name)
		}
	}
	sort.Strings(names)
	return names
}

// Activate makes a certified version the one Trenova runs; the version it replaces is
// superseded.
func (s *Projects) Activate(
	ctx context.Context,
	req *VersionRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	if version.Status != edi.LadingVersionStatusCertified &&
		version.Status != edi.LadingVersionStatusShadow {
		return nil, errortypes.NewBusinessError(
			"Version {0} is {1}; only a certified version is activated",
			version.Number,
			string(version.Status),
		)
	}
	activated, err := s.projects.ActivateVersion(ctx, ediports.ActivateLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ProjectID:  version.ProjectID,
		VersionID:  version.ID,
		ActorID:    req.TenantInfo.UserID,
	})
	if err != nil {
		return nil, err
	}
	comment := fmt.Sprintf("Lading version %d activated", activated.Version.Number)
	if activated.Superseded != nil {
		comment += fmt.Sprintf(", superseding version %d", activated.Superseded.Number)
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    activated.Version,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Previous:  versionSummary(version),
		Current:   versionSummary(activated.Version),
		Comment:   comment,
	})
	return activated.Version, nil
}

// ActivateStarter certifies and activates the first version of a project Trenova made from a
// starter, whose readiness its own tests prove: internal partners run on it from the start.
func (s *Projects) ActivateStarter(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	project *edi.LadingProject,
) (*edi.LadingVersion, error) {
	versions, err := s.projects.ListVersions(ctx, ediports.ListLadingVersionsRequest{
		TenantInfo: tenantInfo,
		ProjectID:  project.ID,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("lading project %s has no version", project.ID)
	}
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: tenantInfo,
		ID:         versions[0].ID,
	})
	if err != nil {
		return nil, err
	}
	report, err := s.test(ctx, tenantInfo, version)
	if err != nil {
		return nil, err
	}
	readiness := readinessOf(report)
	if !readiness.Ready {
		return nil, notReady(readiness)
	}
	now := timeutils.NowUnix()
	version.Status = edi.LadingVersionStatusCertified
	version.Readiness = readiness
	version.CertifiedAt = &now
	if _, err = s.projects.UpdateVersionStatus(ctx, version); err != nil {
		return nil, err
	}
	activated, err := s.projects.ActivateVersion(ctx, ediports.ActivateLadingVersionRequest{
		TenantInfo: tenantInfo,
		ProjectID:  project.ID,
		VersionID:  version.ID,
	})
	if err != nil {
		return nil, err
	}
	return activated.Version, nil
}

// Archive retires a version that is not running.
func (s *Projects) Archive(
	ctx context.Context,
	req *VersionRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	switch version.Status {
	case edi.LadingVersionStatusActive, edi.LadingVersionStatusShadow:
		return nil, errortypes.NewBusinessError(
			"Version {0} is {1}; stop it before archiving it",
			version.Number,
			string(version.Status),
		)
	case edi.LadingVersionStatusArchived:
		return version, nil
	case edi.LadingVersionStatusDraft,
		edi.LadingVersionStatusCertified,
		edi.LadingVersionStatusSuperseded:
	}
	previous := versionSummary(version)
	now := timeutils.NowUnix()
	version.Status = edi.LadingVersionStatusArchived
	version.ArchivedAt = &now
	saved, err := s.projects.UpdateVersionStatus(ctx, version)
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    saved,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Previous:  previous,
		Current:   versionSummary(saved),
		Comment:   fmt.Sprintf("Lading version %d archived", saved.Number),
	})
	return saved, nil
}

type ShadowRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	VersionID  pulid.ID              `json:"versionId"`
	Days       int                   `json:"days"`
}

// StartShadow runs a certified version beside the active one on live traffic for some days,
// sending nothing, and records where they disagree.
func (s *Projects) StartShadow(
	ctx context.Context,
	req *ShadowRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	if req.Days < 1 || req.Days > MaxShadowDays {
		return nil, errortypes.NewValidationError(
			"days",
			errortypes.ErrInvalid,
			"A shadow runs for 1 to 30 days",
		)
	}
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	if version.Status != edi.LadingVersionStatusCertified {
		return nil, errortypes.NewBusinessError(
			"Version {0} is {1}; only a certified version shadows",
			version.Number,
			string(version.Status),
		)
	}
	project, err := s.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
		TenantInfo: req.TenantInfo,
		ID:         version.ProjectID,
	})
	if err != nil {
		return nil, err
	}
	if project.ActiveVersionID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"The project has no active version to shadow; activate this one instead",
		)
	}
	if project.ShadowVersionID.IsNotNil() && project.ShadowVersionID != version.ID {
		if err = s.stopShadow(ctx, req.TenantInfo, project.ShadowVersionID); err != nil {
			return nil, err
		}
	}
	until := time.Now().AddDate(0, 0, req.Days).Unix()
	version.Status = edi.LadingVersionStatusShadow
	version.ShadowUntil = &until
	saved, err := s.projects.UpdateVersionStatus(ctx, version)
	if err != nil {
		return nil, err
	}
	project.ShadowVersionID = version.ID
	if _, err = s.projects.SetShadowVersion(ctx, project); err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    saved,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Current:   versionSummary(saved),
		Comment:   fmt.Sprintf("Lading version %d shadows for %d days", saved.Number, req.Days),
	})
	return saved, nil
}

// StopShadow ends a shadow run; the version is certified again.
func (s *Projects) StopShadow(
	ctx context.Context,
	req *VersionRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingVersion, error) {
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	if version.Status != edi.LadingVersionStatusShadow {
		return version, nil
	}
	if err = s.stopShadow(ctx, req.TenantInfo, version.ID); err != nil {
		return nil, err
	}
	stopped, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    stopped,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Current:   versionSummary(stopped),
		Comment:   fmt.Sprintf("Lading version %d stopped shadowing", stopped.Number),
	})
	return stopped, nil
}

func (s *Projects) stopShadow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
) error {
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: tenantInfo,
		ID:         versionID,
	})
	if err != nil {
		return err
	}
	project, err := s.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
		TenantInfo: tenantInfo,
		ID:         version.ProjectID,
	})
	if err != nil {
		return err
	}
	if version.Status == edi.LadingVersionStatusShadow {
		version.Status = edi.LadingVersionStatusCertified
		version.ShadowUntil = nil
		if _, err = s.projects.UpdateVersionStatus(ctx, version); err != nil {
			return err
		}
	}
	if project.ShadowVersionID == versionID {
		project.ShadowVersionID = ""
		if _, err = s.projects.SetShadowVersion(ctx, project); err != nil {
			return err
		}
	}
	return nil
}

type UpdateSettingsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ProjectID  pulid.ID              `json:"projectId"`
	Settings   map[string]any        `json:"settings"`
	Version    int64                 `json:"version"`
}

// UpdateSettings sets the values of the partner's settings a planner may change without a new
// version; a value equal to nothing is the partner file's default.
func (s *Projects) UpdateSettings(
	ctx context.Context,
	req *UpdateSettingsRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingProject, error) {
	project, err := s.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ProjectID,
	})
	if err != nil {
		return nil, err
	}
	if project.ActiveVersionID.IsNil() {
		return nil, errortypes.NewBusinessError("Activate a version before changing its settings")
	}
	active, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         project.ActiveVersionID,
	})
	if err != nil {
		return nil, err
	}
	settings, err := ValidateSettings(&active.Manifest, req.Settings)
	if err != nil {
		return nil, err
	}
	previous := project.Settings
	project.Settings = settings
	project.Version = req.Version
	saved, err := s.projects.UpdateProjectSettings(ctx, project)
	if err != nil {
		return nil, err
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    saved,
		Actor:     actor,
		Operation: permission.OpUpdate,
		Previous:  previous,
		Current:   saved.Settings,
		Comment:   "Lading partner settings changed",
	})
	return saved, nil
}

// ValidateSettings keeps the values the partner file declares, each of its setting's type.
func ValidateSettings(manifest *lading.Check, values map[string]any) (map[string]any, error) {
	multiErr := errortypes.NewMultiError()
	settings := make(map[string]any, len(values))
	for name, value := range values {
		if len(name) > maxSettingNameSize {
			multiErr.Add("settings", errortypes.ErrInvalid, "Setting {0} is not declared", name)
			continue
		}
		declared := manifest.Setting(name)
		if declared == nil {
			multiErr.Add(
				"settings."+name,
				errortypes.ErrInvalid,
				"The partner file declares no setting {0}",
				name,
			)
			continue
		}
		if value == nil {
			continue
		}
		if !settingFits(declared.Type, value) {
			multiErr.Add(
				"settings."+name,
				errortypes.ErrInvalid,
				"{0} takes a {1}",
				name,
				declared.Type,
			)
			continue
		}
		settings[name] = value
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return settings, nil
}

func settingFits(kind string, value any) bool {
	switch kind {
	case "Bool":
		_, ok := value.(bool)
		return ok
	case "Int", "Duration":
		number, ok := value.(float64)
		return ok && number == float64(int64(number))
	case "Decimal", "Percent":
		switch value.(type) {
		case float64, string:
			return true
		}
		return false
	case "Text":
		_, ok := value.(string)
		return ok
	case "Money":
		switch typed := value.(type) {
		case float64, string:
			return true
		case map[string]any:
			_, ok := typed["amount"]
			return ok
		}
		return false
	default:
		typed, ok := value.(map[string]any)
		if !ok {
			return false
		}
		_, hasValue := typed["value"]
		_, hasUnit := typed["unit"]
		return hasValue && hasUnit
	}
}

type StartReplayRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	VersionID        pulid.ID              `json:"versionId"`
	AgainstVersionID pulid.ID              `json:"againstVersionId"`
	Map              string                `json:"map"`
	From             int64                 `json:"from"`
	To               int64                 `json:"to"`
}

// StartReplay runs a version and the one it is compared with on the partner's real messages
// from a period, in a workflow.
func (s *Projects) StartReplay(
	ctx context.Context,
	req *StartReplayRequest,
	actor *permissionports.RequestActor,
) (*edi.LadingReplay, error) {
	if req.From <= 0 || req.To <= req.From {
		return nil, errortypes.NewValidationError(
			"to",
			errortypes.ErrInvalid,
			"The period ends after it starts",
		)
	}
	if req.To-req.From > MaxReplayDays*24*60*60 {
		return nil, errortypes.NewValidationError(
			"from",
			errortypes.ErrInvalid,
			"A replay covers at most 90 days",
		)
	}
	version, err := s.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.VersionID,
	})
	if err != nil {
		return nil, err
	}
	if version.Manifest.Map(req.Map) == nil {
		return nil, errortypes.NewValidationError(
			"map",
			errortypes.ErrInvalid,
			"The version has no map {0}",
			req.Map,
		)
	}
	if version.ErrorCount > 0 {
		return nil, errortypes.NewBusinessError("Fix the version's errors before replaying it")
	}
	against := req.AgainstVersionID
	if against.IsNil() {
		project, getErr := s.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
			TenantInfo: req.TenantInfo,
			ID:         version.ProjectID,
		})
		if getErr != nil {
			return nil, getErr
		}
		against = project.ActiveVersionID
	}
	if against.IsNil() {
		return nil, errortypes.NewBusinessError(
			"The project has no active version to replay against",
		)
	}
	if against == version.ID {
		return nil, errortypes.NewValidationError(
			"againstVersionId",
			errortypes.ErrInvalid,
			"A version is not replayed against itself",
		)
	}
	replay, err := s.replays.CreateReplay(ctx, &edi.LadingReplay{
		OrganizationID:   req.TenantInfo.OrgID,
		BusinessUnitID:   req.TenantInfo.BuID,
		ProjectID:        version.ProjectID,
		VersionID:        version.ID,
		AgainstVersionID: against,
		Map:              req.Map,
		PeriodStart:      req.From,
		PeriodEnd:        req.To,
		Status:           edi.LadingReplayStatusRunning,
		CreatedByID:      req.TenantInfo.UserID,
	})
	if err != nil {
		return nil, err
	}
	if s.workflows == nil || !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Replays need the workflow service, which is not configured",
		)
	}
	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       "edi-lading-replay:" + replay.ID.String(),
		TaskQueue:                                temporaltype.EDITaskQueue,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		StaticSummary: fmt.Sprintf(
			"Replay Lading version %d",
			version.Number,
		),
	}, temporaltype.RunLadingReplayWorkflowName, &ReplayWorkflowPayload{
		ReplayID:   replay.ID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, fmt.Errorf("start the Lading replay workflow: %w", err)
	}
	LogAction(s.audit, s.l, &AuditEntry{
		Entity:    replay,
		Actor:     actor,
		Operation: permission.OpCreate,
		Current:   replay,
		Comment:   fmt.Sprintf("Replay of Lading version %d started", version.Number),
	})
	return replay, nil
}

type ReplayWorkflowPayload struct {
	ReplayID   pulid.ID              `json:"replayId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

func (s *Projects) GetReplay(
	ctx context.Context,
	req ediports.GetLadingReplayRequest,
) (*edi.LadingReplay, error) {
	return s.replays.GetReplay(ctx, req)
}

func (s *Projects) ListReplays(
	ctx context.Context,
	req ediports.ListLadingReplaysRequest,
) ([]*edi.LadingReplay, error) {
	return s.replays.ListReplays(ctx, req)
}

type ShadowReport struct {
	Summary *ediports.LadingShadowSummary `json:"summary"`
	Results []*edi.LadingShadowResult     `json:"results"`
}

func (s *Projects) ShadowResults(
	ctx context.Context,
	req *ediports.ListLadingShadowResultsRequest,
) (*ShadowReport, error) {
	summary, err := s.replays.SummarizeShadow(ctx, *req)
	if err != nil {
		return nil, err
	}
	results, err := s.replays.ListShadowResults(ctx, *req)
	if err != nil {
		return nil, err
	}
	return &ShadowReport{Summary: summary, Results: results}, nil
}
