package agentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentcontrolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttrustports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/editconflict"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/settingversionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	Repo         tenantports.AgentControlRepository
	Versions     settingversionports.SettingVersionRepository
	AuditService auditports.AuditService
	// Trust promotes the tools that had already earned it when earned
	// autonomy is turned on or its threshold lowered.
	Trust agenttrustports.AgentTrustService `optional:"true"`
}

type Service struct {
	l        *zap.Logger
	db       kernelports.Transactor
	repo     tenantports.AgentControlRepository
	versions settingversionports.SettingVersionRepository
	audit    auditports.AuditService
	trust    agenttrustports.AgentTrustService
	now      func() int64
}

func New(p Params) agentcontrolports.AgentControlService {
	return &Service{
		l:        p.Logger.Named("service.agentcontrol"),
		db:       p.DB,
		repo:     p.Repo,
		versions: p.Versions,
		audit:    p.AuditService,
		trust:    p.Trust,
		now:      timeutils.NowUnix,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*agentcontrol.AgentControl, error) {
	control, err := s.repo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return control, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *agentcontrolports.UpdateAgentControlRequest,
	actor *permissionports.RequestActor,
) (*agentcontrol.AgentControl, error) {
	control, err := s.repo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	previous := *control
	loaded := control.Version
	if req.Version != nil {
		loaded = *req.Version
		control.Version = loaded
	}
	control.ShadowMode = req.ShadowMode
	if req.EarnedAutonomy != nil {
		control.EarnedAutonomy = *req.EarnedAutonomy
	}
	if req.PromotionThreshold != nil {
		control.PromotionThreshold = *req.PromotionThreshold
	}
	if req.PersonMonthlyMessages != nil {
		control.PersonMonthlyMessages = *req.PersonMonthlyMessages
	}
	if req.LearningOff != nil {
		control.LearningOff = *req.LearningOff
	}
	if req.AutoChaptersOff != nil {
		control.AutoChaptersOff = *req.AutoChaptersOff
	}
	applyAgentMoney(control, &req.AgentMoney)

	me := errortypes.NewMultiError()
	consentChanged := s.applyTrainingConsent(control, req.AITrainingConsent, actor, me)
	control.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	updated, err := s.save(ctx, control, actor)
	if err != nil {
		return nil, s.explainConflict(ctx, req.TenantInfo, loaded, err)
	}

	auditActor := actor.AuditActor()
	if err = s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentControl,
		ResourceID:     updated.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(&previous),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, agentControlAuditOptions(ctx, consentChanged, &previous, updated)...); err != nil {
		s.l.Error("failed to log agent control audit", zap.Error(err))
	}

	s.promoteEarned(ctx, &previous, updated, auditActor.UserID)

	return updated, nil
}

// promoteEarned moves up the tools whose streak already met the threshold,
// when this save is what made it count: earned autonomy just turned on, or
// its threshold just lowered. A tool that earned its tier before there was a
// clean approval to promote it on would otherwise wait for one.
func (s *Service) promoteEarned(
	ctx context.Context,
	previous, updated *agentcontrol.AgentControl,
	decidedBy pulid.ID,
) {
	if s.trust == nil || !updated.EarnedAutonomy {
		return
	}
	if previous.EarnedAutonomy && updated.PromotionThreshold >= previous.PromotionThreshold {
		return
	}

	promoted, err := s.trust.PromoteReady(ctx, &agenttrustports.PromoteReadyRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: updated.OrganizationID,
			BuID:  updated.BusinessUnitID,
		},
		Threshold: updated.PromotionThreshold,
		DecidedBy: decidedBy,
	})
	if err != nil {
		s.l.Warn("could not promote the tools that had earned it", zap.Error(err))
		return
	}
	if len(promoted) > 0 {
		s.l.Info("tools promoted as earned autonomy took effect", zap.Int("tools", len(promoted)))
	}
}

// save writes the controls and the version they become together.
func (s *Service) save(
	ctx context.Context,
	control *agentcontrol.AgentControl,
	actor *permissionports.RequestActor,
) (*agentcontrol.AgentControl, error) {
	var updated *agentcontrol.AgentControl
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		updated, txErr = s.repo.Update(txCtx, control)
		if txErr != nil {
			return txErr
		}
		return editconflict.Record(txCtx, s.versions, &editconflict.RecordRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: updated.OrganizationID,
				BuID:  updated.BusinessUnitID,
			},
			Kind:      settingversion.KindAgentControl,
			SubjectID: updated.ID,
			Version:   updated.Version,
			Snapshot:  updated,
			AuthorID:  actor.AuditActor().UserID,
		})
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) explainConflict(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	loaded int64,
	cause error,
) error {
	if !errortypes.IsVersionMismatchError(cause) {
		return cause
	}
	current, err := s.repo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return cause
	}

	return editconflict.Explain(
		ctx,
		s.versions,
		s.l,
		&editconflict.ExplainRequest[agentcontrol.AgentControl]{
			TenantInfo:     tenantInfo,
			Kind:           settingversion.KindAgentControl,
			SubjectID:      current.ID,
			Loaded:         loaded,
			Current:        current,
			CurrentVersion: current.Version,
			UpdatedAt:      current.UpdatedAt,
			Rules:          agentcontrol.AgentControlChangeRules,
			Cause:          cause,
		},
	)
}

func (s *Service) applyTrainingConsent(
	control *agentcontrol.AgentControl,
	consent *bool,
	actor *permissionports.RequestActor,
	me *errortypes.MultiError,
) bool {
	if consent == nil || *consent == control.AITrainingConsent {
		return false
	}

	userID := actor.PersonUserID()
	if userID.IsNil() {
		me.Add(
			"aiTrainingConsent",
			errortypes.ErrInvalid,
			"Training consent can only be changed by a signed-in person",
		)
		return false
	}

	return control.SetAITrainingConsent(*consent, userID, s.now())
}

func applyAgentMoney(
	control *agentcontrol.AgentControl,
	update *agentcontrolports.AgentMoneyLimitsUpdate,
) {
	if update.CallCeilingMinor != nil {
		control.AgentMoneyCallCeilingMinor = *update.CallCeilingMinor
	}
	if update.RunCalls != nil {
		control.AgentMoneyRunCalls = *update.RunCalls
	}
	if update.RunAmountMinor != nil {
		control.AgentMoneyRunAmountMinor = *update.RunAmountMinor
	}
	if update.DayCalls != nil {
		control.AgentMoneyDayCalls = *update.DayCalls
	}
	if update.DayAmountMinor != nil {
		control.AgentMoneyDayAmountMinor = *update.DayAmountMinor
	}
}

func agentControlAuditOptions(
	ctx context.Context,
	consentChanged bool,
	previous, updated *agentcontrol.AgentControl,
) []auditports.LogOption {
	moneyChanged := agentMoneyChanged(previous, updated)
	options := []auditports.LogOption{
		auditservice.WithComment(agentControlAuditComment(consentChanged, moneyChanged, updated)),
		auditservice.WithRequest(ctx),
	}
	if moneyChanged {
		options = append(options, auditservice.WithCritical())
	}

	return options
}

func agentMoneyChanged(previous, updated *agentcontrol.AgentControl) bool {
	return previous.AgentMoneyCallCeilingMinor != updated.AgentMoneyCallCeilingMinor ||
		previous.AgentMoneyRunCalls != updated.AgentMoneyRunCalls ||
		previous.AgentMoneyRunAmountMinor != updated.AgentMoneyRunAmountMinor ||
		previous.AgentMoneyDayCalls != updated.AgentMoneyDayCalls ||
		previous.AgentMoneyDayAmountMinor != updated.AgentMoneyDayAmountMinor
}

func agentControlAuditComment(
	consentChanged, moneyChanged bool,
	control *agentcontrol.AgentControl,
) string {
	switch {
	case consentChanged && control.AITrainingConsent:
		return "AI training consent granted"
	case consentChanged:
		return "AI training consent withdrawn"
	case moneyChanged:
		return "Agent money limits changed"
	default:
		return "Agent control updated"
	}
}
