package archivejobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/cloud/app/archiveengine"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/jobs/archivejobs/archivejobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	progressEvery       = 5 * time.Second
	errorNotRestoring   = "CloudTenantNotRestoring"
	errorNoNightlyGrant = "CloudTenantNoNightlyGrant"
	nightlyPageSize     = 100
)

type ActivitiesParams struct {
	fx.In

	Lifecycle *tenantlifecycle.Service
	Engine    *archiveengine.Engine
	Buckets   archiveports.Buckets
	Slots     kernelports.HeavyJobSlots
	Managed   *managedconfig.Config
	Metrics   *metrics.Registry
	Clock     timeutils.Clock
	Logger    *zap.Logger
}

type Activities struct {
	lifecycle *tenantlifecycle.Service
	engine    *archiveengine.Engine
	buckets   archiveports.Buckets
	slots     kernelports.HeavyJobSlots
	managed   *managedconfig.Config
	metrics   *platformmetrics.Tenancy
	clock     timeutils.Clock
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		lifecycle: p.Lifecycle,
		engine:    p.Engine,
		buckets:   p.Buckets,
		slots:     p.Slots,
		managed:   p.Managed,
		metrics:   metrics.TenancyFrom(p.Metrics),
		clock:     p.Clock,
		l:         p.Logger.Named("cloud.archive-jobs"),
	}
}

func heartbeat(ctx context.Context) func() {
	return func() {
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx)
		}
	}
}

func (a *Activities) reporter(ctx context.Context, commandID string) archiveengine.Reporter {
	var (
		mu    sync.Mutex
		last  time.Time
		phase tenantcommand.Phase
	)
	return func(p tenantcommand.Progress) {
		mu.Lock()
		due := p.Phase != phase || time.Since(last) >= progressEvery
		if due {
			last, phase = time.Now(), p.Phase
		}
		mu.Unlock()
		if !due {
			return
		}
		if err := a.lifecycle.ReportProgress(ctx, commandID, &p); err != nil {
			a.l.Debug("cannot record a command's progress", zap.String("commandId", commandID),
				zap.Error(err))
		}
		heartbeat(ctx)()
	}
}

func (a *Activities) bounded(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, a.managed.Archive.Duration())
}

type Outcome struct {
	Settled bool                  `json:"settled"`
	Failed  bool                  `json:"failed"`
	Code    string                `json:"code,omitempty"`
	Reason  string                `json:"reason,omitempty"`
	SHA256  string                `json:"sha256,omitempty"`
	Rows    int64                 `json:"rows,omitempty"`
	Bytes   int64                 `json:"bytes,omitempty"`
	Purpose tenantarchive.Purpose `json:"purpose,omitempty"`
	Tenant  *TenantView           `json:"tenant,omitempty"`
	owner   pulid.ID
}

type TenantView struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func (a *Activities) RunArchiveActivity(
	ctx context.Context,
	payload *archivejobscontract.CommandPayload,
) (*Outcome, error) {
	cmd, access, err := a.lifecycle.CommandAccess(ctx, payload.CommandID)
	if err != nil {
		return nil, err
	}
	if access == nil {
		return &Outcome{Settled: true}, nil
	}
	t, err := a.lifecycle.TenantFor(ctx, cmd.InstanceID)
	if err != nil {
		return nil, err
	}
	release, err := a.slots.Hold(ctx, kernelports.HeavyJobArchive)
	if err != nil {
		return nil, fmt.Errorf("wait for an archive slot: %w", err)
	}
	defer release()
	if err = a.lifecycle.BeginCommand(ctx, cmd.CommandID); err != nil {
		return nil, err
	}
	bounded, cancel := a.bounded(ctx)
	defer cancel()
	result, err := a.engine.Export(bounded, &archiveengine.ExportRequest{
		Purpose:    tenantarchive.Purpose(cmd.Purpose),
		InstanceID: cmd.InstanceID,
		Tenant: dbscope.Tenant{
			OrganizationID: t.OrganizationID,
			BusinessUnitID: t.BusinessUnitID,
		},
		Access:   access,
		Progress: a.reporter(ctx, cmd.CommandID),
		Beat:     heartbeat(ctx),
	})
	if refusal, ok := tenantarchive.RefusalOf(err); ok {
		a.metrics.RecordArchiveRefusal(cmd.Purpose, string(refusal.Code))
		return a.fail(ctx, cmd.CommandID, refusal, a.lifecycle.FailCommand)
	}
	if err != nil {
		return nil, err
	}
	if err = a.lifecycle.FinishArchive(ctx, &tenantlifecycle.ArchiveResult{
		CommandID: cmd.CommandID,
		Location:  result.Location,
		SHA256:    result.SHA256,
		SizeBytes: result.SizeBytes,
		TakenAt:   result.Manifest.TakenAt,
	}); err != nil {
		return nil, err
	}
	return &Outcome{
		SHA256:  result.SHA256,
		Bytes:   result.SizeBytes,
		Purpose: result.Manifest.Purpose,
	}, nil
}

type failFunc func(ctx context.Context, commandID string, refusal *tenantarchive.RefusalError) error

type restoreFail func(ctx context.Context, commandID string, refusal *tenantarchive.RefusalError, touched bool) error

func untouched(fail restoreFail) failFunc {
	return func(ctx context.Context, commandID string, refusal *tenantarchive.RefusalError) error {
		return fail(ctx, commandID, refusal, false)
	}
}

func incomplete(fail restoreFail) failFunc {
	return func(ctx context.Context, commandID string, refusal *tenantarchive.RefusalError) error {
		return fail(ctx, commandID, refusal, true)
	}
}

func (a *Activities) fail(
	ctx context.Context,
	commandID string,
	refusal *tenantarchive.RefusalError,
	record failFunc,
) (*Outcome, error) {
	if err := record(ctx, commandID, refusal); err != nil {
		return nil, err
	}
	a.l.Info("refused a tenant archive command",
		zap.String("commandId", commandID),
		zap.String("code", string(refusal.Code)),
	)
	return &Outcome{Failed: true, Code: string(refusal.Code), Reason: refusal.Reason}, nil
}

type FailInput struct {
	Payload *archivejobscontract.CommandPayload `json:"payload"`
	Kind    tenantcommand.Kind                  `json:"kind"`
	Reason  string                              `json:"reason"`
	Touched bool                                `json:"touched"`
}

func (a *Activities) FailCommandActivity(ctx context.Context, input *FailInput) error {
	refusal := tenantlifecycle.Failure(errors.New(input.Reason))
	switch input.Kind {
	case tenantcommand.KindRestore:
		return a.lifecycle.FailRestore(ctx, input.Payload.CommandID, refusal, input.Touched)
	case tenantcommand.KindImport:
		return a.lifecycle.FailImport(ctx, input.Payload.CommandID, refusal)
	case tenantcommand.KindArchive:
		return a.lifecycle.FailCommand(ctx, input.Payload.CommandID, refusal)
	default:
		return a.lifecycle.FailCommand(ctx, input.Payload.CommandID, refusal)
	}
}

func (a *Activities) CheckRestoreActivity(
	ctx context.Context,
	payload *archivejobscontract.CommandPayload,
) (bool, error) {
	_, err := a.lifecycle.Restoring(ctx, payload.CommandID)
	switch {
	case errors.Is(err, tenantlifecycle.ErrTenantState):
		return false, nil
	case errors.Is(err, cloudtenantports.ErrCommandNotFound):
		return false, temporal.NewNonRetryableApplicationError(err.Error(), errorNotRestoring, err)
	case err != nil:
		return false, err
	}
	return true, nil
}

func (a *Activities) VerifyRestoreActivity(
	ctx context.Context,
	payload *archivejobscontract.CommandPayload,
) (*Outcome, error) {
	cmd, access, err := a.lifecycle.CommandAccess(ctx, payload.CommandID)
	if err != nil {
		return nil, err
	}
	if access == nil {
		return &Outcome{Settled: true}, nil
	}
	if err = a.lifecycle.BeginCommand(ctx, cmd.CommandID); err != nil {
		return nil, err
	}
	tenant := payload.Tenant()
	bounded, cancel := a.bounded(ctx)
	defer cancel()
	manifest, err := a.engine.Verify(bounded, &archiveengine.ImportRequest{
		InstanceID:   cmd.InstanceID,
		ExpectSHA256: cmd.ExpectSHA256,
		Access:       access,
		Tenant:       &tenant,
		Restoring:    true,
		Beat:         heartbeat(ctx),
	})
	if refusal, ok := tenantarchive.RefusalOf(err); ok {
		return a.fail(ctx, cmd.CommandID, refusal, untouched(a.lifecycle.FailRestore))
	}
	if err != nil {
		return nil, err
	}
	return &Outcome{Rows: manifestRows(manifest)}, nil
}

func manifestRows(m *tenantarchive.Manifest) int64 {
	var rows int64
	for i := range m.Tables {
		rows += m.Tables[i].Rows
	}
	return rows
}

func (a *Activities) RunRestoreLoadActivity(
	ctx context.Context,
	payload *archivejobscontract.CommandPayload,
) (*Outcome, error) {
	cmd, access, err := a.lifecycle.CommandAccess(ctx, payload.CommandID)
	if err != nil {
		return nil, err
	}
	if access == nil {
		return &Outcome{Settled: true}, nil
	}
	tenant := payload.Tenant()
	outcome, err := a.load(ctx, cmd, access, &tenant, "", incomplete(a.lifecycle.FailRestore))
	if err != nil || outcome.Failed {
		return outcome, err
	}
	if err = a.lifecycle.FinishRestore(ctx, &tenantlifecycle.RestoreResult{
		CommandID: cmd.CommandID,
		SHA256:    outcome.SHA256,
		Rows:      outcome.Rows,
	}); err != nil {
		return nil, err
	}
	return outcome, nil
}

func (a *Activities) RunImportActivity(
	ctx context.Context,
	payload *archivejobscontract.CommandPayload,
) (*Outcome, error) {
	cmd, access, err := a.lifecycle.CommandAccess(ctx, payload.CommandID)
	if err != nil {
		return nil, err
	}
	if access == nil {
		return &Outcome{Settled: true}, nil
	}
	t, err := a.lifecycle.TenantFor(ctx, cmd.InstanceID)
	if err != nil {
		return nil, err
	}
	outcome, err := a.load(
		ctx,
		cmd,
		access,
		nil,
		t.Desired.FirstAdministrator.Email,
		a.lifecycle.FailImport,
	)
	if err != nil || outcome.Failed {
		return outcome, err
	}
	if err = a.lifecycle.FinishImport(ctx, &tenantlifecycle.ImportResult{
		CommandID: cmd.CommandID,
		Tenant: dbscope.Tenant{
			OrganizationID: outcome.Tenant.OrganizationID,
			BusinessUnitID: outcome.Tenant.BusinessUnitID,
		},
		Owner:  outcome.owner,
		SHA256: outcome.SHA256,
		Rows:   outcome.Rows,
	}); err != nil {
		return nil, err
	}
	return outcome, nil
}

func (a *Activities) load(
	ctx context.Context,
	cmd *tenantcommand.Command,
	access *tenantarchive.Access,
	tenant *dbscope.Tenant,
	ownerEmail string,
	fail failFunc,
) (*Outcome, error) {
	release, err := a.slots.Hold(ctx, kernelports.HeavyJobArchive)
	if err != nil {
		return nil, fmt.Errorf("wait for an archive slot: %w", err)
	}
	defer release()
	if err = a.lifecycle.BeginCommand(ctx, cmd.CommandID); err != nil {
		return nil, err
	}
	bounded, cancel := a.bounded(ctx)
	defer cancel()
	result, err := a.engine.Import(bounded, &archiveengine.ImportRequest{
		InstanceID:   cmd.InstanceID,
		ExpectSHA256: cmd.ExpectSHA256,
		Access:       access,
		Tenant:       tenant,
		Restoring:    tenant != nil,
		OwnerEmail:   ownerEmail,
		Progress:     a.reporter(ctx, cmd.CommandID),
		Beat:         heartbeat(ctx),
	})
	if refusal, ok := tenantarchive.RefusalOf(err); ok {
		return a.fail(ctx, cmd.CommandID, refusal, fail)
	}
	if err != nil {
		return nil, err
	}
	return &Outcome{
		SHA256: result.ManifestSHA256,
		Rows:   result.Rows,
		Tenant: &TenantView{
			OrganizationID: result.Tenant.OrganizationID,
			BusinessUnitID: result.Tenant.BusinessUnitID,
		},
		owner: result.Owner,
	}, nil
}
