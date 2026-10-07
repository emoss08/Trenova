package agentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/settingversion"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/services/editconflict"
	"github.com/uptrace/bun"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const SystemKeyBillingException = "billing_exception"

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           ports.DBConnection
	Repo         repositories.AgentControlRepository
	Versions     repositories.SettingVersionRepository
	Definitions  repositories.AgentDefinitionRepository
	AuditService services.AuditService
}

type Service struct {
	l           *zap.Logger
	db          ports.DBConnection
	repo        repositories.AgentControlRepository
	versions    repositories.SettingVersionRepository
	definitions repositories.AgentDefinitionRepository
	audit       services.AuditService
	now         func() int64
}

func New(p Params) services.AgentControlService {
	return &Service{
		l:           p.Logger.Named("service.agentcontrol"),
		db:          p.DB,
		repo:        p.Repo,
		versions:    p.Versions,
		definitions: p.Definitions,
		audit:       p.AuditService,
		now:         timeutils.NowUnix,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	control, err := s.repo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	definition, err := s.billingDefinition(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	applyLegacyFields(control, definition)

	return control, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *services.UpdateAgentControlRequest,
	actor *services.RequestActor,
) (*tenant.AgentControl, error) {
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

	definition, err := s.applyLegacyUpdate(ctx, req)
	if err != nil {
		return nil, err
	}
	applyLegacyFields(updated, definition)

	auditActor := actor.AuditActor()
	if err = s.audit.LogAction(&services.LogActionParams{
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
	}, auditservice.WithComment(agentControlAuditComment(consentChanged, updated))); err != nil {
		s.l.Error("failed to log agent control audit", zap.Error(err))
	}

	return updated, nil
}

// save writes the controls and the version they become together.
func (s *Service) save(
	ctx context.Context,
	control *tenant.AgentControl,
	actor *services.RequestActor,
) (*tenant.AgentControl, error) {
	var updated *tenant.AgentControl
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		updated, txErr = s.repo.Update(txCtx, control)
		if txErr != nil {
			return txErr
		}
		return editconflict.Record(txCtx, s.versions, &editconflict.RecordRequest{
			TenantInfo: pagination.TenantInfo{OrgID: updated.OrganizationID, BuID: updated.BusinessUnitID},
			Kind:       settingversion.KindAgentControl,
			SubjectID:  updated.ID,
			Version:    updated.Version,
			Snapshot:   updated,
			AuthorID:   actor.AuditActor().UserID,
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

	return editconflict.Explain(ctx, s.versions, s.l, &editconflict.ExplainRequest[tenant.AgentControl]{
		TenantInfo:     tenantInfo,
		Kind:           settingversion.KindAgentControl,
		SubjectID:      current.ID,
		Loaded:         loaded,
		Current:        current,
		CurrentVersion: current.Version,
		UpdatedAt:      current.UpdatedAt,
		Rules:          tenant.AgentControlChangeRules,
		Cause:          cause,
	})
}

func (s *Service) applyTrainingConsent(
	control *tenant.AgentControl,
	consent *bool,
	actor *services.RequestActor,
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

func agentControlAuditComment(consentChanged bool, control *tenant.AgentControl) string {
	switch {
	case consentChanged && control.AITrainingConsent:
		return "AI training consent granted"
	case consentChanged:
		return "AI training consent withdrawn"
	default:
		return "Agent control updated"
	}
}

func (s *Service) billingDefinition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*agentdefinition.Definition, error) {
	definition, err := s.definitions.GetBySystemKey(
		ctx,
		repositories.GetAgentDefinitionBySystemKeyRequest{
			SystemKey:  SystemKeyBillingException,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return definition, nil
}

func (s *Service) applyLegacyUpdate(
	ctx context.Context,
	req *services.UpdateAgentControlRequest,
) (*agentdefinition.Definition, error) {
	definition, err := s.billingDefinition(ctx, req.TenantInfo)
	if err != nil || definition == nil {
		return definition, err
	}
	if req.BillingAgentEnabled == nil && req.DecisionTimeoutSeconds == nil {
		return definition, nil
	}

	if req.BillingAgentEnabled != nil {
		definition.Enabled = *req.BillingAgentEnabled
	}
	if req.DecisionTimeoutSeconds != nil {
		definition.DecisionTimeoutSeconds = *req.DecisionTimeoutSeconds
	}

	me := errortypes.NewMultiError()
	definition.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	return s.definitions.Update(ctx, definition)
}

func applyLegacyFields(control *tenant.AgentControl, definition *agentdefinition.Definition) {
	if definition == nil {
		control.BillingAgentEnabled = false
		control.DecisionTimeoutSeconds = agentdefinition.DefaultDecisionTimeoutSeconds
		return
	}

	control.BillingAgentEnabled = definition.Enabled
	control.DecisionTimeoutSeconds = definition.DecisionTimeoutSeconds
}
