// Package rateimportservice takes a rate sheet from upload to applied.
//
// The sequence is deliberate and cannot be shortened: upload, read, review,
// commit. An import that went straight from a file to a live contract is what
// bulk ingest is most often blamed for — a tariff nobody read replacing one
// somebody negotiated — and the dry run is the whole point of the feature.
package rateimportservice

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	pkgrateimport "github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// maxUploadBytes bounds one sheet.
//
// A four-axis class tariff exported to CSV is a few megabytes. Past this it is
// somebody uploading the wrong file, and reading it would hold the memory to
// find that out.
const maxUploadBytes = 32 << 20

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.RateImportRepository
	AgreementRepo repositories.RateAgreementRepository
	AuditService  services.AuditService
}

type Service struct {
	l             *zap.Logger
	repo          repositories.RateImportRepository
	agreementRepo repositories.RateAgreementRepository
	audit         services.AuditService
	now           func() int64
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.rateimport"),
		repo:          p.Repo,
		agreementRepo: p.AgreementRepo,
		audit:         p.AuditService,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	req *repositories.GetRateImportBatchByIDRequest,
) (*rateimport.RateImportBatch, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRateImportBatchesRequest,
) (*pagination.ListResult[*rateimport.RateImportBatch], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListRows(
	ctx context.Context,
	req *repositories.ListRateImportRowsRequest,
) (*pagination.ListResult[*rateimport.RateImportRow], error) {
	return s.repo.ListRows(ctx, req)
}

// UploadRequest is one rate sheet, and where it should land.
type UploadRequest struct {
	TenantInfo      pagination.TenantInfo
	RateAgreementID pulid.ID
	FileName        string
	Content         []byte
	// EffectiveFrom is the day the imported rules start pricing. It is the
	// caller's decision rather than the sheet's: a tariff is negotiated to take
	// effect on a date, and the day somebody happened to upload it is not it.
	EffectiveFrom int64
}

// Upload reads a sheet and stages what committing it would do.
//
// Nothing about the agreement changes here. The batch that comes back is the
// dry run, and applying it takes a second, deliberate call.
func (s *Service) Upload(
	ctx context.Context,
	req *UploadRequest,
) (*rateimport.RateImportBatch, error) {
	if len(req.Content) == 0 {
		return nil, errortypes.NewValidationError(
			"file", errortypes.ErrRequired, "A file is required",
		)
	}

	if len(req.Content) > maxUploadBytes {
		return nil, errortypes.NewValidationError(
			"file", errortypes.ErrInvalid, "This file is too large to import",
		)
	}

	format, err := formatOf(req.FileName)
	if err != nil {
		return nil, err
	}

	sheet, err := readSheet(format, req.Content)
	if err != nil {
		return nil, errortypes.NewValidationError("file", errortypes.ErrInvalid, err.Error())
	}

	existing, err := s.currentRules(ctx, req)
	if err != nil {
		return nil, err
	}

	staged, err := stage(sheet, existing)
	if err != nil {
		return nil, errortypes.NewValidationError("file", errortypes.ErrInvalid, err.Error())
	}

	batch := s.buildBatch(req, format, staged)

	multiErr := errortypes.NewMultiError()
	batch.Validate(multiErr)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.repo.Create(ctx, batch, s.stampRows(req, staged))
}

func (s *Service) buildBatch(
	req *UploadRequest,
	format rateimport.SourceFormat,
	staged *stagedBatch,
) *rateimport.RateImportBatch {
	summary := staged.Summary

	batch := &rateimport.RateImportBatch{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		RateAgreementID: req.RateAgreementID,
		FileName:        req.FileName,
		SourceFormat:    format,
		Status:          rateimport.StatusParsed,
		EffectiveFrom:   req.EffectiveFrom,
		Mapping:         mappingText(staged.Mapping),
		UnmappedHeaders: staged.UnmappedHeaders,
		Changes:         staged.Changes,
		Summary:         &summary,
		RowCount:        len(staged.Rows),
		ErrorCount:      staged.ErrorCount,
	}

	if !req.TenantInfo.UserID.IsNil() {
		userID := req.TenantInfo.UserID
		batch.UploadedByID = &userID
	}

	return batch
}

func (s *Service) stampRows(
	req *UploadRequest,
	staged *stagedBatch,
) []*rateimport.RateImportRow {
	for _, row := range staged.Rows {
		row.OrganizationID = req.TenantInfo.OrgID
		row.BusinessUnitID = req.TenantInfo.BuID
	}

	return staged.Rows
}

// mappingText renders the mapping for storage, since JSON object keys are
// strings and the field names are what a person reading it needs anyway.
func mappingText(mapping pkgrateimport.Mapping) map[string]int {
	stored := make(map[string]int, len(mapping))

	for field, index := range mapping {
		stored[string(field)] = index
	}

	return stored
}

func formatOf(fileName string) (rateimport.SourceFormat, error) {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".csv":
		return rateimport.SourceFormatCSV, nil
	case ".xlsx", ".xlsm":
		return rateimport.SourceFormatXLSX, nil
	default:
		return "", errortypes.NewValidationError(
			"file",
			errortypes.ErrInvalid,
			"Rate sheets can be imported as CSV or XLSX",
		)
	}
}

func readSheet(
	format rateimport.SourceFormat,
	content []byte,
) (*pkgrateimport.Sheet, error) {
	if format == rateimport.SourceFormatXLSX {
		return pkgrateimport.ReadXLSX(content)
	}

	return pkgrateimport.ReadCSV(content)
}

// currentRules is what the contract says now, which is the other half of the
// diff.
func (s *Service) currentRules(
	ctx context.Context,
	req *UploadRequest,
) ([]*rateagreement.RateAgreementRule, error) {
	agreement, err := s.agreementRepo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: req.RateAgreementID,
		TenantInfo:      req.TenantInfo,
		IncludeChildren: true,
	})
	if err != nil {
		return nil, err
	}

	return agreement.Rules, nil
}

// CommitRequest applies a reviewed import.
type CommitRequest struct {
	TenantInfo        pagination.TenantInfo
	RateImportBatchID pulid.ID
}

// Commit applies a reviewed sheet to its agreement.
//
// It goes through the same close-out-and-insert primitive a general rate
// increase uses, so nothing is mutated: the lanes this replaces keep their
// history and stop pricing from the effective date forward.
//
// Committing twice would amend the agreement twice, which is why the batch's
// own status is what guards it rather than anything in the caller.
func (s *Service) Commit(
	ctx context.Context,
	req *CommitRequest,
) (*rateimport.RateImportBatch, error) {
	batch, err := s.repo.GetByID(ctx, &repositories.GetRateImportBatchByIDRequest{
		RateImportBatchID: req.RateImportBatchID,
		TenantInfo:        req.TenantInfo,
		IncludeRows:       true,
	})
	if err != nil {
		return nil, err
	}

	if !batch.Status.CanCommit() {
		return nil, errortypes.NewBusinessError(
			"This import has already been " + strings.ToLower(batch.Status.String()),
		)
	}

	staged := &stagedBatch{
		Rows:           batch.Rows,
		ErrorCount:     batch.ErrorCount,
		Changes:        batch.Changes,
		existingByLane: make(map[string]*rateagreement.RateAgreementRule),
	}

	existing, err := s.rulesOf(ctx, batch)
	if err != nil {
		return nil, err
	}

	for _, rule := range existing {
		if rule != nil {
			staged.existingByLane[rule.LaneKey] = rule
		}
	}

	amendment := commitPlan(staged)

	if len(amendment.Rules) == 0 && len(amendment.SupersededIDs) == 0 {
		return nil, errortypes.NewBusinessError(errNothingToCommit.Error())
	}

	s.stampRules(batch, amendment.Rules)

	if err = s.agreementRepo.AmendRules(ctx, &repositories.AmendRateAgreementRulesRequest{
		TenantInfo:      req.TenantInfo,
		RateAgreementID: batch.RateAgreementID,
		EffectiveFrom:   batch.EffectiveFrom,
		SupersededIDs:   amendment.SupersededIDs,
		Rules:           amendment.Rules,
	}); err != nil {
		s.l.Error("failed to apply a rate import", zap.Error(err))
		return nil, err
	}

	committed := s.now()
	batch.Status = rateimport.StatusCommitted
	batch.CommittedAt = &committed

	if !req.TenantInfo.UserID.IsNil() {
		userID := req.TenantInfo.UserID
		batch.CommittedBy = &userID
	}

	return s.repo.Update(ctx, batch)
}

// stampRules re-seats on every imported rule the values it inherits from the
// agreement it is landing in. A sheet describes lanes, never their ownership.
func (s *Service) stampRules(
	batch *rateimport.RateImportBatch,
	rules []*rateagreement.RateAgreementRule,
) {
	for _, rule := range rules {
		rule.ID = pulid.Nil
		rule.OrganizationID = batch.OrganizationID
		rule.BusinessUnitID = batch.BusinessUnitID
		rule.RateAgreementID = batch.RateAgreementID
		rule.EffectiveFrom = batch.EffectiveFrom
	}
}

func (s *Service) rulesOf(
	ctx context.Context,
	batch *rateimport.RateImportBatch,
) ([]*rateagreement.RateAgreementRule, error) {
	agreement, err := s.agreementRepo.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: batch.RateAgreementID,
		TenantInfo: pagination.TenantInfo{
			OrgID: batch.OrganizationID,
			BuID:  batch.BusinessUnitID,
		},
		IncludeChildren: true,
	})
	if err != nil {
		return nil, err
	}

	return agreement.Rules, nil
}

// Discard closes an import somebody read and said no to.
func (s *Service) Discard(
	ctx context.Context,
	req *CommitRequest,
) (*rateimport.RateImportBatch, error) {
	batch, err := s.repo.GetByID(ctx, &repositories.GetRateImportBatchByIDRequest{
		RateImportBatchID: req.RateImportBatchID,
		TenantInfo:        req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !batch.Status.CanCommit() {
		return nil, errortypes.NewBusinessError(
			"This import has already been " + strings.ToLower(batch.Status.String()),
		)
	}

	batch.Status = rateimport.StatusDiscarded

	return s.repo.Update(ctx, batch)
}
