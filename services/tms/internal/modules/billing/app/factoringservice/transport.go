package factoringservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxSftpSecretBytes = 16 << 10

type TransportParams struct {
	fx.In

	Repo       factoringsubmissionports.TransportRepository
	Factors    factoringcompanyports.FactoringCompanyRepository
	Cipher     kernelports.SecretCipher
	Transports []factoringsubmissionports.FactoringTransport `group:"factoring_transports"`
	Audit      auditports.AuditService
	Realtime   realtimeports.RealtimeService `                             optional:"true"`
	Logger     *zap.Logger
}

type TransportService struct {
	repo       factoringsubmissionports.TransportRepository
	factors    factoringcompanyports.FactoringCompanyRepository
	cipher     kernelports.SecretCipher
	transports map[factoring.TransportMethod]factoringsubmissionports.FactoringTransport
	rec        recorder
	l          *zap.Logger
	now        func() int64
}

var _ factoringsubmissionports.TransportService = (*TransportService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewTransportService(p TransportParams) factoringsubmissionports.TransportService {
	l := p.Logger.Named("service.factoring-transport")
	return &TransportService{
		repo:       p.Repo,
		factors:    p.Factors,
		cipher:     p.Cipher,
		transports: transportsByMethod(p.Transports),
		rec:        recorder{audit: p.Audit, realtime: p.Realtime, l: l},
		l:          l,
		now:        timeutils.NowUnix,
	}
}

func transportsByMethod(
	list []factoringsubmissionports.FactoringTransport,
) map[factoring.TransportMethod]factoringsubmissionports.FactoringTransport {
	out := make(
		map[factoring.TransportMethod]factoringsubmissionports.FactoringTransport,
		len(list),
	)
	for _, transport := range list {
		if transport != nil {
			out[transport.Method()] = transport
		}
	}
	return out
}

func (s *TransportService) Get(
	ctx context.Context,
	req factoringsubmissionports.GetTransportRequest,
) (*factoring.Transport, error) {
	return s.repo.Get(ctx, req)
}

func (s *TransportService) Save(
	ctx context.Context,
	req *factoringsubmissionports.SaveTransportRequest,
	actor *permissionports.RequestActor,
) (*factoring.Transport, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Changing how schedules reach a factor"); err != nil {
		return nil, err
	}
	if _, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:         req.FactoringCompanyID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	current, err := s.repo.Get(ctx, factoringsubmissionports.GetTransportRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: req.FactoringCompanyID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	exists := err == nil
	if exists && (req.Version == nil || *req.Version != current.Version) {
		return nil, errortypes.NewConflictError(
			"The factor's delivery settings changed since you opened them; reload and try again",
		)
	}

	entity := &factoring.Transport{
		ID:                 pulid.MustNew("factrn_"),
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		FactoringCompanyID: req.FactoringCompanyID,
	}
	var previous *factoring.Transport
	if exists {
		snapshot := *current
		previous = &snapshot
		entity = current
	}
	if err = s.apply(ctx, req, entity, previous); err != nil {
		return nil, err
	}

	var saved *factoring.Transport
	if exists {
		saved, err = s.repo.Update(ctx, entity)
	} else {
		saved, err = s.repo.Create(ctx, entity)
	}
	if err != nil {
		return nil, err
	}

	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceFactoringCompany,
		entity:   redacted(saved),
		previous: redacted(previous),
		op:       permission.OpUpdate,
		comment:  "Schedule delivery set to " + saved.Method.String() + " " + saved.Target(),
	})
	s.rec.publish(ctx, permission.ResourceFactoringCompany, redacted(saved), actor, "updated")
	return saved, nil
}

func (s *TransportService) apply(
	ctx context.Context,
	req *factoringsubmissionports.SaveTransportRequest,
	entity *factoring.Transport,
	previous *factoring.Transport,
) error {
	in := req.Input
	entity.Method = in.Method
	entity.EmailRecipients = in.EmailRecipients
	entity.AttachPackets = in.AttachPackets
	entity.SftpHost = in.SftpHost
	entity.SftpPort = in.SftpPort
	entity.SftpUsername = in.SftpUsername
	entity.SftpAuthMethod = in.SftpAuthMethod
	entity.SftpKnownHostKey = in.SftpKnownHostKey
	entity.SftpDirectory = in.SftpDirectory

	switch {
	case in.SftpSecret != nil && strings.TrimSpace(*in.SftpSecret) != "":
		if len(*in.SftpSecret) > maxSftpSecretBytes {
			return errortypes.NewValidationError(
				"sftpSecret",
				errortypes.ErrInvalid,
				"The SFTP password or private key is too long",
			)
		}
		sealed, err := s.seal(ctx, req.TenantInfo, entity, *in.SftpSecret)
		if err != nil {
			return err
		}
		entity.SftpSecret = sealed
	case previous != nil && previous.SftpAuthMethod != in.SftpAuthMethod:
		entity.SftpSecret = ""
	}

	entity.Normalize()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	return validationOf(multiErr)
}

func (s *TransportService) seal(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *factoring.Transport,
	secret string,
) (string, error) {
	sealed, err := s.cipher.Seal(ctx, secret, transportSecretScope(tenantInfo, entity))
	if err != nil {
		return "", errortypes.NewBusinessError("The SFTP secret could not be stored").
			WithInternal(err)
	}
	return sealed, nil
}

func (s *TransportService) Test(
	ctx context.Context,
	req *factoringsubmissionports.TestTransportRequest,
	actor *permissionports.RequestActor,
) (*factoringsubmissionports.TransportTestResult, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireActor(actor, "Testing a factor's delivery settings"); err != nil {
		return nil, err
	}
	entity, err := s.repo.Get(ctx, factoringsubmissionports.GetTransportRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: req.FactoringCompanyID,
	})
	if err != nil {
		return nil, err
	}
	sender, ok := s.transports[entity.Method]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"{0} delivery is not available",
			entity.Method.String(),
		)
	}
	secret, err := openSecret(ctx, s.cipher, req.TenantInfo, entity)
	if err != nil {
		return nil, err
	}

	testErr := sender.Test(ctx, &factoringsubmissionports.TransportTestRequest{
		TenantInfo: req.TenantInfo,
		Transport:  entity,
		Secret:     secret,
	})
	entity.RecordTest(s.now(), testErr)
	if err = s.repo.RecordTest(ctx, entity); err != nil {
		return nil, err
	}

	result := &factoringsubmissionports.TransportTestResult{
		Transport: entity,
		Passed:    testErr == nil,
	}
	if testErr != nil {
		result.Message = testErr.Error()
	} else {
		result.Message = "Connected to " + entity.Target()
	}
	s.rec.publish(ctx, permission.ResourceFactoringCompany, redacted(entity), actor, "updated")
	return result, nil
}

func openSecret(
	ctx context.Context,
	cipher kernelports.SecretCipher,
	tenantInfo pagination.TenantInfo,
	entity *factoring.Transport,
) (string, error) {
	if entity.Method != factoring.TransportMethodSftp || entity.SftpSecret == "" {
		return "", nil
	}
	secret, err := cipher.Open(ctx, entity.SftpSecret, transportSecretScope(tenantInfo, entity))
	if err != nil {
		return "", errortypes.NewBusinessError(
			"The SFTP password or private key could not be read; enter it again on the factor",
		).WithInternal(err)
	}
	return secret, nil
}

func transportSecretScope(
	tenantInfo pagination.TenantInfo,
	entity *factoring.Transport,
) *kernelports.SecretScope {
	return &kernelports.SecretScope{
		Purpose:    kernelports.SecretPurposeFactoringTransport,
		Tenant:     tenantInfo,
		ResourceID: entity.ID.String() + ":sftp_secret",
	}
}

func redacted(entity *factoring.Transport) *factoring.Transport {
	if entity == nil {
		return nil
	}
	copied := *entity
	if copied.SftpSecret != "" {
		copied.SftpSecret = "set"
	}
	return &copied
}
