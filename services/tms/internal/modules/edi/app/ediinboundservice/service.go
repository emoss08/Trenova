//nolint:gocritic // FX constructor params and repository request structs follow the existing value contracts.
package ediinboundservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	InboundFileRepo     edimessageports.EDIInboundFileRepository
	ProfileRepo         edipartnerports.EDICommunicationProfileRepository
	PartnerRepo         edipartnerports.EDIPartnerRepository
	MessageRepo         edimessageports.EDIMessageRepository
	DocumentTypeRepo    edipartnerports.EDIDocumentTypeRepository
	TransferRepo        editenderports.EDILoadTenderTransferRepository
	TenderRecipientRepo editenderports.EDITenderRecipientRepository
	TenderRepo          tenderports.TenderRepository           `optional:"true"`
	TenderResponses     tenderports.TenderResponseRecorder     `optional:"true"`
	InvoiceMatcher      carrierports.CarrierInvoiceAutoMatcher `optional:"true"`
	EDIService          *ediservice.Service
	Lading              *edilading.Runtime
	Flows               *edilading.Flows
	Replayer            *edilading.Replayer `optional:"true"`
	Transport           editransportports.EDITransportDispatcher
	ContentGate         contentscanports.Gate
	WorkflowStarter     kernelports.WorkflowStarter
	Metrics             *metrics.Registry `optional:"true"`
	// Watchtower puts a held-back file on the feed a person reads.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
	// Publisher wakes whichever agent covers intake on a file that could
	// not be turned into shipments or updates.
	Publisher agentports.AgentEventPublisher `optional:"true"`
}

type Service struct {
	l                   *zap.Logger
	inboundFileRepo     edimessageports.EDIInboundFileRepository
	profileRepo         edipartnerports.EDICommunicationProfileRepository
	partnerRepo         edipartnerports.EDIPartnerRepository
	messageRepo         edimessageports.EDIMessageRepository
	documentTypeRepo    edipartnerports.EDIDocumentTypeRepository
	transferRepo        editenderports.EDILoadTenderTransferRepository
	tenderRecipientRepo editenderports.EDITenderRecipientRepository
	tenderRepo          tenderports.TenderRepository
	tenderResponses     tenderports.TenderResponseRecorder
	invoiceMatcher      carrierports.CarrierInvoiceAutoMatcher
	ediService          *ediservice.Service
	lading              *edilading.Runtime
	flows               *edilading.Flows
	replayer            *edilading.Replayer
	transport           editransportports.EDITransportDispatcher
	workflowStarter     kernelports.WorkflowStarter
	metrics             *metrics.EDI
	watchtower          watchtowerports.WatchtowerProjector
	publisher           agentports.AgentEventPublisher
	contentGate         contentscanports.Gate
}

func New(p Params) *Service {
	ediMetrics := metrics.NewEDI(nil, p.Logger, false)
	if p.Metrics != nil {
		ediMetrics = p.Metrics.EDI
	}
	return &Service{
		contentGate:         p.ContentGate,
		l:                   p.Logger.Named("service.edi-inbound"),
		metrics:             ediMetrics,
		watchtower:          p.Watchtower,
		publisher:           p.Publisher,
		inboundFileRepo:     p.InboundFileRepo,
		profileRepo:         p.ProfileRepo,
		partnerRepo:         p.PartnerRepo,
		messageRepo:         p.MessageRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		transferRepo:        p.TransferRepo,
		tenderRecipientRepo: p.TenderRecipientRepo,
		tenderRepo:          p.TenderRepo,
		tenderResponses:     p.TenderResponses,
		invoiceMatcher:      p.InvoiceMatcher,
		ediService:          p.EDIService,
		lading:              p.Lading,
		flows:               p.Flows,
		replayer:            p.Replayer,
		transport:           p.Transport,
		workflowStarter:     p.WorkflowStarter,
	}
}

func (s *Service) ListPollableProfiles(ctx context.Context) ([]PollableProfile, error) {
	profiles, err := s.profileRepo.ListInboundPollingProfiles(ctx)
	if err != nil {
		return nil, err
	}
	pollable := make([]PollableProfile, 0, len(profiles))
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		pollable = append(pollable, PollableProfile{
			ProfileID: profile.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: profile.OrganizationID,
				BuID:  profile.BusinessUnitID,
			},
		})
	}
	return pollable, nil
}

func (s *Service) PollMailbox(
	ctx context.Context,
	req *PollMailboxRequest,
) (*PollMailboxResult, error) {
	if req == nil || req.ProfileID.IsNil() {
		return nil, errors.New("EDI communication profile ID is required for inbound polling")
	}
	profile, err := s.profileRepo.GetProfileByID(
		ctx,
		edipartnerports.GetEDICommunicationProfileByIDRequest{
			ID:         req.ProfileID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if profile.EDIPartnerID.IsNil() {
		return nil, errors.New(
			"EDI communication profile requires a partner before inbound polling",
		)
	}
	secrets, err := s.ediService.ProfileTransportSecrets(profile)
	if err != nil {
		return nil, err
	}
	fetchReq := &editransportports.EDIInboundFetchRequest{Profile: profile, Secrets: secrets}
	remoteFiles, err := s.transport.FetchInbound(ctx, profile.Method, fetchReq)
	if err != nil {
		s.metrics.RecordInboundPoll(string(profile.Method), "failed")
		s.recordPollOutcome(ctx, profile, false, err.Error())
		return nil, err
	}
	s.metrics.RecordInboundPoll(string(profile.Method), "success")
	s.recordPollOutcome(ctx, profile, true, "")

	result := &PollMailboxResult{ProfileID: profile.ID}
	for _, remoteFile := range remoteFiles {
		if remoteFile == nil || strings.TrimSpace(remoteFile.Contents) == "" {
			result.SkippedFiles++
			continue
		}
		staged, skipped, stageErr := s.stageInboundFile(ctx, profile, remoteFile)
		if stageErr != nil {
			return result, stageErr
		}
		if skipped {
			result.SkippedFiles++
		} else {
			result.StagedFileIDs = append(result.StagedFileIDs, staged.ID)
		}
		if archiveErr := s.transport.ArchiveInbound(
			ctx,
			profile.Method,
			fetchReq,
			remoteFile.Path,
		); archiveErr != nil {
			s.l.Warn(
				"failed to archive processed inbound EDI file on remote mailbox",
				zap.String("profileId", profile.ID.String()),
				zap.String("remotePath", remoteFile.Path),
				zap.Error(archiveErr),
			)
		}
	}
	return result, nil
}

func (s *Service) recordPollOutcome(
	ctx context.Context,
	profile *edi.EDICommunicationProfile,
	success bool,
	pollErr string,
) {
	if err := s.profileRepo.RecordInboundPollOutcome(
		ctx,
		edipartnerports.RecordEDIProfilePollOutcomeRequest{
			ProfileID: profile.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: profile.OrganizationID,
				BuID:  profile.BusinessUnitID,
			},
			PolledAt: timeutils.NowUnix(),
			Success:  success,
			Error:    pollErr,
		},
	); err != nil {
		s.l.Warn(
			"failed to record EDI inbound poll outcome",
			zap.String("profileId", profile.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) PollAndProcessMailbox(
	ctx context.Context,
	req *PollMailboxRequest,
) (*PollMailboxResult, error) {
	result, err := s.PollMailbox(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, fileID := range result.StagedFileIDs {
		if _, procErr := s.ProcessInboundFile(ctx, &ProcessInboundFileRequest{
			FileID:     fileID,
			TenantInfo: req.TenantInfo,
		}); procErr != nil {
			s.l.Error(
				"failed to process staged inbound EDI file during manual poll",
				zap.String("fileId", fileID.String()),
				zap.Error(procErr),
			)
		}
	}
	return result, nil
}

func (s *Service) stageInboundFile(
	ctx context.Context,
	profile *edi.EDICommunicationProfile,
	remoteFile *editransportports.EDIInboundRemoteFile,
) (*edi.EDIInboundFile, bool, error) {
	checksum := sha256.Sum256([]byte(remoteFile.Contents))
	checksumHex := hex.EncodeToString(checksum[:])
	tenantInfo := pagination.TenantInfo{
		OrgID: profile.OrganizationID,
		BuID:  profile.BusinessUnitID,
	}
	exists, err := s.inboundFileRepo.ExistsByChecksum(
		ctx,
		edimessageports.ExistsEDIInboundFileByChecksumRequest{
			TenantInfo:             tenantInfo,
			CommunicationProfileID: profile.ID,
			Checksum:               checksumHex,
		},
	)
	if err != nil {
		return nil, false, err
	}
	if exists {
		s.metrics.RecordInboundFile(
			profile.EDIPartnerID.String(),
			string(profile.Method),
			"duplicate",
		)
		return nil, true, nil
	}
	staged, err := s.inboundFileRepo.CreateInboundFile(ctx, &edi.EDIInboundFile{
		BusinessUnitID:         profile.BusinessUnitID,
		OrganizationID:         profile.OrganizationID,
		CommunicationProfileID: profile.ID,
		EDIPartnerID:           profile.EDIPartnerID,
		Method:                 profile.Method,
		RemotePath:             remoteFile.Path,
		FileName:               remoteFile.Name,
		Checksum:               checksumHex,
		SizeBytes:              remoteFile.Size,
		RawContent:             remoteFile.Contents,
		Status:                 edi.InboundFileStatusReceived,
		ReceivedAt:             timeutils.NowUnix(),
	})
	if err != nil {
		return nil, false, err
	}
	s.metrics.RecordInboundFile(profile.EDIPartnerID.String(), string(profile.Method), "staged")
	return staged, false, nil
}

func (s *Service) ProcessInboundFile(
	ctx context.Context,
	req *ProcessInboundFileRequest,
) (*edi.EDIInboundFile, error) {
	startedAt := time.Now()
	updated, err := observability.RunWithSpanReturn(
		ctx,
		"edi.process_inbound_file",
		func(ctx context.Context) (*edi.EDIInboundFile, error) {
			return s.processInboundFile(ctx, req)
		},
	)
	status := "error"
	if err == nil && updated != nil {
		status = string(updated.Status)
	}
	s.metrics.RecordInboundParse(status, time.Since(startedAt).Seconds())
	return updated, err
}

//nolint:funlen // Inbound file processing walks parse, dedup, routing, and ack stages sequentially.
func (s *Service) processInboundFile(
	ctx context.Context,
	req *ProcessInboundFileRequest,
) (*edi.EDIInboundFile, error) {
	if req == nil || req.FileID.IsNil() {
		return nil, errortypes.NewValidationError(
			"fileId",
			errortypes.ErrRequired,
			"EDI inbound file ID is required",
		)
	}
	file, err := s.inboundFileRepo.GetInboundFileByID(
		ctx,
		edimessageports.GetEDIInboundFileByIDRequest{ID: req.FileID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if req.Reprocess {
		if err = CheckReprocessable(file); err != nil {
			return nil, err
		}
	} else if file.Status != edi.InboundFileStatusReceived {
		return file, nil
	}

	if _, scanErr := contentscanports.Check(ctx, s.contentGate, &contentscanports.ScanRequest{
		TenantInfo: req.TenantInfo,
		Source:     contentscanports.SourceEDIInboundFile,
		FileName:   file.FileName,
		Content:    []byte(file.RawContent),
	}); scanErr != nil {
		var infected *contentscanports.InfectedError
		if errors.As(scanErr, &infected) {
			return s.quarantineFile(ctx, file, "the malware scanner found "+infected.Signature)
		}
		if errors.Is(scanErr, contentscanports.ErrTooLarge) {
			return s.quarantineFile(ctx, file, scanErr.Error())
		}
		return nil, scanErr
	}

	interchange, problem, err := parseInterchange(ctx, file.RawContent)
	if err != nil {
		return nil, err
	}
	if problem != nil {
		return s.quarantineFile(ctx, file, problem.Error())
	}

	file.InterchangeControlNumber = interchange.controlNumber
	file.ISASenderQualifier = interchange.senderQualifier
	file.ISASenderID = interchange.senderID
	file.ISAReceiverQualifier = interchange.receiverQualifier
	file.ISAReceiverID = interchange.receiverID
	file.TransactionCount = len(interchange.transactions)
	file.Status = edi.InboundFileStatusParsed
	file.FailureReason = ""
	file, err = s.inboundFileRepo.UpdateInboundFile(ctx, file)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			file.Status = edi.InboundFileStatusDuplicate
			file.FailureReason = fmt.Sprintf(
				"interchange control number %s was already processed for this partner",
				interchange.controlNumber,
			)
			return s.inboundFileRepo.UpdateInboundFile(ctx, file)
		}
		return nil, err
	}

	partner, err := s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
		ID:         file.EDIPartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return s.quarantineFile(
			ctx,
			file,
			"inbound file partner could not be resolved: "+err.Error(),
		)
	}

	read, err := s.readFile(ctx, req.TenantInfo, partner, interchange, file.RawContent)
	if errors.Is(err, errNoReceivingVersion) {
		return s.quarantineFile(ctx, file, fmt.Sprintf(
			"%s: activate a version of %s's Lading project to read its files",
			err.Error(),
			partner.Name,
		))
	}
	if err != nil {
		return nil, err
	}

	warnings := make([]string, 0)
	failures := 0
	processed := 0
	tendered := false
	first := map[edi.TransactionSet]*edi.EDIMessage{}
	for index := range interchange.transactions {
		transaction := &interchange.transactions[index]
		outcome := s.processTransaction(ctx, file, partner, interchange, read, transaction)
		if outcome.message != nil && first[transaction.set] == nil &&
			read.transaction(transaction) != nil {
			first[transaction.set] = outcome.message
		}
		warnings = append(warnings, outcome.warnings...)
		tendered = tendered || outcome.tendered
		if outcome.err != nil {
			failures++
			warnings = append(warnings, fmt.Sprintf(
				"transaction %s/%s failed: %s",
				transaction.set,
				transaction.controlNumber,
				outcome.err.Error(),
			))
			continue
		}
		processed++
	}

	s.shadow(ctx, file, read, first)
	s.generateInboundAcknowledgments(ctx, file, partner, interchange, read, &warnings)

	now := timeutils.NowUnix()
	file.ProcessedAt = &now
	file.FailureReason = strings.Join(warnings, "; ")
	switch {
	case processed == 0 && failures > 0:
		file.Status = edi.InboundFileStatusQuarantined
	case failures > 0 || len(warnings) > 0:
		file.Status = edi.InboundFileStatusPartiallyProcessed
	default:
		file.Status = edi.InboundFileStatusProcessed
	}
	updated, err := s.inboundFileRepo.UpdateInboundFile(ctx, file)
	if err != nil {
		return nil, err
	}
	s.metrics.RecordInboundOutcome(updated.EDIPartnerID.String(), string(updated.Status))
	if updated.Status == edi.InboundFileStatusQuarantined {
		s.notifyQuarantinedFile(ctx, updated)
	}
	if tendered {
		agentports.PublishAgentEvent(ctx, s.publisher, agentports.AgentEvent{
			Kind:       agent.EventEDITenderReceived,
			SubjectID:  updated.ID,
			TenantInfo: inboundFileTenant(updated),
		})
	}
	return updated, nil
}

func inboundFileTenant(file *edi.EDIInboundFile) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: file.OrganizationID,
		BuID:  file.BusinessUnitID,
	}
}

func (s *Service) quarantineFile(
	ctx context.Context,
	file *edi.EDIInboundFile,
	reason string,
) (*edi.EDIInboundFile, error) {
	now := timeutils.NowUnix()
	file.Status = edi.InboundFileStatusQuarantined
	file.FailureReason = reason
	file.ProcessedAt = &now
	updated, err := s.inboundFileRepo.UpdateInboundFile(ctx, file)
	if err != nil {
		return nil, err
	}
	s.metrics.RecordInboundOutcome(updated.EDIPartnerID.String(), string(updated.Status))
	s.notifyQuarantinedFile(ctx, updated)
	return updated, nil
}

func (s *Service) notifyQuarantinedFile(ctx context.Context, file *edi.EDIInboundFile) {
	if s.watchtower != nil {
		s.watchtower.Upsert(ctx, watchtowersources.DescribeQuarantinedFile(file))
	}

	agentports.PublishAgentEvent(ctx, s.publisher, agentports.AgentEvent{
		Kind:       agent.EventEDIFileQuarantined,
		SubjectID:  file.ID,
		TenantInfo: inboundFileTenant(file),
	})

	s.ediService.NotifyOperationalFailure(ctx, &ediservice.EDIOperationalAlert{
		OrganizationID: file.OrganizationID,
		BusinessUnitID: file.BusinessUnitID,
		EventType:      ediservice.EDIAlertEventInboundFileQuarantined,
		PartnerID:      file.EDIPartnerID,
		Title:          "EDI inbound file quarantined",
		Message: fmt.Sprintf(
			"Inbound file %s could not be processed: %s",
			stringutils.FirstNonEmpty(file.FileName, file.ID.String()),
			file.FailureReason,
		),
		RelatedEntities: map[string]any{
			"inboundFileId": file.ID,
			"partnerId":     file.EDIPartnerID,
		},
		Data: map[string]any{
			"fileName": file.FileName,
			"error":    file.FailureReason,
			"link":     "/edi/inbound-files?panelType=edit&panelEntityId=" + file.ID.String(),
		},
	})
}

func (s *Service) processTransaction(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	interchange *parsedInterchange,
	r *reading,
	transaction *parsedTransaction,
) transactionOutcome {
	read := r.transaction(transaction)
	if read != nil {
		read.apply(interchange, transaction)
	}
	recorded, err := s.recordInboundMessage(ctx, file, transaction, read, r.programs.Active)
	if err != nil {
		return transactionOutcome{err: err}
	}
	message := recorded.message
	outcome := transactionOutcome{message: message}
	switch {
	case recorded.unreadable != nil:
		outcome.err = recorded.unreadable
		return outcome
	case transaction.set.IsAcknowledgment():
		if transaction.rejected() {
			outcome.err = rejectedTransactionError(transaction)
			return outcome
		}
		outcome.warnings = s.routeAcknowledgment(ctx, file, partner, message, transaction)
		return outcome
	case read == nil:
		reason, ok := r.skipped[transaction.set]
		if !ok {
			reason = "the partner's map read no transaction with this control number"
		}
		outcome.warnings = []string{fmt.Sprintf(
			"transaction %s/%s recorded without processing: %s",
			transaction.set,
			transaction.controlNumber,
			reason,
		)}
		return outcome
	}
	outcome = s.receive(ctx, &received{
		file:        file,
		partner:     partner,
		message:     message,
		transaction: transaction,
		read:        read,
		programs:    r.programs,
	})
	if transaction.rejected() && outcome.err == nil {
		outcome.err = rejectedTransactionError(transaction)
	}
	return outcome
}

type recordedMessage struct {
	message    *edi.EDIMessage
	unreadable error
}

// recordInboundMessage keeps a received transaction with what the partner's version read from
// it. A document the version read that Trenova cannot take is kept as failed and reported as
// unreadable.
func (s *Service) recordInboundMessage(
	ctx context.Context,
	file *edi.EDIInboundFile,
	transaction *parsedTransaction,
	read *readTransaction,
	program *edilading.Program,
) (*recordedMessage, error) {
	documentType, err := s.inboundDocumentType(ctx, transaction.set)
	if err != nil {
		return nil, err
	}
	payload := edi.DocumentPayload{TransactionSet: transaction.set}
	var payloadErr error
	switch {
	case transaction.set.IsAcknowledgment():
		payload = transaction.documentPayload()
	case read != nil:
		payload, payloadErr = read.payload(transaction.set)
		if payloadErr != nil {
			payloadErr = fmt.Errorf(
				"the document map %s read could not be taken: %w",
				read.clause.Map,
				payloadErr,
			)
		}
	}
	status := edi.MessageStatusGenerated
	if transaction.rejected() || payloadErr != nil {
		status = edi.MessageStatusFailed
	}
	message := &edi.EDIMessage{
		BusinessUnitID:  file.BusinessUnitID,
		OrganizationID:  file.OrganizationID,
		EDIPartnerID:    file.EDIPartnerID,
		DocumentTypeID:  documentType.ID,
		InboundFileID:   file.ID,
		LadingVersionID: program.Version.ID,
		Direction:       edi.DocumentDirectionInbound,
		Standard:        edi.EDIStandardX12,
		TransactionSet:  transaction.set,
		X12Version: stringutils.FirstNonEmpty(
			transaction.version,
			documentType.DefaultVersion,
		),
		Status:                   status,
		InterchangeControlNumber: file.InterchangeControlNumber,
		GroupControlNumber:       transaction.groupControlNumber,
		TransactionControlNumber: transaction.controlNumber,
		SegmentCount:             int64(len(transaction.segments)),
		RawX12:                   transaction.raw,
		PayloadSnapshot:          payload,
		AckStatus:                edi.MessageAcknowledgmentStatusNotExpected,
	}
	if read != nil {
		message.Document = read.result.Document
		message.Lineage = read.result.Lineage
		message.Outcomes = read.result.Outcomes
	}
	created, err := s.messageRepo.CreateMessageWithDiagnostics(
		ctx,
		edimessageports.CreateEDIMessageWithDiagnosticsRequest{
			Message:     message,
			Diagnostics: findingDiagnostics(transaction.findings),
		},
	)
	if err != nil {
		return nil, err
	}
	return &recordedMessage{message: created, unreadable: payloadErr}, nil
}

func findingDiagnostics(findings []x12.Finding) []*edi.EDIMessageValidationError {
	diagnostics := make([]*edi.EDIMessageValidationError, 0, len(findings))
	for index := range findings {
		finding := &findings[index]
		diagnostic := edix12inspect.FindingDiagnostic(finding, i18n.Default)
		diagnostics = append(diagnostics, &edi.EDIMessageValidationError{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			SegmentID:       diagnostic.SegmentID,
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
		})
	}
	return diagnostics
}

func rejectedTransactionError(transaction *parsedTransaction) error {
	count := 0
	first := ""
	for index := range transaction.findings {
		if transaction.findings[index].IsError() {
			if count == 0 {
				first = transaction.findings[index].Message.String()
			}
			count++
		}
	}
	return fmt.Errorf(
		"%w: %d errors against the partner's guide, the first: %s",
		errRejectedTransaction,
		count,
		first,
	)
}

func (s *Service) inboundDocumentType(
	ctx context.Context,
	transactionSet edi.TransactionSet,
) (*edi.EDIDocumentType, error) {
	documentTypes, err := s.documentTypeRepo.ListDocumentTypes(
		ctx,
		edipartnerports.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: transactionSet,
			Direction:      edi.DocumentDirectionInbound,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(documentTypes) == 0 {
		documentTypes, err = s.documentTypeRepo.ListDocumentTypes(
			ctx,
			edipartnerports.ListEDIDocumentTypesRequest{
				Standard:       edi.EDIStandardX12,
				TransactionSet: transactionSet,
			},
		)
		if err != nil {
			return nil, err
		}
	}
	if len(documentTypes) == 0 {
		return nil, fmt.Errorf(
			"x12 %s document type is not seeded",
			transactionSet,
		)
	}
	return documentTypes[0], nil
}

func (s *Service) generateInboundAcknowledgments(
	ctx context.Context,
	file *edi.EDIInboundFile,
	partner *edi.EDIPartner,
	interchange *parsedInterchange,
	read *reading,
	warnings *[]string,
) {
	tenantInfo := inboundFileTenant(file)
	for index := range interchange.groups {
		group := &interchange.groups[index]
		set := acknowledgedSet(interchange, index)
		if set == "" {
			continue
		}
		ackType := read.programs.Active.Acknowledgment(set)
		if ackType == edi.AcknowledgmentTypeNone {
			continue
		}
		_, err := s.ediService.GenerateAcknowledgment(
			ctx,
			&ediservice.GenerateAcknowledgmentRequest{
				TenantInfo:   tenantInfo,
				EDIPartnerID: partner.ID,
				Type:         ackType,
				Group:        group.group,
				IdempotencyKey: fmt.Sprintf(
					"inbound-ack:%s:%d:%d",
					file.ID,
					file.Version,
					index,
				),
			},
		)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf(
				"failed to generate %s acknowledgment for group %s: %s",
				ackType,
				stringutils.FirstNonEmpty(group.controlNumber, strconv.Itoa(index+1)),
				err.Error(),
			))
		}
	}
}

// acknowledgedSet is the transaction set whose receive clause decides how a group is
// acknowledged: its first set that is not itself an acknowledgment, since a group of
// acknowledgments is never acknowledged.
func acknowledgedSet(interchange *parsedInterchange, group int) edi.TransactionSet {
	for index := range interchange.transactions {
		transaction := &interchange.transactions[index]
		if transaction.group != group {
			continue
		}
		if transaction.set != edi.TransactionSet997 && transaction.set != edi.TransactionSet999 {
			return transaction.set
		}
	}
	return ""
}

func (s *Service) ListInboundFiles(
	ctx context.Context,
	req *edimessageports.ListEDIInboundFilesRequest,
) (*pagination.ListResult[*edi.EDIInboundFile], error) {
	return s.inboundFileRepo.ListInboundFiles(ctx, req)
}

func (s *Service) GetInboundFile(
	ctx context.Context,
	req edimessageports.GetEDIInboundFileByIDRequest,
) (*edi.EDIInboundFile, error) {
	return s.inboundFileRepo.GetInboundFileByID(ctx, req)
}

func (s *Service) InspectInboundFile(
	ctx context.Context,
	req edimessageports.GetEDIInboundFileByIDRequest,
) (*edix12inspect.FileInspection, error) {
	file, err := s.inboundFileRepo.GetInboundFileByID(
		ctx,
		edimessageports.GetEDIInboundFileByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if file.RawPurgedAt != nil || strings.TrimSpace(file.RawContent) == "" {
		return nil, errortypes.NewValidationError(
			"rawContent",
			errortypes.ErrInvalid,
			"The raw file was purged under the retention policy, so it cannot be inspected",
		)
	}

	result, err := edix12inspect.InspectX12(
		ctx,
		&edix12inspect.InspectX12Request{RawX12: file.RawContent},
	)
	if err != nil {
		return nil, err
	}
	overview := result.FileInspection()
	return &overview, nil
}

func (s *Service) ListInboundFilesCursor(
	ctx context.Context,
	req *edimessageports.ListEDIInboundFilesRequest,
) (*pagination.CursorListResult[*edi.EDIInboundFile], error) {
	return s.inboundFileRepo.ListInboundFilesCursor(ctx, req)
}
