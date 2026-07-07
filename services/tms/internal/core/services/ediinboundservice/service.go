//nolint:gocritic // FX constructor params and repository request structs follow the existing value contracts.
package ediinboundservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	InboundFileRepo     repositories.EDIInboundFileRepository
	ProfileRepo         repositories.EDICommunicationProfileRepository
	PartnerRepo         repositories.EDIPartnerRepository
	MessageRepo         repositories.EDIMessageRepository
	DocumentTypeRepo    repositories.EDIDocumentTypeRepository
	DocumentProfileRepo repositories.EDIPartnerDocumentProfileRepository
	TransferRepo        repositories.EDILoadTenderTransferRepository
	TenderRecipientRepo repositories.EDITenderRecipientRepository
	EDIService          *ediservice.Service
	Transport           services.EDITransportDispatcher
	WorkflowStarter     services.WorkflowStarter
	Metrics             *metrics.Registry `optional:"true"`
}

type Service struct {
	l                   *zap.Logger
	inboundFileRepo     repositories.EDIInboundFileRepository
	profileRepo         repositories.EDICommunicationProfileRepository
	partnerRepo         repositories.EDIPartnerRepository
	messageRepo         repositories.EDIMessageRepository
	documentTypeRepo    repositories.EDIDocumentTypeRepository
	documentProfileRepo repositories.EDIPartnerDocumentProfileRepository
	transferRepo        repositories.EDILoadTenderTransferRepository
	tenderRecipientRepo repositories.EDITenderRecipientRepository
	ediService          *ediservice.Service
	transport           services.EDITransportDispatcher
	workflowStarter     services.WorkflowStarter
	metrics             *metrics.EDI
}

func New(p Params) *Service {
	ediMetrics := metrics.NewEDI(nil, p.Logger, false)
	if p.Metrics != nil {
		ediMetrics = p.Metrics.EDI
	}
	return &Service{
		l:                   p.Logger.Named("service.edi-inbound"),
		metrics:             ediMetrics,
		inboundFileRepo:     p.InboundFileRepo,
		profileRepo:         p.ProfileRepo,
		partnerRepo:         p.PartnerRepo,
		messageRepo:         p.MessageRepo,
		documentTypeRepo:    p.DocumentTypeRepo,
		documentProfileRepo: p.DocumentProfileRepo,
		transferRepo:        p.TransferRepo,
		tenderRecipientRepo: p.TenderRecipientRepo,
		ediService:          p.EDIService,
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
		repositories.GetEDICommunicationProfileByIDRequest{
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
	fetchReq := &services.EDIInboundFetchRequest{Profile: profile, Secrets: secrets}
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
		repositories.RecordEDIProfilePollOutcomeRequest{
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

func (s *Service) stageInboundFile(
	ctx context.Context,
	profile *edi.EDICommunicationProfile,
	remoteFile *services.EDIInboundRemoteFile,
) (*edi.EDIInboundFile, bool, error) {
	checksum := sha256.Sum256([]byte(remoteFile.Contents))
	checksumHex := hex.EncodeToString(checksum[:])
	tenantInfo := pagination.TenantInfo{
		OrgID: profile.OrganizationID,
		BuID:  profile.BusinessUnitID,
	}
	exists, err := s.inboundFileRepo.ExistsByChecksum(
		ctx,
		repositories.ExistsEDIInboundFileByChecksumRequest{
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
		repositories.GetEDIInboundFileByIDRequest{ID: req.FileID, TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if req.Reprocess {
		if !file.Status.IsReprocessable() {
			return nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Only quarantined or partially processed EDI inbound files can be reprocessed",
			)
		}
	} else if file.Status != edi.InboundFileStatusReceived {
		return file, nil
	}

	interchange, parseErr := parseInterchange(file.RawContent)
	if parseErr != nil {
		return s.quarantineFile(ctx, file, parseErr.Error())
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

	partner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
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

	warnings := make([]string, 0)
	failures := 0
	processed := 0
	for index := range interchange.transactions {
		transaction := &interchange.transactions[index]
		outcome := s.processTransaction(ctx, file, partner, transaction)
		warnings = append(warnings, outcome.warnings...)
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

	s.generateInboundAcknowledgments(ctx, file, partner, interchange, &warnings)

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
	return updated, nil
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
	transaction *parsedTransaction,
) transactionOutcome {
	message, err := s.recordInboundMessage(ctx, file, transaction)
	if err != nil {
		return transactionOutcome{err: err}
	}
	outcome := transactionOutcome{message: message}
	switch transaction.set {
	case edi.TransactionSet997, edi.TransactionSet999:
		outcome.warnings = s.routeAcknowledgment(ctx, file, partner, message, transaction)
	case edi.TransactionSet990:
		outcome.warnings, outcome.err = s.routeTenderResponse(ctx, file, partner, transaction)
	case edi.TransactionSet214:
		outcome.warnings, outcome.err = s.routeShipmentStatus(ctx, file, partner, transaction)
	case edi.TransactionSet204:
		outcome.warnings, outcome.err = s.routeLoadTender(ctx, file, partner, message, transaction)
	case edi.TransactionSet210:
		outcome.warnings, outcome.err = s.routeFreightInvoice(
			ctx,
			file,
			partner,
			message,
			transaction,
		)
	default:
		outcome.warnings = []string{fmt.Sprintf(
			"transaction %s/%s recorded without processing: unsupported transaction set",
			transaction.set,
			transaction.controlNumber,
		)}
	}
	return outcome
}

func (s *Service) recordInboundMessage(
	ctx context.Context,
	file *edi.EDIInboundFile,
	transaction *parsedTransaction,
) (*edi.EDIMessage, error) {
	documentType, err := s.inboundDocumentType(ctx, transaction.set)
	if err != nil {
		return nil, err
	}
	payload := transaction.documentPayload()
	message := &edi.EDIMessage{
		BusinessUnitID:           file.BusinessUnitID,
		OrganizationID:           file.OrganizationID,
		EDIPartnerID:             file.EDIPartnerID,
		DocumentTypeID:           documentType.ID,
		InboundFileID:            file.ID,
		Direction:                edi.DocumentDirectionInbound,
		Standard:                 edi.EDIStandardX12,
		TransactionSet:           transaction.set,
		X12Version:               documentType.DefaultVersion,
		Status:                   edi.MessageStatusGenerated,
		ValidationMode:           edi.ValidationModeDisabled,
		InterchangeControlNumber: file.InterchangeControlNumber,
		GroupControlNumber:       transaction.groupControlNumber,
		TransactionControlNumber: transaction.controlNumber,
		SegmentCount:             int64(len(transaction.segments)),
		RawX12:                   transaction.raw,
		PayloadSnapshot:          payload,
		AckStatus:                edi.MessageAcknowledgmentStatusNotExpected,
	}
	return s.messageRepo.CreateMessageWithDiagnostics(
		ctx,
		repositories.CreateEDIMessageWithDiagnosticsRequest{Message: message},
	)
}

func (s *Service) inboundDocumentType(
	ctx context.Context,
	transactionSet edi.TransactionSet,
) (*edi.EDIDocumentType, error) {
	documentTypes, err := s.documentTypeRepo.ListDocumentTypes(
		ctx,
		repositories.ListEDIDocumentTypesRequest{
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
			repositories.ListEDIDocumentTypesRequest{
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
	warnings *[]string,
) {
	ackSets := map[edi.TransactionSet][]*parsedTransaction{}
	for index := range interchange.transactions {
		transaction := &interchange.transactions[index]
		if transaction.set == edi.TransactionSet997 ||
			transaction.set == edi.TransactionSet999 {
			continue
		}
		ackSets[transaction.set] = append(ackSets[transaction.set], transaction)
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: file.OrganizationID,
		BuID:  file.BusinessUnitID,
	}
	for transactionSet, transactions := range ackSets {
		ackType := s.inboundAcknowledgmentType(ctx, tenantInfo, partner.ID, transactionSet)
		if ackType == edi.AcknowledgmentTypeNone {
			continue
		}
		for _, transaction := range transactions {
			if err := s.generateAcknowledgment(
				ctx,
				tenantInfo,
				partner,
				transaction,
				ackType,
			); err != nil {
				*warnings = append(*warnings, fmt.Sprintf(
					"failed to generate %s acknowledgment for %s/%s: %s",
					ackType,
					transaction.set,
					transaction.controlNumber,
					err.Error(),
				))
			}
		}
	}
}

func (s *Service) inboundAcknowledgmentType(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
	transactionSet edi.TransactionSet,
) edi.AcknowledgmentType {
	profile, err := s.documentProfileRepo.GetActivePartnerDocumentProfile(
		ctx,
		repositories.GetActiveEDIPartnerDocumentProfileRequest{
			PartnerID:      partnerID,
			TenantInfo:     tenantInfo,
			TransactionSet: transactionSet,
			Direction:      edi.DocumentDirectionInbound,
		},
	)
	if err != nil || profile == nil {
		return edi.AcknowledgmentTypeNone
	}
	if !profile.Acknowledgment.Expected ||
		profile.Acknowledgment.Type == edi.AcknowledgmentTypeNone {
		return edi.AcknowledgmentTypeNone
	}
	return profile.Acknowledgment.Type
}

func (s *Service) generateAcknowledgment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
	transaction *parsedTransaction,
	ackType edi.AcknowledgmentType,
) error {
	ackSet := edi.TransactionSet997
	if ackType == edi.AcknowledgmentType999 {
		ackSet = edi.TransactionSet999
	}
	profile, err := s.ediService.EnsureOutboundDocumentProfile(
		ctx,
		tenantInfo,
		partner.ID,
		ackSet,
	)
	if err != nil {
		return err
	}
	payload := acknowledgmentPayloadForTransaction(transaction, ackSet)
	_, err = s.ediService.GenerateDocument(ctx, &ediservice.GenerateEDIDocumentRequest{
		TenantInfo:               tenantInfo,
		PartnerDocumentProfileID: profile.ID,
		EDIPartnerID:             partner.ID,
		TransactionSet:           ackSet,
		Direction:                edi.DocumentDirectionOutbound,
		Payload:                  &payload,
	})
	return err
}

func (s *Service) ListInboundFiles(
	ctx context.Context,
	req *repositories.ListEDIInboundFilesRequest,
) (*pagination.ListResult[*edi.EDIInboundFile], error) {
	return s.inboundFileRepo.ListInboundFiles(ctx, req)
}

func (s *Service) GetInboundFile(
	ctx context.Context,
	req repositories.GetEDIInboundFileByIDRequest,
) (*edi.EDIInboundFile, error) {
	return s.inboundFileRepo.GetInboundFileByID(ctx, req)
}

func (s *Service) ListInboundFilesCursor(
	ctx context.Context,
	req *repositories.ListEDIInboundFilesRequest,
) (*pagination.CursorListResult[*edi.EDIInboundFile], error) {
	return s.inboundFileRepo.ListInboundFilesCursor(ctx, req)
}
