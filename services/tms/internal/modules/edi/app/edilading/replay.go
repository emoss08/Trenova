package edilading

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	replayBatch        = 200
	maxReplayMessages  = 20_000
	maxReplayItemsKept = 500
)

type ReplayerParams struct {
	fx.In

	Logger   *zap.Logger
	Runtime  *Runtime
	Projects ediports.LadingProjectRepository
	Replays  ediports.LadingReplayRepository
	Messages ediports.EDIMessageRepository
}

type Replayer struct {
	l        *zap.Logger
	runtime  *Runtime
	projects ediports.LadingProjectRepository
	replays  ediports.LadingReplayRepository
	messages ediports.EDIMessageRepository
}

func NewReplayer(p ReplayerParams) *Replayer {
	return &Replayer{
		l:        p.Logger.Named("edi.lading.replay"),
		runtime:  p.Runtime,
		projects: p.Projects,
		replays:  p.Replays,
		messages: p.Messages,
	}
}

// Run replays a version against another on the messages of the replay's period, a batch at a
// time, and records what differs. heartbeat is called after every batch.
func (r *Replayer) Run(
	ctx context.Context,
	payload *ReplayWorkflowPayload,
	heartbeat func(done int),
) (*edi.LadingReplay, error) {
	replay, err := r.replays.GetReplay(ctx, ediports.GetLadingReplayRequest{
		TenantInfo: payload.TenantInfo,
		ID:         payload.ReplayID,
	})
	if err != nil {
		return nil, err
	}
	if replay.Status != edi.LadingReplayStatusRunning {
		return replay, nil
	}
	result, runErr := r.run(ctx, payload.TenantInfo, replay, heartbeat)
	now := timeutils.NowUnix()
	replay.FinishedAt = &now
	if runErr != nil {
		replay.Status = edi.LadingReplayStatusFailed
		replay.Error = runErr.Error()
	} else {
		replay.Status = edi.LadingReplayStatusCompleted
		replay.Summary = &result.summary
		replay.Items = result.items
	}
	return r.replays.FinishReplay(ctx, replay)
}

type replayResult struct {
	summary lading.ReplaySummary
	items   []edi.LadingReplayItem
}

func (r *Replayer) run(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	replay *edi.LadingReplay,
	heartbeat func(done int),
) (*replayResult, error) {
	project, err := r.projects.GetProject(ctx, ediports.GetLadingProjectRequest{
		TenantInfo: tenantInfo,
		ID:         replay.ProjectID,
	})
	if err != nil {
		return nil, err
	}
	after, err := r.program(ctx, tenantInfo, project, replay.VersionID)
	if err != nil {
		return nil, err
	}
	before, err := r.program(ctx, tenantInfo, project, replay.AgainstVersionID)
	if err != nil {
		return nil, err
	}
	target := after.Manifest.Map(replay.Map)
	if target == nil {
		return nil, fmt.Errorf("version %d has no map %s", after.Version.Number, replay.Map)
	}
	direction := edi.DocumentDirectionInbound
	if target.Direction == lading.DirectionOutbound {
		direction = edi.DocumentDirectionOutbound
	}
	result := &replayResult{summary: lading.ReplaySummary{Fields: map[string]int{}}}
	cursor := pulid.Nil
	for done := 0; done < maxReplayMessages; {
		messages, listErr := r.messages.ListReplayMessages(ctx, &ediports.ListEDIReplayMessagesRequest{
			TenantInfo:     tenantInfo,
			PartnerID:      project.EDIPartnerID,
			TransactionSet: edi.TransactionSet(target.Set),
			Direction:      direction,
			From:           replay.PeriodStart,
			To:             replay.PeriodEnd,
			AfterID:        cursor,
			Limit:          replayBatch,
		})
		if listErr != nil {
			return nil, listErr
		}
		if len(messages) == 0 {
			break
		}
		cursor = messages[len(messages)-1].ID
		inputs := make([]lading.ReplayInput, 0, len(messages))
		for _, message := range messages {
			input, ok := replayInput(message, direction)
			if ok {
				inputs = append(inputs, input)
			}
		}
		if err = r.batch(ctx, tenantInfo, project, before, after, replay.Map, inputs, result); err != nil {
			return nil, err
		}
		done += len(messages)
		if heartbeat != nil {
			heartbeat(done)
		}
		if len(messages) < replayBatch {
			break
		}
	}
	return result, nil
}

func (r *Replayer) program(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	project *edi.LadingProject,
	versionID pulid.ID,
) (*Program, error) {
	version, err := r.projects.GetVersion(ctx, ediports.GetLadingVersionRequest{
		TenantInfo: tenantInfo,
		ID:         versionID,
	})
	if err != nil {
		return nil, err
	}
	return r.runtime.Version(ctx, tenantInfo, project, version)
}

func replayInput(message *edi.EDIMessage, direction edi.DocumentDirection) (lading.ReplayInput, bool) {
	input := lading.ReplayInput{
		ID:  message.ID.String(),
		Now: time.Unix(message.GeneratedAt, 0).UTC().Format(time.RFC3339),
	}
	if direction == edi.DocumentDirectionInbound {
		if message.RawX12 == "" {
			return input, false
		}
		input.Input = lading.X12Input(message.RawX12)
		return input, true
	}
	if len(message.Document) == 0 {
		return input, false
	}
	input.Input = lading.DocumentInput(message.Document)
	return input, true
}

func (r *Replayer) batch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	project *edi.LadingProject,
	before, after *Program,
	name string,
	inputs []lading.ReplayInput,
	result *replayResult,
) error {
	if len(inputs) == 0 {
		return nil
	}
	lookups, err := r.lookups(ctx, tenantInfo, project.EDIPartnerID, name, inputs, before, after)
	if err != nil {
		return err
	}
	for index := range inputs {
		inputs[index].Lookups = lookups
	}
	settings, err := Settings(project)
	if err != nil {
		return err
	}
	report, err := r.runtime.client.Replay(ctx, &lading.ReplayRequest{
		Before:   before.Encoded.Project,
		After:    after.Encoded.Project,
		Context:  after.Encoded.Context,
		Map:      name,
		Inputs:   inputs,
		Settings: settings,
	})
	if err != nil {
		return err
	}
	fold(&result.summary, &report.Summary)
	for index := range report.Items {
		item := &report.Items[index]
		if item.Same || len(result.items) >= maxReplayItemsKept {
			continue
		}
		result.items = append(result.items, edi.LadingReplayItem{
			MessageID:  pulid.ID(item.ID),
			ReplayItem: *item,
		})
	}
	return nil
}

// lookups answers every lookup either version's map needs for a batch, so both read the same
// mappings.
func (r *Replayer) lookups(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
	name string,
	inputs []lading.ReplayInput,
	programs ...*Program,
) ([]lading.LookupAnswer, error) {
	needs := map[lading.Need]bool{}
	for _, program := range programs {
		for index := range inputs {
			result, err := r.runtime.client.Run(ctx, program.Encoded, &lading.RunRequest{
				Map:   name,
				Input: inputs[index].Input,
				Now:   inputs[index].Now,
				Rules: true,
			})
			if err != nil {
				return nil, err
			}
			for _, need := range result.Needs {
				needs[need] = true
			}
		}
	}
	if len(needs) == 0 {
		return nil, nil
	}
	list := make([]lading.Need, 0, len(needs))
	for need := range needs {
		list = append(list, need)
	}
	return r.runtime.Lookup(ctx, tenantInfo, partnerID, list)
}

func fold(total, batch *lading.ReplaySummary) {
	total.Total += batch.Total
	total.Same += batch.Same
	total.Differ += batch.Differ
	total.NewlyRejected += batch.NewlyRejected
	total.NewlyHeld += batch.NewlyHeld
	total.NewlyDeclined += batch.NewlyDeclined
	total.Failed += batch.Failed
	for field, count := range batch.Fields {
		total.Fields[field] += count
	}
}

type ShadowRunRequest struct {
	TenantInfo pagination.TenantInfo
	Programs   *Programs
	Map        string
	MessageID  pulid.ID
	Set        edi.TransactionSet
	Input      lading.RunInput
	Lookups    []lading.LookupAnswer
	Now        time.Time
}

// Shadow compares the shadow version with the active one on one live message and records the
// result; it never fails the message it compares.
func (r *Replayer) Shadow(ctx context.Context, req *ShadowRunRequest) {
	programs := req.Programs
	if programs == nil || programs.Shadow == nil || programs.Active == nil {
		return
	}
	if programs.Shadow.Manifest.Map(req.Map) == nil {
		return
	}
	settings, err := Settings(programs.Project)
	if err != nil {
		r.l.Warn("failed to read Lading settings for a shadow run", zap.Error(err))
		return
	}
	report, err := r.runtime.client.Replay(ctx, &lading.ReplayRequest{
		Before:  programs.Active.Encoded.Project,
		After:   programs.Shadow.Encoded.Project,
		Context: programs.Active.Encoded.Context,
		Map:     req.Map,
		Inputs: []lading.ReplayInput{{
			ID:      req.MessageID.String(),
			Input:   req.Input,
			Lookups: req.Lookups,
			Now:     instant(req.Now),
		}},
		Settings: settings,
	})
	result := &edi.LadingShadowResult{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		VersionID:      programs.Shadow.Version.ID,
		MessageID:      req.MessageID,
		TransactionSet: req.Set,
	}
	switch {
	case err != nil:
		result.Error = err.Error()
	case len(report.Items) == 0:
		result.Same = true
	default:
		item := report.Items[0]
		result.Same = item.Same
		result.Differences = item.Differences
		result.NewOutcomes = item.NewOutcomes
		result.GoneOutcomes = item.GoneOutcomes
		if item.AfterError != nil {
			result.Error = item.AfterError.String()
		}
	}
	if err = r.replays.RecordShadowResult(ctx, result); err != nil {
		r.l.Warn("failed to record a Lading shadow result",
			zap.String("messageId", req.MessageID.String()),
			zap.Error(err),
		)
	}
}
