package formulalibraryservice

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	tarifferrors "github.com/emoss08/trenova/internal/modules/rating/app/tariff/errors"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	fieldSource       = "source"
	maxBulkLibraryIDs = 100
)

type sourceChecker interface {
	CheckSource(ctx context.Context, req *tariff.SourceCheckRequest) (*tariff.SourceCheck, error)
	TestChecked(
		ctx context.Context,
		checked *tariff.SourceCheck,
		source string,
	) (*rules.TariffTestRun, error)
}

type usageLister interface {
	ListLibraryUsages(
		ctx context.Context,
		req *formulatemplateports.ListLibraryUsagesRequest,
	) ([]*formulatemplateports.LibraryUsage, error)
}

type Params struct {
	fx.In

	Logger           *zap.Logger
	DB               kernelports.Transactor
	Repo             formulalibraryports.FormulaLibraryRepository
	VersionRepo      formulalibraryports.FormulaLibraryVersionRepository
	ReviewRepo       formulalibraryports.FormulaLibraryReviewRepository
	TemplateVersions formulatemplateports.FormulaTemplateVersionRepository
	Checker          *tariff.Checker
	AuditService     auditports.AuditService
	Notifications    *notificationservice.Service   `optional:"true"`
	Roles            permissionports.RoleRepository `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	repo          formulalibraryports.FormulaLibraryRepository
	versionRepo   formulalibraryports.FormulaLibraryVersionRepository
	reviewRepo    formulalibraryports.FormulaLibraryReviewRepository
	usages        usageLister
	checker       sourceChecker
	auditService  auditports.AuditService
	notifications *notificationservice.Service
	roles         permissionports.RoleRepository
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:             p.Logger.Named("service.formulalibrary"),
		db:            p.DB,
		repo:          p.Repo,
		versionRepo:   p.VersionRepo,
		reviewRepo:    p.ReviewRepo,
		usages:        p.TemplateVersions,
		checker:       p.Checker,
		auditService:  p.AuditService,
		notifications: p.Notifications,
		roles:         p.Roles,
	}
}

type contentCheck struct {
	checked *tariff.SourceCheck
	tests   *rules.TariffTestRun
}

func (c *contentCheck) checks() bool {
	return c.checked != nil && c.checked.OK()
}

func (c *contentCheck) refusal() error {
	return tarifferrors.CheckRefusal(c.checked.Check, tarifferrors.CheckFields{Source: fieldSource})
}

func (c *contentCheck) testCount() int32 {
	return intutils.SafeToInt32(len(c.tests.Tests))
}

func (c *contentCheck) passed() int {
	passed := 0
	for i := range c.tests.Tests {
		if c.tests.Tests[i].Status == rules.TariffTestPassed {
			passed++
		}
	}
	return passed
}

func (c *contentCheck) failing() []string {
	names := make([]string, 0, len(c.tests.Tests))
	for i := range c.tests.Tests {
		if c.tests.Tests[i].Status != rules.TariffTestPassed {
			names = append(names, c.tests.Tests[i].Name)
		}
	}
	return names
}

func libraryTenant(library *formulalibrary.FormulaLibrary) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: library.OrganizationID, BuID: library.BusinessUnitID}
}

func (s *Service) checkContent(
	ctx context.Context,
	library *formulalibrary.FormulaLibrary,
) (*contentCheck, error) {
	return s.checkSource(ctx, libraryTenant(library), library.Source)
}

func (s *Service) checkSource(
	ctx context.Context,
	tenant pagination.TenantInfo,
	source string,
) (*contentCheck, error) {
	checked, err := s.checker.CheckSource(ctx, &tariff.SourceCheckRequest{
		TenantInfo: tenant,
		Source:     source,
		Kind:       rules.TariffKindLibrary,
		Mode:       tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}
	tests, err := s.checker.TestChecked(ctx, checked, source)
	if err != nil {
		return nil, err
	}
	return &contentCheck{checked: checked, tests: tests}, nil
}

func (s *Service) prepareContent(
	ctx context.Context,
	library *formulalibrary.FormulaLibrary,
) error {
	multiErr := errortypes.NewMultiError()
	library.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	content, err := s.checkContent(ctx, library)
	if err != nil {
		return err
	}
	if refusal := content.refusal(); refusal != nil {
		return refusal
	}
	if err = headerMatches(library, content.checked.Check.Header); err != nil {
		return err
	}
	applyContent(library, content)
	return nil
}

func headerMatches(library *formulalibrary.FormulaLibrary, header *rules.TariffHeader) error {
	if header != nil && header.Name == library.Path() {
		return nil
	}
	named := ""
	if header != nil {
		named = header.Name
	}
	return errortypes.NewValidationError(
		fieldSource,
		errortypes.ErrInvalid,
		"The library is saved as {0}, so its first line must be library \"{0}\", not \"{1}\"",
		library.Path(),
		named,
	)
}

func applyContent(library *formulalibrary.FormulaLibrary, content *contentCheck) {
	imports := content.checked.Imports
	kept := imports[:0:0]
	for _, imp := range imports {
		if imp.LibraryID != library.ID {
			kept = append(kept, imp)
		}
	}
	library.Imports = kept
	library.TestCount = content.testCount()
}

func clearApprovalFields(library *formulalibrary.FormulaLibrary) {
	library.SubmittedByID = nil
	library.SubmittedAt = nil
	library.ApprovedByID = nil
	library.ApprovedAt = nil
	library.ReviewComment = ""
}

func withContent(original, payload *formulalibrary.FormulaLibrary) *formulalibrary.FormulaLibrary {
	next := *original
	next.Name = payload.Name
	next.Description = payload.Description
	next.Source = payload.Source
	next.Version = payload.Version
	return &next
}

func (s *Service) Create(
	ctx context.Context,
	entity *formulalibrary.FormulaLibrary,
	userID pulid.ID,
) (*formulalibrary.FormulaLibrary, error) {
	log := s.l.With(zap.String("operation", "Create"), zap.String("name", entity.Name))

	entity.Status = formulatemplate.StatusDraft
	entity.SourceLibraryID = nil
	entity.SourceVersionNumber = nil
	clearApprovalFields(entity)
	if err := s.prepareContent(ctx, entity); err != nil {
		return nil, err
	}
	entity.CurrentVersionNumber = 1

	created, err := s.createWithSnapshot(ctx, entity, userID, "Initial version")
	if err != nil {
		log.Error("failed to create formula library", zap.Error(err))
		return nil, err
	}

	s.logAuditAction(ctx, created, permission.OpCreate, userID, nil, "Formula library created")
	return created, nil
}

func (s *Service) createWithSnapshot(
	ctx context.Context,
	entity *formulalibrary.FormulaLibrary,
	userID pulid.ID,
	changeMessage string,
) (*formulalibrary.FormulaLibrary, error) {
	var created *formulalibrary.FormulaLibrary
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		saved, txErr := s.repo.Create(txCtx, entity)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.versionRepo.Create(txCtx, formulalibrary.NewVersionFromLibrary(
			saved, 1, userID, changeMessage, nil,
		)); txErr != nil {
			return txErr
		}
		created = saved
		return nil
	})
	return created, err
}

func (s *Service) Update(
	ctx context.Context,
	payload *formulalibrary.FormulaLibrary,
	userID pulid.ID,
) (*formulalibrary.FormulaLibrary, error) {
	log := s.l.With(zap.String("operation", "Update"), zap.String("id", payload.ID.String()))

	original, err := s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
		TenantInfo: libraryTenant(payload),
		LibraryID:  payload.ID,
	})
	if err != nil {
		return nil, err
	}

	entity := withContent(original, payload)
	if err = s.refuseRename(ctx, entity, original); err != nil {
		return nil, err
	}
	if err = s.prepareContent(ctx, entity); err != nil {
		return nil, err
	}

	material := entity.HasMaterialChange(original)
	comment := "Formula library updated"
	revertsApproval := material &&
		(original.Status == formulatemplate.StatusActive ||
			original.Status == formulatemplate.StatusInReview)
	if revertsApproval {
		entity.Status = formulatemplate.StatusDraft
		clearApprovalFields(entity)
		comment = "Material change reverted approval"
	}

	entity.CurrentVersionNumber = original.CurrentVersionNumber
	var changeSummary map[string]jsonutils.FieldChange
	if material {
		entity.CurrentVersionNumber = original.CurrentVersionNumber + 1
		changeSummary, err = jsonutils.JSONDiff(original, entity, nil)
		if err != nil {
			log.Warn("failed to compute change summary for version snapshot", zap.Error(err))
		}
	}

	var updated *formulalibrary.FormulaLibrary
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		saved, txErr := s.repo.Update(txCtx, entity)
		if txErr != nil {
			return txErr
		}
		if material {
			if _, txErr = s.versionRepo.Create(txCtx, formulalibrary.NewVersionFromLibrary(
				saved, saved.CurrentVersionNumber, userID, "", changeSummary,
			)); txErr != nil {
				return txErr
			}
		}
		updated = saved
		return nil
	})
	if err != nil {
		log.Error("failed to update formula library", zap.Error(err))
		return nil, err
	}

	opts := make([]auditports.LogOption, 0, 1)
	if material {
		opts = append(opts, auditservice.WithCritical())
	}
	s.logAuditAction(ctx, updated, permission.OpUpdate, userID, original, comment, opts...)
	return updated, nil
}

func (s *Service) refuseRename(
	ctx context.Context,
	entity, original *formulalibrary.FormulaLibrary,
) error {
	if entity.Name == original.Name {
		return nil
	}
	approved, err := s.versionRepo.GetLatestByStatus(
		ctx,
		&formulalibraryports.GetLatestLibraryVersionByStatusRequest{
			TenantInfo: libraryTenant(original),
			LibraryID:  original.ID,
			Status:     formulatemplate.StatusActive,
		},
	)
	if err != nil {
		return err
	}
	if approved == nil {
		return nil
	}
	return errortypes.NewValidationError(
		"name",
		errortypes.ErrInvalid,
		"{0} has an approved version, and templates import it as {1}; a library cannot be renamed once it has been approved",
		original.Name,
		original.Path(),
	)
}

func (s *Service) GetByID(
	ctx context.Context,
	req *formulalibraryports.GetLibraryByIDRequest,
) (*formulalibrary.FormulaLibrary, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req *formulalibraryports.GetLibrariesByIDsRequest,
) ([]*formulalibrary.FormulaLibrary, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *formulalibraryports.ListLibraryConnectionRequest,
) (*pagination.CursorListResult[*formulalibrary.FormulaLibrary], error) {
	return s.repo.ListConnection(ctx, req)
}

type DuplicateRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	LibraryIDs []pulid.ID            `json:"libraryIds"`
}

func (s *Service) Duplicate(
	ctx context.Context,
	req *DuplicateRequest,
) ([]*formulalibrary.FormulaLibrary, error) {
	if len(req.LibraryIDs) == 0 || len(req.LibraryIDs) > maxBulkLibraryIDs {
		return nil, errortypes.NewValidationError(
			"libraryIds",
			errortypes.ErrInvalid,
			"Choose between 1 and {0} libraries to duplicate",
			maxBulkLibraryIDs,
		)
	}

	sources, err := s.repo.GetByIDs(ctx, &formulalibraryports.GetLibrariesByIDsRequest{
		TenantInfo: req.TenantInfo,
		LibraryIDs: req.LibraryIDs,
	})
	if err != nil {
		return nil, err
	}

	pending := make([]*formulalibrary.FormulaLibrary, 0, len(sources))
	taken := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		existing, nameErr := s.repo.NamesLike(ctx, req.TenantInfo, source.Name+"Copy")
		if nameErr != nil {
			return nil, nameErr
		}
		for _, name := range existing {
			taken[strings.ToLower(name)] = struct{}{}
		}
		name := nextCopyName(taken, source.Name)
		taken[strings.ToLower(name)] = struct{}{}

		sourceID := source.ID
		sourceVersion := source.CurrentVersionNumber
		copied := &formulalibrary.FormulaLibrary{
			OrganizationID:       req.TenantInfo.OrgID,
			BusinessUnitID:       req.TenantInfo.BuID,
			Name:                 name,
			Description:          source.Description,
			Source:               renameHeader(source.Source, formulalibrary.Path(name)),
			Imports:              source.Imports,
			TestCount:            source.TestCount,
			Status:               formulatemplate.StatusDraft,
			SourceLibraryID:      &sourceID,
			SourceVersionNumber:  &sourceVersion,
			CurrentVersionNumber: 1,
		}
		multiErr := errortypes.NewMultiError()
		copied.Validate(multiErr)
		if multiErr.HasErrors() {
			return nil, multiErr
		}
		pending = append(pending, copied)
	}

	created := make([]*formulalibrary.FormulaLibrary, 0, len(pending))
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		for _, entity := range pending {
			saved, txErr := s.repo.Create(txCtx, entity)
			if txErr != nil {
				return txErr
			}
			if _, txErr = s.versionRepo.Create(txCtx, formulalibrary.NewVersionFromLibrary(
				saved, 1, req.TenantInfo.UserID, "Duplicated", nil,
			)); txErr != nil {
				return txErr
			}
			created = append(created, saved)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, entity := range created {
		s.logAuditAction(ctx, entity, permission.OpCreate, req.TenantInfo.UserID, nil,
			"Formula library duplicated")
	}
	return created, nil
}

var libraryHeader = regexp.MustCompile(`(?m)^([ \t]*library[ \t]+)"[^"\n]*"`)

func renameHeader(source, path string) string {
	located := libraryHeader.FindStringSubmatchIndex(source)
	if located == nil {
		return source
	}
	return source[:located[3]] + strconv.Quote(path) + source[located[1]:]
}

func nextCopyName(taken map[string]struct{}, base string) string {
	candidate := base + "Copy"
	if len(candidate) > formulalibrary.MaxNameLength {
		candidate = candidate[:formulalibrary.MaxNameLength]
	}
	if _, exists := taken[strings.ToLower(candidate)]; !exists {
		return candidate
	}
	for suffix := 2; ; suffix++ {
		next := fmt.Sprintf("%sCopy%d", base, suffix)
		if _, exists := taken[strings.ToLower(next)]; !exists {
			return next
		}
	}
}

type ArchiveRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	LibraryID    pulid.ID              `json:"-"`
	ConfirmInUse bool                  `json:"confirmInUse"`
}

func (s *Service) Archive(
	ctx context.Context,
	req *ArchiveRequest,
) (*formulalibrary.FormulaLibrary, error) {
	library, err := s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
	})
	if err != nil {
		return nil, err
	}
	if library.Status != formulatemplate.StatusActive {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only an active library can be archived; this one is {0}",
			library.Status,
		)
	}

	usages, err := s.usages.ListLibraryUsages(ctx, &formulatemplateports.ListLibraryUsagesRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}
	if len(usages) > 0 && !req.ConfirmInUse {
		return nil, errortypes.NewValidationError(
			"confirmInUse",
			errortypes.ErrInvalid,
			"Templates import {0}. Versions already approved keep the version they pinned, but no new template version can import it. Confirm to archive it anyway",
			library.Path(),
		)
	}

	original := *library
	library.Status = formulatemplate.StatusInactive
	updated, err := s.repo.Update(ctx, library)
	if err != nil {
		return nil, err
	}

	s.logAuditAction(ctx, updated, permission.OpArchive, req.TenantInfo.UserID, &original,
		"Formula library archived", auditservice.WithCritical())
	return updated, nil
}

type ListUsagesRequest struct {
	TenantInfo    pagination.TenantInfo
	LibraryID     pulid.ID
	VersionNumber *int64
	Limit         int
}

func (s *Service) ListUsages(
	ctx context.Context,
	req *ListUsagesRequest,
) ([]*formulatemplateports.LibraryUsage, error) {
	if _, err := s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
	}); err != nil {
		return nil, err
	}
	usages, err := s.usages.ListLibraryUsages(ctx, &formulatemplateports.ListLibraryUsagesRequest{
		TenantInfo:    req.TenantInfo,
		LibraryID:     req.LibraryID,
		VersionNumber: req.VersionNumber,
		Limit:         req.Limit,
	})
	if err != nil {
		return nil, err
	}
	if usages == nil {
		usages = []*formulatemplateports.LibraryUsage{}
	}
	return usages, nil
}

func (s *Service) logAuditAction(
	ctx context.Context,
	entity *formulalibrary.FormulaLibrary,
	operation permission.Operation,
	userID pulid.ID,
	previous *formulalibrary.FormulaLibrary,
	comment string,
	extra ...auditports.LogOption,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceFormulaLibrary,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(entity),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}
	opts := []auditports.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithRequest(ctx),
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		opts = append(opts, auditservice.WithDiff(previous, entity))
	}
	opts = append(opts, extra...)

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log audit action",
			zap.String("libraryID", entity.ID.String()),
			zap.Error(err),
		)
	}
}
