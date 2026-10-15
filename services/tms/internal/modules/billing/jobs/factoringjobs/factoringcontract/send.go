package factoringcontract

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrSendUnavailable = errors.New("factoring schedule sending is not available in this process")

const (
	SendWorkflowName     = "FactoringSubmissionSendWorkflow"
	sendWorkflowIDPrefix = "factoring-submission-"
)

type SendPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	SubmissionID   pulid.ID `json:"submissionId"`
	Group          int      `json:"group"`
}

func (p *SendPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

func SendWorkflowID(submissionID pulid.ID, group int) string {
	return sendWorkflowIDPrefix + submissionID.String() + "-" + strconv.Itoa(group)
}

type SendStarterParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter `optional:"true"`
	Logger    *zap.Logger
}

type SendStarter struct {
	workflows kernelports.WorkflowStarter
	l         *zap.Logger
}

var _ factoringports.SubmissionSendStarter = (*SendStarter)(nil)

func NewSendStarter(p SendStarterParams) factoringports.SubmissionSendStarter {
	return &SendStarter{workflows: p.Workflows, l: p.Logger.Named("factoring-send-starter")}
}

func (s *SendStarter) StartSend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	submission *factoring.Submission,
) error {
	if submission == nil {
		return nil
	}
	if s.workflows == nil || !s.workflows.Enabled() {
		return ErrSendUnavailable
	}
	_, err := s.workflows.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                    SendWorkflowID(submission.ID, submission.SendAttemptGroup),
			TaskQueue:             temporaltype.TaskQueueBilling.String(),
			StaticSummary:         "Send factoring schedule " + submission.Number,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
			Priority:              temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		SendWorkflowName,
		&SendPayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			SubmissionID:   submission.ID,
			Group:          submission.SendAttemptGroup,
		},
	)
	if err != nil {
		return fmt.Errorf("start sending factoring schedule %s: %w", submission.Number, err)
	}
	return nil
}
