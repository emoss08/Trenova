package edijobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const PollInboundMailboxesWorkflowName = "PollInboundEDIMailboxesWorkflow"

type PollInboundMailboxesResult struct {
	PolledProfiles int `json:"polledProfiles"`
	StagedFiles    int `json:"stagedFiles"`
	ProcessedFiles int `json:"processedFiles"`
	FailedProfiles int `json:"failedProfiles"`
}

var inboundPollActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

var inboundProcessActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            "edi-inbound-poll",
			Description:   "Poll partner SFTP and VAN mailboxes for inbound EDI files",
			Spec:          schedule.Every(2 * time.Minute),
			Workflow:      PollInboundMailboxesWorkflow,
			TaskQueue:     temporaltype.EDITaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "edi-inbound-poll",
			},
		},
		{
			ID:            "edi-raw-retention-purge",
			Description:   "Purge raw EDI payloads past each organization's retention window",
			Spec:          schedule.Cron("0 3 * * *"),
			Workflow:      PurgeEDIRawPayloadsWorkflow,
			TaskQueue:     temporaltype.EDITaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "edi-raw-retention-purge",
			},
		},
	}
}

//nolint:funlen // The polling workflow sequences discovery, per-profile polling, and per-file processing.
func PollInboundMailboxesWorkflow(
	ctx workflow.Context,
) (*PollInboundMailboxesResult, error) {
	pollCtx := workflow.WithActivityOptions(ctx, inboundPollActivityOptions)
	processCtx := workflow.WithActivityOptions(ctx, inboundProcessActivityOptions)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	result := new(PollInboundMailboxesResult)
	var profiles []ediinboundservice.PollableProfile
	if err := workflow.ExecuteActivity(
		pollCtx,
		a.ListPollableEDIProfilesActivity,
	).Get(pollCtx, &profiles); err != nil {
		logger.Error("failed to list pollable EDI profiles", "error", err)
		return nil, err
	}

	for _, profile := range profiles {
		result.PolledProfiles++
		var pollResult ediinboundservice.PollMailboxResult
		if err := workflow.ExecuteActivity(
			pollCtx,
			a.PollEDIMailboxActivity,
			&ediinboundservice.PollMailboxRequest{
				ProfileID:  profile.ProfileID,
				TenantInfo: profile.TenantInfo,
			},
		).Get(pollCtx, &pollResult); err != nil {
			result.FailedProfiles++
			logger.Error(
				"failed to poll EDI mailbox",
				"profileId", profile.ProfileID.String(),
				"error", err,
			)
			continue
		}
		result.StagedFiles += len(pollResult.StagedFileIDs)
		for _, fileID := range pollResult.StagedFileIDs {
			if err := workflow.ExecuteActivity(
				processCtx,
				a.ProcessInboundEDIFileActivity,
				&ediinboundservice.ProcessInboundFileRequest{
					FileID:     fileID,
					TenantInfo: profile.TenantInfo,
				},
			).Get(processCtx, nil); err != nil {
				logger.Error(
					"failed to process inbound EDI file",
					"fileId", fileID.String(),
					"error", err,
				)
				continue
			}
			result.ProcessedFiles++
		}
	}
	return result, nil
}

func (a *Activities) ListPollableEDIProfilesActivity(
	ctx context.Context,
) ([]ediinboundservice.PollableProfile, error) {
	profiles, err := a.inboundService.ListPollableProfiles(ctx)
	if err != nil {
		a.logger.Error("failed to list pollable EDI profiles", zap.Error(err))
		return nil, err
	}
	return profiles, nil
}

func (a *Activities) PollEDIMailboxActivity(
	ctx context.Context,
	req *ediinboundservice.PollMailboxRequest,
) (*ediinboundservice.PollMailboxResult, error) {
	result, err := a.inboundService.PollMailbox(ctx, req)
	if err != nil {
		a.logger.Error(
			"EDI mailbox polling activity failed",
			zap.String("profileId", req.ProfileID.String()),
			zap.Error(err),
		)
		return nil, err
	}
	return result, nil
}

func (a *Activities) ProcessInboundEDIFileActivity(
	ctx context.Context,
	req *ediinboundservice.ProcessInboundFileRequest,
) error {
	file, err := a.inboundService.ProcessInboundFile(ctx, req)
	if err != nil {
		a.logger.Error(
			"inbound EDI file processing activity failed",
			zap.String("fileId", req.FileID.String()),
			zap.Error(err),
		)
		return err
	}
	a.logger.Info(
		"inbound EDI file processed",
		zap.String("fileId", file.ID.String()),
		zap.String("status", string(file.Status)),
	)
	return nil
}

func ProcessInboundEDIFileWorkflow(
	ctx workflow.Context,
	req *ediinboundservice.ProcessInboundFileRequest,
) error {
	processCtx := workflow.WithActivityOptions(ctx, inboundProcessActivityOptions)
	var a *Activities
	if err := workflow.ExecuteActivity(
		processCtx,
		a.ProcessInboundEDIFileActivity,
		req,
	).Get(processCtx, nil); err != nil {
		workflow.GetLogger(ctx).Error("inbound EDI file workflow failed", "error", err)
		return err
	}
	return nil
}
