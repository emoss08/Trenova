package fuelpurchaseservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const maxImportBytes = 32 << 20

func batchTenant(batch *fuelpurchase.ImportBatch) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
}

func (s *Service) GetImport(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*fuelpurchase.ImportBatch, error) {
	return s.repo.GetImportBatchByID(ctx, &repositories.GetImportBatchByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) ListImportRows(
	ctx context.Context,
	req *repositories.ListImportRowsRequest,
) (*pagination.CursorListResult[*fuelpurchase.ImportRow], error) {
	return s.repo.ListImportRows(ctx, req)
}

func (s *Service) Template(provider fuelpurchase.CardProvider) (fileName, content string) {
	if !provider.IsValid() {
		provider = fuelpurchase.CardProviderOther
	}
	return fuelimport.TemplateFileName(provider), fuelimport.Template(provider)
}

type CreateImportRequest struct {
	TenantInfo        pagination.TenantInfo
	Provider          fuelpurchase.CardProvider
	DefaultFuelType   domaintypes.IFTAFuelType
	DefaultFuelCardID pulid.ID
	DefaultCurrency   string
	Mapping           map[string]int
	UserID            pulid.ID
}

func (s *Service) CreateImport(
	ctx context.Context,
	req *CreateImportRequest,
) (*fuelpurchase.ImportBatch, error) {
	if _, err := fuelimport.MappingFromStrings(req.Mapping); err != nil {
		return nil, errortypes.NewValidationError("mapping", errortypes.ErrInvalid, err.Error())
	}

	batch := &fuelpurchase.ImportBatch{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		Provider:        req.Provider,
		Status:          fuelpurchase.ImportStatusPending,
		DefaultFuelType: req.DefaultFuelType,
		DefaultCurrency: req.DefaultCurrency,
		Mapping:         req.Mapping,
		UploadedByID:    req.UserID,
	}
	if !req.DefaultFuelCardID.IsNil() {
		if _, err := s.repo.GetCardByID(ctx, &repositories.GetFuelCardByIDRequest{
			ID:         req.DefaultFuelCardID,
			TenantInfo: req.TenantInfo,
		}); err != nil {
			return nil, referenceError(
				err,
				"defaultFuelCardId",
				"Default fuel card does not exist in your organization",
			)
		}
		cardID := req.DefaultFuelCardID
		batch.DefaultFuelCardID = &cardID
	}

	batch.Normalize()
	if err := validateEntity(batch); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateImportBatch(ctx, batch)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchaseImport,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    created,
		comment:    "Started a " + created.Provider.Label() + " fuel card import",
	})
	s.publish(ctx, req.TenantInfo, realtimeImport, permission.OpCreate, created.ID, req.UserID)

	return created, nil
}

type StageRequest struct {
	TenantInfo pagination.TenantInfo
	BatchID    pulid.ID
	DocumentID pulid.ID
	Mapping    map[string]int
	UserID     pulid.ID
}

func (s *Service) Stage(
	ctx context.Context,
	req *StageRequest,
) (*fuelpurchase.ImportBatch, error) {
	log := s.l.With(
		zap.String("operation", "Stage"),
		zap.String("batchId", req.BatchID.String()),
	)

	batch, err := s.repo.GetImportBatchByID(ctx, &repositories.GetImportBatchByIDRequest{
		ID:         req.BatchID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !batch.CanStage() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This import is {0} and cannot be staged again", strings.ToLower(batch.Status.Label()),
		)
	}
	if req.DocumentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrRequired,
			"Upload the statement before staging the import",
		)
	}

	doc, err := s.documents.Get(ctx, repositories.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, referenceError(err, "documentId", "The uploaded file could not be found")
	}
	if doc.ResourceType != ImportDocumentResourceType || doc.ResourceID != batch.ID.String() {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"This file was not uploaded for this import",
		)
	}
	if doc.FileSize > maxImportBytes {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"This file is too large to import; the limit is 32 MiB",
		)
	}

	fileName := doc.OriginalName
	if strings.TrimSpace(fileName) == "" {
		fileName = doc.FileName
	}
	format, err := formatOf(fileName)
	if err != nil {
		return nil, err
	}

	storedMapping := batch.Mapping
	if batch.StagedAt != nil {
		storedMapping = nil
	}
	mapping, err := s.stageMapping(req.Mapping, storedMapping)
	if err != nil {
		return nil, err
	}

	content, err := s.readDocument(ctx, req.TenantInfo, req.DocumentID)
	if err != nil {
		return nil, err
	}

	documentID := req.DocumentID
	batch.DocumentID = &documentID
	batch.FileName = fileName
	batch.SourceFormat = format
	if batch.UploadedByID.IsNil() {
		batch.UploadedByID = req.UserID
	}
	if len(mapping) > 0 {
		batch.Mapping = mapping.Strings()
	}

	sheet, err := readSheet(format, content)
	if err != nil {
		return s.failStage(ctx, batch, sheetProblem(err), req.UserID)
	}

	staged := fuelimport.Stage(sheet, fuelimport.StageOptions{
		Provider:        batch.Provider,
		Mapping:         mapping,
		DefaultFuelType: batch.DefaultFuelType,
		DefaultCurrency: batch.DefaultCurrency,
	})
	batch.Mapping = staged.Mapping.Strings()
	batch.UnmappedHeaders = staged.Unmapped
	if staged.HasProblems() {
		messages := make([]string, 0, len(staged.Problems))
		for _, problem := range staged.Problems {
			messages = append(messages, problem.Message)
		}
		return s.failStage(ctx, batch, strings.Join(messages, "; "), req.UserID)
	}

	rows, summary, err := s.resolveRows(ctx, batch, staged)
	if err != nil {
		log.Error("failed to resolve staged rows", zap.Error(err))
		return nil, err
	}

	if err = s.repo.ReplaceImportRows(ctx, batch, rows); err != nil {
		return nil, err
	}

	stagedAt := s.now()
	batch.Status = fuelpurchase.ImportStatusParsed
	batch.Summary = summary
	batch.RowCount = summary.RowCount
	batch.ErrorCount = summary.ErrorCount
	batch.Error = ""
	batch.StagedAt = &stagedAt
	if err = validateEntity(batch); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateImportBatch(ctx, batch)
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchaseImport,
		resourceID: updated.ID.String(),
		operation:  permission.OpImport,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		comment: "Staged " + fileName + ": " + strconv.Itoa(summary.NewCount) + " new, " +
			strconv.Itoa(summary.DuplicateInFileCount+summary.AlreadyImportedCount) +
			" duplicate, " + strconv.Itoa(summary.ErrorCount) + " with errors",
	})
	s.publish(ctx, req.TenantInfo, realtimeImport, permission.OpImport, updated.ID, req.UserID)

	return updated, nil
}

func (s *Service) stageMapping(
	override map[string]int,
	stored map[string]int,
) (fuelimport.Mapping, error) {
	source := override
	field := "mapping"
	if len(source) == 0 {
		source = stored
	}
	mapping, err := fuelimport.MappingFromStrings(source)
	if err != nil {
		return nil, errortypes.NewValidationError(field, errortypes.ErrInvalid, err.Error())
	}
	return mapping, nil
}

func (s *Service) readDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentID pulid.ID,
) ([]byte, error) {
	content, err := s.documents.GetDownloadContent(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if content == nil || content.Body == nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"The uploaded file is empty",
		)
	}
	defer func() { _ = content.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(content.Body, maxImportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read import document: %w", err)
	}
	if len(data) > maxImportBytes {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"This file is too large to import; the limit is 32 MiB",
		)
	}
	if len(data) == 0 {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"The uploaded file is empty",
		)
	}

	return data, nil
}

func (s *Service) failStage(
	ctx context.Context,
	batch *fuelpurchase.ImportBatch,
	message string,
	userID pulid.ID,
) (*fuelpurchase.ImportBatch, error) {
	if err := s.repo.ReplaceImportRows(ctx, batch, nil); err != nil {
		return nil, err
	}

	stagedAt := s.now()
	batch.Status = fuelpurchase.ImportStatusFailed
	batch.Error = message
	batch.Summary = nil
	batch.RowCount = 0
	batch.ErrorCount = 0
	batch.StagedAt = &stagedAt

	updated, err := s.repo.UpdateImportBatch(ctx, batch)
	if err != nil {
		return nil, err
	}

	tenantInfo := batchTenant(updated)
	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchaseImport,
		resourceID: updated.ID.String(),
		operation:  permission.OpImport,
		userID:     userID,
		tenant:     tenantInfo,
		current:    updated,
		comment:    "Staging failed: " + message,
	})
	s.publish(ctx, tenantInfo, realtimeImport, permission.OpImport, updated.ID, userID)

	return updated, nil
}

func sheetProblem(err error) string {
	switch {
	case errors.Is(err, rateimport.ErrNoRows):
		return "This file has a header row and no transactions under it"
	case errors.Is(err, rateimport.ErrNoTable):
		return "Nothing in this file looks like a table of transactions"
	default:
		return err.Error()
	}
}

type stageContext struct {
	batch          *fuelpurchase.ImportBatch
	existing       map[string]pulid.ID
	tractorByCode  map[string]*tractor.Tractor
	tractorByPlate map[string]*tractor.Tractor
	cardByLastFour map[string]*fuelpurchase.FuelCard
	defaultCard    *fuelpurchase.FuelCard
	jurisdictions  map[string]*ifta.Jurisdiction
}

func upperKey(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func (s *Service) resolveRows(
	ctx context.Context,
	batch *fuelpurchase.ImportBatch,
	staged *fuelimport.StageResult,
) ([]*fuelpurchase.ImportRow, *fuelpurchase.ImportSummary, error) {
	sc, err := s.loadStageContext(ctx, batch, staged)
	if err != nil {
		return nil, nil, err
	}

	rows := make([]*fuelpurchase.ImportRow, 0, len(staged.Rows))
	summary := &fuelpurchase.ImportSummary{
		ByFuelType:     map[string]string{},
		ByJurisdiction: map[string]string{},
	}
	totalGallons := decimal.Zero
	gallonsByFuel := make(map[string]decimal.Decimal, 4)
	gallonsByJurisdiction := make(map[string]decimal.Decimal, 16)

	for _, stagedRow := range staged.Rows {
		row := s.buildImportRow(sc, stagedRow)
		rows = append(rows, row)
		summary.RowCount++

		switch row.Status {
		case fuelpurchase.ImportRowStatusError:
			summary.ErrorCount++
		case fuelpurchase.ImportRowStatusDuplicateInFile:
			summary.DuplicateInFileCount++
		case fuelpurchase.ImportRowStatusAlreadyImported:
			summary.AlreadyImportedCount++
		case fuelpurchase.ImportRowStatusNew:
			summary.NewCount++
			purchase := row.Parsed
			totalGallons = totalGallons.Add(purchase.Gallons)
			summary.TotalAmountMinor += purchase.TotalAmountMinor
			fuelKey := string(purchase.FuelType)
			gallonsByFuel[fuelKey] = gallonsByFuel[fuelKey].Add(purchase.Gallons)
			jurisdictionKey := stagedRow.Parsed.JurisdictionKey()
			gallonsByJurisdiction[jurisdictionKey] = gallonsByJurisdiction[jurisdictionKey].
				Add(purchase.Gallons)
			if summary.EarliestPurchasedAt == 0 ||
				purchase.PurchasedAt < summary.EarliestPurchasedAt {
				summary.EarliestPurchasedAt = purchase.PurchasedAt
			}
			if purchase.PurchasedAt > summary.LatestPurchasedAt {
				summary.LatestPurchasedAt = purchase.PurchasedAt
			}
		case fuelpurchase.ImportRowStatusCommitted, fuelpurchase.ImportRowStatusSkipped:
		}
	}

	summary.TotalGallons = totalGallons.StringFixed(3)
	for key, gallons := range gallonsByFuel {
		summary.ByFuelType[key] = gallons.StringFixed(3)
	}
	for key, gallons := range gallonsByJurisdiction {
		summary.ByJurisdiction[key] = gallons.StringFixed(3)
	}

	return rows, summary, nil
}

func (s *Service) loadStageContext(
	ctx context.Context,
	batch *fuelpurchase.ImportBatch,
	staged *fuelimport.StageResult,
) (*stageContext, error) {
	tenantInfo := batchTenant(batch)
	sc := &stageContext{
		batch:          batch,
		existing:       map[string]pulid.ID{},
		tractorByCode:  map[string]*tractor.Tractor{},
		tractorByPlate: map[string]*tractor.Tractor{},
		cardByLastFour: map[string]*fuelpurchase.FuelCard{},
		jurisdictions:  map[string]*ifta.Jurisdiction{},
	}

	references := make([]string, 0, len(staged.Rows))
	codes := make([]string, 0, len(staged.Rows))
	plates := make([]string, 0, len(staged.Rows))
	lastFours := make(map[string]struct{}, 16)
	jurisdictionKeys := make(map[string]struct{}, 16)

	for _, row := range staged.Rows {
		if row.Parsed == nil || row.IsDuplicate() {
			continue
		}
		references = append(references, row.Parsed.Reference)
		if row.Parsed.TractorCode != "" {
			codes = append(codes, row.Parsed.TractorCode)
		}
		if row.Parsed.LicensePlate != "" {
			plates = append(plates, row.Parsed.LicensePlate)
		}
		if row.Parsed.CardLastFour != "" {
			lastFours[row.Parsed.CardLastFour] = struct{}{}
		}
		jurisdictionKeys[row.Parsed.JurisdictionKey()] = struct{}{}
	}

	if len(references) > 0 {
		existing, err := s.repo.FindReferences(ctx, &repositories.FindFuelPurchaseReferencesRequest{
			TenantInfo: tenantInfo,
			References: references,
		})
		if err != nil {
			return nil, err
		}
		sc.existing = existing
	}

	if len(codes) > 0 || len(plates) > 0 {
		byCode, err := s.tractorRepo.GetByCodes(ctx, repositories.GetTractorsByCodesRequest{
			TenantInfo:    tenantInfo,
			Codes:         codes,
			LicensePlates: plates,
		})
		if err != nil {
			return nil, err
		}
		for code, entity := range byCode {
			sc.tractorByCode[upperKey(code)] = entity
			if plate := upperKey(entity.LicensePlateNumber); plate != "" {
				sc.tractorByPlate[plate] = entity
			}
		}
	}

	for lastFour := range lastFours {
		card, err := s.repo.FindCardByLastFour(ctx, &repositories.FindFuelCardByLastFourRequest{
			TenantInfo: tenantInfo,
			Provider:   batch.Provider,
			LastFour:   lastFour,
		})
		if err != nil {
			return nil, err
		}
		if card != nil {
			sc.cardByLastFour[lastFour] = card
		}
	}

	if batch.DefaultFuelCardID != nil && !batch.DefaultFuelCardID.IsNil() {
		card, err := s.repo.GetCardByID(ctx, &repositories.GetFuelCardByIDRequest{
			ID:         *batch.DefaultFuelCardID,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return nil, err
		}
		sc.defaultCard = card
	}

	if len(jurisdictionKeys) > 0 {
		keys := make([]string, 0, len(jurisdictionKeys))
		for key := range jurisdictionKeys {
			keys = append(keys, key)
		}
		found, err := s.jurisdictions.FindJurisdictionsByCodes(ctx, keys)
		if err != nil {
			return nil, err
		}
		sc.jurisdictions = found
	}

	return sc, nil
}

func (s *Service) buildImportRow(
	sc *stageContext,
	stagedRow *fuelimport.StagedRow,
) *fuelpurchase.ImportRow {
	row := &fuelpurchase.ImportRow{
		RowNumber: stagedRow.RowNumber,
		Cells:     stagedRow.Cells,
		Status:    fuelpurchase.ImportRowStatusNew,
	}

	if stagedRow.Err != nil {
		row.Status = fuelpurchase.ImportRowStatusError
		row.Error = stagedRow.Err.Error()
		return row
	}

	parsed := stagedRow.Parsed
	purchase := parsed.Purchase
	row.Parsed = &purchase
	row.TransactionReference = parsed.Reference

	if stagedRow.IsDuplicate() {
		row.Status = fuelpurchase.ImportRowStatusDuplicateInFile
		row.Error = "Duplicate of row " + strconv.Itoa(stagedRow.DuplicateOf)
		return row
	}

	if existingID, ok := sc.existing[parsed.Reference]; ok {
		row.Status = fuelpurchase.ImportRowStatusAlreadyImported
		row.FuelPurchaseID = &existingID
		row.Error = "A purchase with this reference was imported before"
		return row
	}

	notes := make([]string, 0, 3)

	jurisdiction := sc.jurisdictions[parsed.JurisdictionKey()]
	switch {
	case jurisdiction == nil:
		row.Status = fuelpurchase.ImportRowStatusError
		row.Error = parsed.JurisdictionCode + " is not a jurisdiction this system knows"
		return row
	case !jurisdiction.IsActive():
		row.Status = fuelpurchase.ImportRowStatusError
		row.Error = jurisdiction.Label() + " is not an active jurisdiction"
		return row
	}
	row.ResolvedJurisdictionID = &jurisdiction.ID
	purchase.JurisdictionID = jurisdiction.ID

	card := sc.cardByLastFour[parsed.CardLastFour]
	if card == nil && sc.defaultCard != nil {
		card = sc.defaultCard
		if parsed.CardLastFour != "" {
			notes = append(notes, "No card ending "+parsed.CardLastFour+
				" was found; the import's default card was used")
		} else {
			notes = append(notes, "The import's default card was used")
		}
	}
	if card != nil {
		row.ResolvedFuelCardID = &card.ID
		purchase.FuelCardID = &card.ID
		if purchase.CardLastFour == "" {
			purchase.CardLastFour = card.LastFour
		}
		if card.AssignedWorkerID != nil && !card.AssignedWorkerID.IsNil() {
			purchase.WorkerID = card.AssignedWorkerID
		}
	}

	resolved, note := resolveTractor(sc, parsed, card)
	if resolved == nil {
		row.Status = fuelpurchase.ImportRowStatusError
		row.Error = tractorFailure(parsed)
		return row
	}
	if note != "" {
		notes = append(notes, note)
	}
	row.ResolvedTractorID = &resolved.ID
	purchase.TractorID = resolved.ID
	if purchase.WorkerID == nil && !resolved.PrimaryWorkerID.IsNil() {
		workerID := resolved.PrimaryWorkerID
		purchase.WorkerID = &workerID
	}

	purchase.Source = fuelpurchase.PurchaseSourceCardImport
	purchase.ImportBatchID = &sc.batch.ID
	purchase.OrganizationID = sc.batch.OrganizationID
	purchase.BusinessUnitID = sc.batch.BusinessUnitID
	purchase.Normalize()

	multiErr := errortypes.NewMultiError()
	purchase.Validate(multiErr)
	if multiErr.HasErrors() {
		row.Status = fuelpurchase.ImportRowStatusError
		row.Error = validationSummary(multiErr)
	}

	if len(notes) > 0 {
		row.ResolutionNotes = notes
	}

	return row
}

func resolveTractor(
	sc *stageContext,
	parsed *fuelimport.ParsedRow,
	card *fuelpurchase.FuelCard,
) (*tractor.Tractor, string) {
	if code := upperKey(parsed.TractorCode); code != "" {
		if entity := sc.tractorByCode[code]; entity != nil {
			return entity, ""
		}
	}
	if plate := upperKey(parsed.LicensePlate); plate != "" {
		if entity := sc.tractorByPlate[plate]; entity != nil {
			return entity, "Tractor matched by license plate " + parsed.LicensePlate
		}
	}
	if card != nil && card.AssignedTractorID != nil && !card.AssignedTractorID.IsNil() {
		return &tractor.Tractor{
			ID:             *card.AssignedTractorID,
			OrganizationID: card.OrganizationID,
			BusinessUnitID: card.BusinessUnitID,
		}, "Tractor taken from card ending " + card.LastFour
	}
	return nil, ""
}

func tractorFailure(parsed *fuelimport.ParsedRow) string {
	switch {
	case parsed.TractorCode != "":
		return "No tractor matched unit " + parsed.TractorCode
	case parsed.LicensePlate != "":
		return "No tractor matched license plate " + parsed.LicensePlate
	case parsed.CardLastFour != "":
		return "No tractor could be determined for card ending " + parsed.CardLastFour
	default:
		return "No tractor could be determined for this row"
	}
}

func validationSummary(multiErr *errortypes.MultiError) string {
	messages := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		if fieldErr == nil {
			continue
		}
		messages = append(messages, fieldErr.Message)
	}
	return strings.Join(messages, "; ")
}

type CommitRequest struct {
	TenantInfo pagination.TenantInfo
	BatchID    pulid.ID
	Version    int64
	UserID     pulid.ID
}

func (s *Service) Commit(
	ctx context.Context,
	req *CommitRequest,
) (*fuelpurchase.ImportBatch, error) {
	batch, err := s.repo.GetImportBatchByID(ctx, &repositories.GetImportBatchByIDRequest{
		ID:          req.BatchID,
		TenantInfo:  req.TenantInfo,
		IncludeRows: true,
	})
	if err != nil {
		return nil, err
	}
	if batch.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("FuelPurchaseImportBatch", batch.ID.String())
	}
	if !batch.CanCommit() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This import is {0} and cannot be committed", strings.ToLower(batch.Status.Label()),
		)
	}

	purchases := make([]*fuelpurchase.FuelPurchase, 0, batch.NewRowCount())
	rowIDByReference := make(map[string]pulid.ID, batch.NewRowCount())
	multiErr := errortypes.NewMultiError()

	for _, row := range batch.Rows {
		if !row.WillCommit() {
			continue
		}
		purchase := *row.Parsed
		purchase.ID = ""
		purchase.Version = 0
		purchase.OrganizationID = batch.OrganizationID
		purchase.BusinessUnitID = batch.BusinessUnitID
		purchase.Source = fuelpurchase.PurchaseSourceCardImport
		purchase.ImportBatchID = &batch.ID
		purchase.CreatedByID = req.UserID
		if purchase.TransactionReference == "" {
			purchase.TransactionReference = row.TransactionReference
		}
		purchase.Normalize()

		if purchase.TransactionReference == "" {
			multiErr.WithIndex("rows", row.RowNumber).
				Add("transactionReference", errortypes.ErrRequired, "Row has no reference")
			continue
		}
		purchase.Validate(multiErr.WithIndex("rows", row.RowNumber))

		purchases = append(purchases, &purchase)
		rowIDByReference[purchase.TransactionReference] = row.ID
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if len(purchases) == 0 {
		return nil, errortypes.NewValidationError(
			"rows",
			errortypes.ErrInvalid,
			"Nothing in this import is ready to commit",
		)
	}

	result, err := s.repo.CommitImport(ctx, &repositories.CommitImportRequest{
		Batch:            batch,
		Purchases:        purchases,
		RowIDByReference: rowIDByReference,
		CommittedByID:    req.UserID,
		CommittedAt:      s.now(),
	})
	if err != nil {
		return nil, err
	}

	committed, err := s.repo.GetImportBatchByID(ctx, &repositories.GetImportBatchByIDRequest{
		ID:         req.BatchID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchaseImport,
		resourceID: committed.ID.String(),
		operation:  permission.OpImport,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    committed,
		comment: "Imported " + strconv.Itoa(result.Committed) + " fuel purchases; " +
			strconv.Itoa(result.AlreadyImported) + " were already on file",
	})
	s.publish(ctx, req.TenantInfo, realtimeImport, permission.OpImport, committed.ID, req.UserID)
	s.publish(ctx, req.TenantInfo, realtimePurchase, permission.OpImport, committed.ID, req.UserID)

	return committed, nil
}

type DiscardRequest struct {
	TenantInfo pagination.TenantInfo
	BatchID    pulid.ID
	Version    int64
	Reason     string
	UserID     pulid.ID
}

func (s *Service) Discard(
	ctx context.Context,
	req *DiscardRequest,
) (*fuelpurchase.ImportBatch, error) {
	batch, err := s.repo.GetImportBatchByID(ctx, &repositories.GetImportBatchByIDRequest{
		ID:         req.BatchID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if batch.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("FuelPurchaseImportBatch", batch.ID.String())
	}
	if !batch.CanDiscard() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This import is {0} and cannot be discarded", strings.ToLower(batch.Status.Label()),
		)
	}

	previous := *batch
	batch.Status = fuelpurchase.ImportStatusDiscarded
	updated, err := s.repo.UpdateImportBatch(ctx, batch)
	if err != nil {
		return nil, err
	}

	comment := "Discarded the import"
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		comment += ": " + reason
	}
	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchaseImport,
		resourceID: updated.ID.String(),
		operation:  permission.OpCancel,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    comment,
	})
	s.publish(ctx, req.TenantInfo, realtimeImport, permission.OpCancel, updated.ID, req.UserID)

	return updated, nil
}

func formatOf(fileName string) (fuelpurchase.SourceFormat, error) {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))) {
	case ".csv", ".txt":
		return fuelpurchase.SourceFormatCSV, nil
	case ".xlsx", ".xlsm":
		return fuelpurchase.SourceFormatXLSX, nil
	default:
		return "", errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Fuel card statements can be imported as CSV or XLSX",
		)
	}
}

func readSheet(format fuelpurchase.SourceFormat, content []byte) (*rateimport.Sheet, error) {
	if format == fuelpurchase.SourceFormatXLSX {
		return rateimport.ReadXLSX(content)
	}
	return rateimport.ReadCSV(content)
}

func (s *Service) ListImports(
	ctx context.Context,
	req *repositories.ListImportBatchesRequest,
) (*pagination.CursorListResult[*fuelpurchase.ImportBatch], error) {
	return s.repo.ListImportBatches(ctx, req)
}
