package factoringcontract

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
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

var _ factoringsubmissionports.SubmissionSendStarter = (*SendStarter)(nil)

func NewSendStarter(p SendStarterParams) factoringsubmissionports.SubmissionSendStarter {
	return &SendStarter{workflows: p.Workflows, l: p.Logger.Named("factoring-send-starter")}
}

func (s *SendStarter) StartSend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	submission *factoring.Submission,
) {
	if submission == nil {
		return
	}
	if s.workflows == nil || !s.workflows.Enabled() {
		s.l.Warn("factoring schedule sending is not available; the schedule waits in Sending",
			zap.String("submissionId", submission.ID.String()))
		return
	}
	id := SendWorkflowID(submission.ID, submission.SendAttemptGroup)
	payload := &SendPayload{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		SubmissionID:   submission.ID,
		Group:          submission.SendAttemptGroup,
	}
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		_, err := s.workflows.StartWorkflow(
			committed,
			client.StartWorkflowOptions{
				ID:                    id,
				TaskQueue:             temporaltype.TaskQueueBilling.String(),
				StaticSummary:         "Send factoring schedule " + submission.Number,
				WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
				Priority:              temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
			},
			SendWorkflowName,
			payload,
		)
		if err != nil {
			s.l.Error("could not start sending a factoring schedule",
				zap.String("submissionId", submission.ID.String()),
				zap.Error(err))
		}
	})
}
