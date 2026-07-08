package ediinboundservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	as2AsyncMDNTimeout = 30 * time.Second
	as2FileNameSuffix  = ".edi"
)

var ErrAS2ProfileNotFound = errortypes.NewAuthorizationError(
	"No active AS2 communication profile matches the AS2 identifiers",
)

type ReceiveAS2MessageRequest struct {
	From                           string
	To                             string
	MessageID                      string
	ContentType                    string
	TransferEncoding               string
	DispositionNotificationTo      string
	DispositionNotificationOptions string
	ReceiptDeliveryOption          string
	Body                           []byte
}

type ReceiveAS2MessageResult struct {
	MDNContentType string
	MDNBody        []byte
	MDNHeaders     textproto.MIMEHeader
	AsyncMDN       bool
	Duplicate      bool
	Rejected       bool
	RejectReason   string
	IsMDN          bool
}

func (s *Service) ReceiveAS2Message(
	ctx context.Context,
	req *ReceiveAS2MessageRequest,
) (*ReceiveAS2MessageResult, error) {
	return observability.RunWithSpanReturn(
		ctx,
		"edi.receive_as2_message",
		func(ctx context.Context) (*ReceiveAS2MessageResult, error) {
			return s.receiveAS2Message(ctx, req)
		},
	)
}

func (s *Service) receiveAS2Message(
	ctx context.Context,
	req *ReceiveAS2MessageRequest,
) (*ReceiveAS2MessageResult, error) {
	if req == nil || strings.TrimSpace(req.From) == "" || strings.TrimSpace(req.To) == "" {
		return nil, errortypes.NewValidationError(
			"headers",
			errortypes.ErrRequired,
			"AS2-From and AS2-To headers are required",
		)
	}
	if as2.IsMDNContentType(req.ContentType) {
		if err := s.ediService.ApplyAS2MDN(ctx, &ediservice.ApplyAS2MDNRequest{
			ContentType: req.ContentType,
			Body:        req.Body,
		}); err != nil {
			return nil, err
		}
		return &ReceiveAS2MessageResult{IsMDN: true}, nil
	}

	profile, err := s.profileRepo.GetActiveAS2ProfileByIdentifiers(
		ctx,
		repositories.GetActiveAS2ProfileByIdentifiersRequest{
			LocalAS2ID:   strings.TrimSpace(req.To),
			PartnerAS2ID: strings.TrimSpace(req.From),
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, ErrAS2ProfileNotFound
		}
		return nil, err
	}
	secrets, err := s.ediService.ProfileTransportSecrets(profile)
	if err != nil {
		return nil, err
	}
	cfg, err := editransport.AS2ConfigFromProfile(profile, secrets)
	if err != nil {
		return nil, err
	}

	parsed, parseErr := as2.ParseMessage(req.ContentType, req.Body, &as2.ParseMessageOptions{
		DecryptionCertificate: cfg.LocalCertificate,
		DecryptionKey:         cfg.PrivateKey,
		PartnerCertificate:    cfg.PartnerSigningCertificate,
		MICAlgorithm:          micAlgorithmFromOptions(req.DispositionNotificationOptions),
		TransferEncoding:      req.TransferEncoding,
		RequireSignature:      cfg.RequireSignedInbound,
		RequireEncryption:     cfg.RequireEncryptedInbound,
	})
	if parseErr != nil {
		s.l.Warn(
			"inbound AS2 message could not be decrypted or verified",
			zap.String("profileId", profile.ID.String()),
			zap.Error(parseErr),
		)
		s.metrics.RecordInboundFile(
			profile.EDIPartnerID.String(),
			string(profile.Method),
			"rejected",
		)
		return s.buildAS2MDNResult(req, cfg, "", parseErr)
	}

	fileName := stringutils.FirstNonEmpty(
		parsed.FileName,
		sanitizeAS2FileName(req.MessageID)+as2FileNameSuffix,
	)
	staged, skipped, err := s.stageInboundFile(ctx, profile, &services.EDIInboundRemoteFile{
		Path:     "as2:" + req.MessageID,
		Name:     fileName,
		Contents: string(parsed.Payload),
		Size:     int64(len(parsed.Payload)),
	})
	if err != nil {
		return s.buildAS2MDNResult(req, cfg, parsed.MIC, err)
	}

	result, err := s.buildAS2MDNResult(req, cfg, parsed.MIC, nil)
	if err != nil {
		return nil, err
	}
	result.Duplicate = skipped
	if skipped {
		return result, nil
	}
	s.startInboundProcessing(ctx, profile, staged)
	return result, nil
}

func (s *Service) startInboundProcessing(
	ctx context.Context,
	profile *edi.EDICommunicationProfile,
	file *edi.EDIInboundFile,
) {
	request := &ProcessInboundFileRequest{
		FileID: file.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: profile.OrganizationID,
			BuID:  profile.BusinessUnitID,
		},
	}
	if s.workflowStarter != nil && s.workflowStarter.Enabled() {
		if _, err := s.workflowStarter.StartWorkflow(
			ctx,
			client.StartWorkflowOptions{
				ID:        "edi-process-inbound-file-" + file.ID.String(),
				TaskQueue: temporaltype.EDITaskQueue,
				StaticSummary: fmt.Sprintf(
					"Process inbound AS2 EDI file %s",
					file.ID.String(),
				),
			},
			temporaltype.ProcessInboundEDIFileWorkflowName,
			request,
		); err != nil {
			s.l.Warn(
				"failed to start inbound AS2 processing workflow; processing inline",
				zap.String("fileId", file.ID.String()),
				zap.Error(err),
			)
		} else {
			return
		}
	}
	if _, err := s.ProcessInboundFile(ctx, request); err != nil {
		s.l.Error(
			"failed to process inbound AS2 file",
			zap.String("fileId", file.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) buildAS2MDNResult(
	req *ReceiveAS2MessageRequest,
	cfg *editransport.AS2Config,
	mic string,
	receiveErr error,
) (*ReceiveAS2MessageResult, error) {
	options := &as2.BuildMDNOptions{
		From:               strings.TrimSpace(req.To),
		To:                 strings.TrimSpace(req.From),
		OriginalMessageID:  req.MessageID,
		ReceivedContentMIC: mic,
	}
	if receiveErr != nil {
		options.ErrorText = receiveErr.Error()
	}
	if wantsSignedMDN(req.DispositionNotificationOptions) &&
		cfg.LocalCertificate != nil && cfg.PrivateKey != nil {
		options.SigningCertificate = cfg.LocalCertificate
		options.SigningKey = cfg.PrivateKey
		options.SigningAlgorithm = cfg.SigningAlgorithm
	}
	mdn, err := as2.BuildMDN(options)
	if err != nil {
		return nil, err
	}
	result := &ReceiveAS2MessageResult{
		MDNContentType: mdn.ContentType,
		MDNBody:        mdn.Body,
		MDNHeaders:     mdn.Headers,
		AsyncMDN:       strings.TrimSpace(req.ReceiptDeliveryOption) != "",
		Rejected:       receiveErr != nil,
	}
	if receiveErr != nil {
		result.RejectReason = receiveErr.Error()
	}
	return result, nil
}

func (s *Service) SendAsyncAS2MDN(
	ctx context.Context,
	returnURL string,
	result *ReceiveAS2MessageResult,
) error {
	if result == nil || len(result.MDNBody) == 0 {
		return errors.New("AS2 MDN payload is required for async delivery")
	}
	target, err := httpsafe.ValidateURL(returnURL)
	if err != nil {
		return fmt.Errorf("async AS2 MDN return URL is not permitted: %w", err)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		target.String(),
		bytes.NewReader(result.MDNBody),
	)
	if err != nil {
		return fmt.Errorf("async AS2 MDN request could not be created: %w", err)
	}
	request.Header.Set("Content-Type", result.MDNContentType)
	for key, values := range result.MDNHeaders {
		for _, value := range values {
			request.Header.Set(key, value)
		}
	}
	httpClient := httpsafe.NewClient(as2AsyncMDNTimeout)
	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("async AS2 MDN delivery to %s failed: %w", returnURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf(
			"async AS2 MDN delivery to %s was rejected with status %d",
			returnURL,
			response.StatusCode,
		)
	}
	return nil
}

func (s *Service) LogAsyncMDNFailure(returnURL string, err error) {
	s.l.Error(
		"failed to deliver asynchronous AS2 MDN",
		zap.String("returnUrl", returnURL),
		zap.Error(err),
	)
}

func micAlgorithmFromOptions(options string) string {
	for part := range strings.SplitSeq(options, ";") {
		key, value, found := strings.Cut(part, "=")
		if !found || !strings.EqualFold(strings.TrimSpace(key), "signed-receipt-micalg") {
			continue
		}
		fields := strings.Split(value, ",")
		algorithm := strings.TrimSpace(fields[len(fields)-1])
		return strings.ReplaceAll(strings.ToLower(algorithm), "sha-", "sha")
	}
	return ""
}

func wantsSignedMDN(options string) bool {
	return strings.Contains(strings.ToLower(options), "pkcs7-signature")
}

func sanitizeAS2FileName(messageID string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, strings.Trim(messageID, "<>"))
	if cleaned == "" {
		return "as2-message"
	}
	return cleaned
}
