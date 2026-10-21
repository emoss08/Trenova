package contentscanservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/clamav"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/shared/hashutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Config    *config.Config
	Audit     auditports.AuditService
	Logger    *zap.Logger
	Lifecycle fx.Lifecycle `optional:"true"`
}

type Service struct {
	scanner  contentscanports.Scanner
	failOpen bool
	audit    auditports.AuditService
	l        *zap.Logger
}

var _ contentscanports.Gate = (*Service)(nil)

func New(p Params) contentscanports.Gate {
	cfg := p.Config.GetContentScanConfig()
	var scanner contentscanports.Scanner
	if cfg.Enabled() {
		scanner = clamav.New(cfg.ClamAV)
	}
	svc := NewWithScanner(scanner, cfg.FailOpen, p.Audit, p.Logger)
	if p.Lifecycle != nil {
		p.Lifecycle.Append(fx.Hook{OnStart: svc.announce})
	}
	return svc
}

func NewWithScanner(
	scanner contentscanports.Scanner,
	failOpen bool,
	auditService auditports.AuditService,
	logger *zap.Logger,
) *Service {
	return &Service{
		scanner:  scanner,
		failOpen: failOpen,
		audit:    auditService,
		l:        logger.Named("service.content-scan"),
	}
}

func (s *Service) announce(ctx context.Context) error {
	if s.scanner == nil {
		s.l.Warn("malware scanning is off: no contentScan.clamav.address is configured, " +
			"so uploads, inbound attachments, capture pages and inbound EDI files are " +
			"parsed without being scanned")
		return nil
	}
	if err := s.scanner.Ping(ctx); err != nil {
		s.l.Warn("the malware scanner did not answer; files will be held until it does",
			zap.Bool("failOpen", s.failOpen), zap.Error(err))
	}
	return nil
}

func (s *Service) Check(
	ctx context.Context,
	req *contentscanports.ScanRequest,
) (*contentscanports.ScanResult, error) {
	if s.scanner == nil {
		return &contentscanports.ScanResult{Verdict: contentscanports.VerdictUnscanned}, nil
	}

	result, err := s.scanner.Scan(ctx, req.Content)
	if err != nil {
		if !s.failOpen || errors.Is(err, contentscanports.ErrTooLarge) {
			return nil, err
		}
		s.l.Warn("let a file through unscanned because the scanner could not check it",
			zap.String("source", string(req.Source)),
			zap.String("fileName", req.FileName),
			zap.Error(err))
		return &contentscanports.ScanResult{Verdict: contentscanports.VerdictUnscanned}, nil
	}

	if result.Verdict != contentscanports.VerdictInfected {
		return result, nil
	}

	s.recordInfected(req, result)
	return result, &contentscanports.InfectedError{Signature: result.Signature}
}

func (s *Service) recordInfected(
	req *contentscanports.ScanRequest,
	result *contentscanports.ScanResult,
) {
	checksum := hashutils.SHA256BytesHex(req.Content)
	s.l.Warn("refused a file the malware scanner flagged",
		zap.String("source", string(req.Source)),
		zap.String("fileName", req.FileName),
		zap.String("signature", result.Signature),
		zap.String("sha256", checksum),
		zap.String("organizationId", req.TenantInfo.OrgID.String()))

	if s.audit == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceDocument,
		ResourceID:     checksum,
		Operation:      permission.OpReject,
		UserID:         req.TenantInfo.UserID,
		PrincipalType:  permissionports.PrincipalTypeSystem,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Critical:       true,
		CurrentState: map[string]any{
			"fileName":  req.FileName,
			"source":    string(req.Source),
			"signature": result.Signature,
			"sha256":    checksum,
			"sizeBytes": len(req.Content),
		},
	}
	comment := fmt.Sprintf("Refused %q from %s: the malware scanner found %s",
		req.FileName, req.Source, result.Signature)
	if err := s.audit.LogAction(params, func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}); err != nil {
		s.l.Error("could not record a refused infected file", zap.Error(err))
	}
}
