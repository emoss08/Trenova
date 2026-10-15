package factoringservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/factoringremit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	intakeNoFactor      = "No factor uses this address"
	intakeManyFactors   = "More than one factor uses this address; import the remittance by hand"
	intakeNoRemittance  = "The message has no CSV, spreadsheet or PDF to read"
	maxEmailRemittances = 10
)

func (s *RemittanceService) IngestEmail(
	ctx context.Context,
	req *factoringports.IngestRemittanceEmailRequest,
) (*factoringports.IngestRemittanceEmailResult, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if req.InboundMessageID.IsNil() {
		return nil, errortypes.NewValidationError(
			"inboundMessageId",
			errortypes.ErrRequired,
			"Choose the message",
		)
	}
	result := &factoringports.IngestRemittanceEmailResult{RemittanceIDs: []pulid.ID{}}
	factor, problem, err := s.factorForSender(ctx, req.TenantInfo, req.FromAddress)
	if err != nil {
		return nil, err
	}
	if factor == nil {
		result.Problem = problem
		return result, nil
	}
	result.FactoringCompanyID = factor.ID
	result.FactorName = factor.Name

	for idx := range req.Attachments {
		if len(result.RemittanceIDs) >= maxEmailRemittances {
			break
		}
		attachment := &req.Attachments[idx]
		id, ingestErr := s.ingestAttachment(ctx, req, factor, attachment)
		if ingestErr != nil {
			return nil, ingestErr
		}
		if id.IsNotNil() {
			result.RemittanceIDs = append(result.RemittanceIDs, id)
		}
	}
	if len(result.RemittanceIDs) == 0 {
		result.Problem = intakeNoRemittance
	}
	return result, nil
}

func (s *RemittanceService) factorForSender(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	address string,
) (*factoringcompany.FactoringCompany, string, error) {
	found, err := s.factors.FindByEmail(
		ctx,
		factoringcompanyports.FindFactoringCompaniesByEmailRequest{
			TenantInfo: tenantInfo,
			Address:    strings.TrimSpace(address),
		},
	)
	if err != nil {
		return nil, "", err
	}
	switch {
	case len(found.Exact) == 1:
		return found.Exact[0], "", nil
	case len(found.Exact) > 1:
		return nil, intakeManyFactors, nil
	case emailutils.IsPublicMailDomain(emailutils.Domain(address)):
		return nil, intakeNoFactor, nil
	case len(found.ByDomain) == 1:
		return found.ByDomain[0], "", nil
	case len(found.ByDomain) > 1:
		return nil, intakeManyFactors, nil
	default:
		return nil, intakeNoFactor, nil
	}
}

func (s *RemittanceService) ingestAttachment(
	ctx context.Context,
	req *factoringports.IngestRemittanceEmailRequest,
	factor *factoringcompany.FactoringCompany,
	attachment *factoringports.RemittanceEmailAttachment,
) (pulid.ID, error) {
	if attachment.DocumentID.IsNil() {
		return pulid.Nil, nil
	}
	if !factoringremit.IsRemittanceFile(attachment.FileName, attachment.ContentType) {
		return pulid.Nil, nil
	}
	existing, err := s.remittances.FindByInboundDocument(
		ctx,
		&factoringports.FindRemittanceByDocumentRequest{
			TenantInfo:       req.TenantInfo,
			InboundMessageID: req.InboundMessageID,
			DocumentID:       attachment.DocumentID,
		},
	)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return pulid.Nil, err
	}
	if existing != nil {
		return existing.ID, nil
	}

	actor := new(systemActor(req.TenantInfo))
	rem := &factoring.Remittance{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		FactoringCompanyID: factor.ID,
		Source:             factoring.RemittanceSourceEmail,
		InboundMessageID:   req.InboundMessageID,
		DocumentID:         attachment.DocumentID,
		FileName:           attachment.FileName,
		SenderEmail:        senderEmail(req.FromAddress),
		DefaultKind:        factoring.RemittanceLineFunding,
	}
	if req.ReceivedAt > 0 && req.ReceivedAt <= s.now()+maxFutureMovementSeconds {
		rem.RemittedOn = new(req.ReceivedAt)
	}
	created, err := s.create(ctx, rem, actor)
	if err != nil {
		return pulid.Nil, err
	}
	if err = s.stageEmailed(ctx, created, attachment.DocumentID); err != nil {
		return pulid.Nil, err
	}
	return created.ID, nil
}

func (s *RemittanceService) stageEmailed(
	ctx context.Context,
	created *factoring.Remittance,
	documentID pulid.ID,
) error {
	tenantInfo := remittanceTenant(created)
	actor := new(systemActor(tenantInfo))
	prepared := &preparedStage{}
	var staging *factoring.RemittanceStaging
	doc, err := s.remittanceDocument(ctx, created, documentID)
	if err == nil {
		staging, err = stagingFor(created, doc, "", nil, s.now())
	}
	if err == nil {
		prepared, err = s.prepare(ctx, created, doc, staging, true)
	}
	if err != nil {
		if !isUserFacing(err) {
			return err
		}
		prepared = &preparedStage{problem: problemOf(err)}
	}
	return s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		rem, lockErr := s.lock(txCtx, tenantInfo, created.ID, created.Version)
		if lockErr != nil {
			return lockErr
		}
		if doc == nil || staging == nil {
			_, lockErr = s.fail(txCtx, rem, nil, prepared.problem, actor)
			return lockErr
		}
		_, lockErr = s.applyStage(txCtx, &stageInput{
			rem:     rem,
			doc:     doc,
			staging: staging,
			actor:   actor,
		}, prepared)
		if lockErr != nil && isUserFacing(lockErr) {
			s.l.Warn("an emailed remittance could not be read",
				zap.String("remittanceId", rem.ID.String()),
				zap.Error(lockErr))
			_, lockErr = s.fail(txCtx, rem, staging, problemOf(lockErr), actor)
		}
		return lockErr
	})
}

func senderEmail(address string) string {
	address = strings.TrimSpace(address)
	if emailutils.Validate(address) != nil || len(address) > factoring.MaxRemittanceSenderLength {
		return ""
	}
	return address
}
