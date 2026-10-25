package factoringservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/factoringremit"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	problemNoRows       = "The file has a header row and no lines under it"
	problemNoTable      = "Nothing in this file looks like a table of invoices"
	problemTooManyLines = "The remittance has more lines than can be read at once; split it and import each part"
	problemPdfUnread    = "The PDF's text could not be read; upload the factor's CSV or spreadsheet"
	problemPdfSlow      = "The PDF was still being read after fifteen minutes; read it again"
	problemNoPdfModel   = "No model reads PDF remittances here; upload the factor's CSV or spreadsheet"
	problemPdfNoLines   = "No invoice lines were found in the PDF; upload the factor's CSV or spreadsheet"
)

type stageInput struct {
	rem      *factoring.Remittance
	doc      *document.Document
	staging  *factoring.RemittanceStaging
	override map[string]int
	actor    *permissionports.RequestActor
}

type preparedStage struct {
	sheet     *rateimport.Sheet
	fromModel bool
	problem   string
	await     bool
}

func (s *RemittanceService) Stage(
	ctx context.Context,
	req *factoringports.StageRemittanceRequest,
	actor *permissionports.RequestActor,
) (*factoring.Remittance, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Reading a remittance"); err != nil {
		return nil, err
	}
	if err := s.validateStageRequest(req); err != nil {
		return nil, err
	}
	current, err := s.remittances.Get(ctx, factoringports.GetRemittanceRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	if err = requireVersion(current, req.Version); err != nil {
		return nil, err
	}
	if err = current.RequireStageable(); err != nil {
		return nil, err
	}
	doc, err := s.remittanceDocument(ctx, current, req.DocumentID)
	if err != nil {
		return nil, err
	}
	staging, err := stagingFor(current, doc, req.DefaultKind, req.RemittedOn, s.now())
	if err != nil {
		return nil, err
	}
	prepared, err := s.prepare(ctx, current, doc, staging, true)
	if err != nil {
		return nil, err
	}

	var updated *factoring.Remittance
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		rem, lockErr := s.lock(txCtx, req.TenantInfo, req.ID, req.Version)
		if lockErr != nil {
			return lockErr
		}
		if lockErr = rem.RequireStageable(); lockErr != nil {
			return lockErr
		}
		updated, lockErr = s.applyStage(txCtx, &stageInput{
			rem:      rem,
			doc:      doc,
			staging:  staging,
			override: req.Mapping,
			actor:    actor,
		}, prepared)
		return lockErr
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *RemittanceService) validateStageRequest(req *factoringports.StageRemittanceRequest) error {
	if req.DocumentID.IsNil() {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrRequired,
			"Upload the factor's remittance before reading it",
		)
	}
	if req.DefaultKind != "" &&
		(!req.DefaultKind.IsValid() || req.DefaultKind == factoring.RemittanceLineOther) {
		return errortypes.NewValidationError(
			"defaultKind",
			errortypes.ErrInvalid,
			"Choose a funding, rejection, reserve release or chargeback report",
		)
	}
	return s.requireRemittedOn(req.RemittedOn)
}

func requireVersion(rem *factoring.Remittance, version int64) error {
	if rem.Version == version {
		return nil
	}
	return errortypes.NewConflictError(
		"Remittance {0} changed since you opened it; reload and try again",
		remittanceName(rem),
	)
}

func (s *RemittanceService) StagePending(
	ctx context.Context,
	req *factoringports.RemittanceVersionRequest,
	giveUp bool,
) (*factoringports.StagePendingResult, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	current, err := s.remittances.Get(ctx, factoringports.GetRemittanceRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	result := &factoringports.StagePendingResult{Remittance: current}
	if !current.AwaitsReading() {
		return result, nil
	}
	actor := new(systemActor(req.TenantInfo))
	prepared := &preparedStage{}
	var staging *factoring.RemittanceStaging
	doc, err := s.remittanceDocument(ctx, current, current.DocumentID)
	if err == nil {
		staging, err = stagingFor(current, doc, "", nil, s.now())
	}
	if err == nil {
		prepared, err = s.prepare(ctx, current, doc, staging, !giveUp)
	}
	if err != nil {
		if !isUserFacing(err) {
			return nil, err
		}
		prepared.problem = problemOf(err)
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		rem, lockErr := s.remittances.Get(txCtx, factoringports.GetRemittanceRequest{
			TenantInfo:   req.TenantInfo,
			ID:           req.ID,
			IncludeLines: true,
			ForUpdate:    true,
		})
		if lockErr != nil {
			return lockErr
		}
		result.Remittance = rem
		if !rem.AwaitsReading() || rem.Version != current.Version {
			return nil
		}
		if doc == nil || staging == nil {
			result.Remittance, lockErr = s.fail(txCtx, rem, nil, prepared.problem, actor)
			return lockErr
		}
		result.Remittance, lockErr = s.applyStage(txCtx, &stageInput{
			rem:     rem,
			doc:     doc,
			staging: staging,
			actor:   actor,
		}, prepared)
		return lockErr
	})
	if err != nil {
		return nil, err
	}
	result.Waiting = result.Remittance.AwaitsReading()
	return result, nil
}

func isUserFacing(err error) bool {
	var validationErr *errortypes.Error
	var businessErr *errortypes.BusinessError
	var multiErr *errortypes.MultiError
	return errors.As(err, &validationErr) || errors.As(err, &businessErr) ||
		errors.As(err, &multiErr)
}

func stagingFor(
	rem *factoring.Remittance,
	doc *document.Document,
	defaultKind factoring.RemittanceLineKind,
	remittedOn *int64,
	at int64,
) (*factoring.RemittanceStaging, error) {
	fileName := strings.TrimSpace(doc.OriginalName)
	if fileName == "" {
		fileName = doc.FileName
	}
	format, err := factoringremit.FormatFor(fileName, doc.FileType)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"A remittance must be a CSV, an Excel workbook (.xlsx) or a PDF",
		)
	}
	if defaultKind == "" {
		defaultKind = rem.DefaultKind
	}
	if remittedOn == nil {
		remittedOn = rem.RemittedOn
	}
	return &factoring.RemittanceStaging{
		DocumentID:  doc.ID,
		FileName:    fileName,
		Format:      format,
		DefaultKind: defaultKind,
		RemittedOn:  remittedOn,
		Mapping:     rem.Mapping,
		Headers:     rem.Headers,
		At:          at,
	}, nil
}

func (s *RemittanceService) remittanceDocument(
	ctx context.Context,
	rem *factoring.Remittance,
	documentID pulid.ID,
) (*document.Document, error) {
	doc, err := s.documents.Get(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: remittanceTenant(rem),
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"documentId",
				errortypes.ErrInvalid,
				"The uploaded file could not be found",
			)
		}
		return nil, err
	}
	if !documentBelongsTo(doc, rem) {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"This file was not uploaded for this remittance",
		)
	}
	if doc.FileSize > factoring.MaxRemittanceBytes {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"This file is too large to read; the limit is 32 MiB",
		)
	}
	return doc, nil
}

func documentBelongsTo(doc *document.Document, rem *factoring.Remittance) bool {
	switch rem.Source {
	case factoring.RemittanceSourceEmail:
		return rem.InboundMessageID.IsNotNil() &&
			doc.ResourceType == inboundmessage.AttachmentResourceType &&
			doc.ResourceID == rem.InboundMessageID.String()
	case factoring.RemittanceSourceUpload:
		return doc.ResourceType == factoring.RemittanceDocumentResourceType &&
			doc.ResourceID == rem.ID.String()
	}
	return false
}

func (s *RemittanceService) prepare(
	ctx context.Context,
	rem *factoring.Remittance,
	doc *document.Document,
	staging *factoring.RemittanceStaging,
	wait bool,
) (*preparedStage, error) {
	if staging.Format == factoring.RemittanceFormatPdf {
		return s.preparePdf(ctx, rem, doc, wait)
	}
	data, err := s.readDocument(ctx, rem, doc.ID)
	if err != nil {
		return nil, err
	}
	sheet, err := factoringremit.ReadSheet(data, staging.Format, s.sheetLimits)
	if err != nil {
		return &preparedStage{problem: sheetProblemOf(err)}, nil
	}
	return &preparedStage{sheet: sheet}, nil
}

func (s *RemittanceService) applyStage(
	ctx context.Context,
	in *stageInput,
	prepared *preparedStage,
) (*factoring.Remittance, error) {
	switch {
	case prepared.problem != "":
		return s.fail(ctx, in.rem, in.staging, prepared.problem, in.actor)
	case prepared.await:
		return s.awaitPdf(ctx, in)
	case prepared.fromModel:
		mapping := factoringremit.ModelMapping()
		in.staging.Headers = prepared.sheet.Headers
		in.staging.UnmappedHeaders = nil
		in.staging.Mapping = mapping.Strings()
		parsed := factoringremit.Parse(prepared.sheet, mapping, factoringremit.Options{
			DefaultKind: in.staging.DefaultKind,
			Location:    time.UTC,
		})
		return s.finishStage(ctx, in, factoringremit.ToDomain(parsed))
	default:
		return s.stageSheet(ctx, in, prepared.sheet)
	}
}

func (s *RemittanceService) stageSheet(
	ctx context.Context,
	in *stageInput,
	sheet *rateimport.Sheet,
) (*factoring.Remittance, error) {
	mapping, chosen, err := s.resolveMapping(ctx, in, sheet.Headers)
	if err != nil {
		return nil, err
	}
	in.staging.Headers = sheet.Headers
	in.staging.Mapping = mapping.Strings()
	in.staging.UnmappedHeaders = factoringremit.UnmappedHeaders(sheet.Headers, mapping)
	if len(factoringremit.Validate(mapping)) > 0 {
		return s.needsMapping(ctx, in)
	}
	parsed := factoringremit.Parse(sheet, mapping, factoringremit.Options{
		DefaultKind: in.staging.DefaultKind,
		Location:    time.UTC,
	})
	updated, err := s.finishStage(ctx, in, factoringremit.ToDomain(parsed))
	if err != nil || !chosen || updated.Status != factoring.RemittanceStatusParsed {
		return updated, err
	}
	if err = s.remittances.SaveProfile(ctx, &factoring.RemittanceProfile{
		OrganizationID:     updated.OrganizationID,
		BusinessUnitID:     updated.BusinessUnitID,
		FactoringCompanyID: updated.FactoringCompanyID,
		Mapping:            mapping.Strings(),
		HeaderSignature:    factoringremit.HeaderSignature(sheet.Headers),
	}); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *RemittanceService) resolveMapping(
	ctx context.Context,
	in *stageInput,
	headers []string,
) (mapping factoringremit.Mapping, chosen bool, err error) {
	if len(in.override) > 0 {
		mapping, err = mappingFromRequest(in.override, len(headers))
		return mapping, err == nil, err
	}
	guessed := factoringremit.GuessMapping(headers)
	profile, err := s.remittances.GetProfile(ctx, factoringports.GetRemittanceProfileRequest{
		TenantInfo:         remittanceTenant(in.rem),
		FactoringCompanyID: in.rem.FactoringCompanyID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, false, err
	}
	if profile == nil {
		return guessed, false, nil
	}
	saved, usable := usableProfile(profile, len(headers))
	if !usable {
		return guessed, false, nil
	}
	if profile.HeaderSignature == factoringremit.HeaderSignature(headers) ||
		len(factoringremit.Validate(guessed)) > 0 {
		return saved, false, nil
	}
	return guessed, false, nil
}

func usableProfile(
	profile *factoring.RemittanceProfile,
	width int,
) (factoringremit.Mapping, bool) {
	saved, err := factoringremit.MappingFromStrings(profile.Mapping)
	if err != nil {
		return nil, false
	}
	return saved, saved.FitsHeaders(width) && len(factoringremit.Validate(saved)) == 0
}

func mappingFromRequest(raw map[string]int, width int) (factoringremit.Mapping, error) {
	mapping, err := factoringremit.MappingFromStrings(raw)
	if err != nil {
		return nil, errortypes.NewValidationError("mapping", errortypes.ErrInvalid, err.Error())
	}
	if !mapping.FitsHeaders(width) {
		return nil, errortypes.NewValidationError(
			"mapping",
			errortypes.ErrInvalid,
			"The mapping names a column the file does not have",
		)
	}
	problems := factoringremit.Validate(mapping)
	if len(problems) == 0 {
		return mapping, nil
	}
	multiErr := errortypes.NewMultiError()
	for _, problem := range problems {
		multiErr.Add("mapping."+problem.Field.String(), errortypes.ErrInvalid, problem.Message)
	}
	return nil, multiErr
}

func (s *RemittanceService) needsMapping(
	ctx context.Context,
	in *stageInput,
) (*factoring.Remittance, error) {
	before := cloneRemittance(in.rem)
	if err := in.rem.MarkNeedsMapping(in.staging); err != nil {
		return nil, err
	}
	if err := s.remittances.ReplaceLines(ctx, in.rem); err != nil {
		return nil, err
	}
	updated, err := s.save(ctx, in.rem)
	if err != nil {
		return nil, err
	}
	s.record(ctx, in.actor, before, updated, permission.OpUpdate,
		"Remittance "+updated.FileName+" needs its columns mapped")
	return updated, nil
}

func (s *RemittanceService) finishStage(
	ctx context.Context,
	in *stageInput,
	lines []*factoring.RemittanceLine,
) (*factoring.Remittance, error) {
	switch {
	case len(lines) == 0:
		return s.fail(ctx, in.rem, in.staging, problemNoRows, in.actor)
	case len(lines) > factoring.MaxRemittanceLines:
		return s.fail(ctx, in.rem, in.staging, problemTooManyLines, in.actor)
	}
	before := cloneRemittance(in.rem)
	if err := in.rem.MarkParsed(in.staging, lines); err != nil {
		return nil, err
	}
	candidates, err := s.candidatesFor(ctx, in.rem, lines)
	if err != nil {
		return nil, err
	}
	factoring.MatchRemittance(lines, candidates)
	in.rem.Recount()
	if err = s.remittances.ReplaceLines(ctx, in.rem); err != nil {
		return nil, err
	}
	updated, err := s.save(ctx, in.rem)
	if err != nil {
		return nil, err
	}
	s.record(ctx, in.actor, before, updated, permission.OpUpdate,
		"Remittance "+updated.FileName+" read: "+countsPhrase(updated))
	s.announceStaged(ctx, updated)
	return updated, nil
}

func (s *RemittanceService) fail(
	ctx context.Context,
	rem *factoring.Remittance,
	staging *factoring.RemittanceStaging,
	problem string,
	actor *permissionports.RequestActor,
) (*factoring.Remittance, error) {
	before := cloneRemittance(rem)
	documentID, fileName, at := rem.DocumentID, rem.FileName, s.now()
	if staging != nil {
		documentID, fileName = staging.DocumentID, staging.FileName
		rem.SourceFormat = staging.Format
	}
	if err := rem.MarkFailed(documentID, fileName, problem, at); err != nil {
		return nil, err
	}
	if err := s.remittances.ReplaceLines(ctx, rem); err != nil {
		return nil, err
	}
	updated, err := s.save(ctx, rem)
	if err != nil {
		return nil, err
	}
	s.record(ctx, actor, before, updated, permission.OpUpdate,
		"Remittance "+remittanceName(updated)+" could not be read: "+updated.Error)
	return updated, nil
}

func (s *RemittanceService) readDocument(
	ctx context.Context,
	rem *factoring.Remittance,
	documentID pulid.ID,
) ([]byte, error) {
	content, err := s.documents.GetDownloadContent(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: remittanceTenant(rem),
	})
	if err != nil {
		return nil, err
	}
	if content == nil || content.Body == nil {
		return nil, errEmptyRemittance()
	}
	defer func() { _ = content.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(content.Body, factoring.MaxRemittanceBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read remittance document: %w", err)
	}
	switch {
	case len(data) > factoring.MaxRemittanceBytes:
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"This file is too large to read; the limit is 32 MiB",
		)
	case len(data) == 0:
		return nil, errEmptyRemittance()
	}
	return data, nil
}

func errEmptyRemittance() error {
	return errortypes.NewValidationError(
		"documentId",
		errortypes.ErrInvalid,
		"The uploaded file is empty",
	)
}

func sheetProblemOf(err error) string {
	switch {
	case errors.Is(err, rateimport.ErrNoRows):
		return problemNoRows
	case errors.Is(err, rateimport.ErrNoTable):
		return problemNoTable
	default:
		return err.Error()
	}
}

func problemOf(err error) string {
	var validationErr *errortypes.Error
	if errors.As(err, &validationErr) {
		return validationErr.Message
	}
	var businessErr *errortypes.BusinessError
	if errors.As(err, &businessErr) {
		return businessErr.Message
	}
	return "The remittance could not be read"
}

func (s *RemittanceService) preparePdf(
	ctx context.Context,
	rem *factoring.Remittance,
	doc *document.Document,
	wait bool,
) (*preparedStage, error) {
	if s.completion == nil || s.content == nil {
		return &preparedStage{problem: problemNoPdfModel}, nil
	}
	content, err := s.content.GetContent(ctx, doc.ID, remittanceTenant(rem))
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	status := documentcontent.StatusPending
	if content != nil {
		status = content.Status
	}
	switch status {
	case documentcontent.StatusFailed:
		return &preparedStage{problem: problemPdfUnread}, nil
	case documentcontent.StatusIndexed:
	case documentcontent.StatusPending,
		documentcontent.StatusExtracting,
		documentcontent.StatusExtracted:
		if !wait {
			return &preparedStage{problem: problemPdfSlow}, nil
		}
		return &preparedStage{await: true}, nil
	}
	if strings.TrimSpace(content.ContentText) == "" {
		return &preparedStage{problem: problemPdfNoLines}, nil
	}
	sheet, err := s.extractPdf(ctx, rem, doc.ID, content.ContentText)
	if err != nil {
		var problem *pdfProblemError
		if errors.As(err, &problem) {
			return &preparedStage{problem: problem.message}, nil
		}
		return nil, err
	}
	return &preparedStage{sheet: sheet, fromModel: true}, nil
}

func (s *RemittanceService) awaitPdf(
	ctx context.Context,
	in *stageInput,
) (*factoring.Remittance, error) {
	before := cloneRemittance(in.rem)
	if err := in.rem.MarkReading(in.staging); err != nil {
		return nil, err
	}
	if err := s.remittances.ReplaceLines(ctx, in.rem); err != nil {
		return nil, err
	}
	updated, err := s.save(ctx, in.rem)
	if err != nil {
		return nil, err
	}
	s.record(ctx, in.actor, before, updated, permission.OpUpdate,
		"Remittance "+updated.FileName+" is waiting for its PDF to be read")
	if s.starter != nil {
		s.starter.StartStage(ctx, remittanceTenant(updated), updated)
	}
	return updated, nil
}
