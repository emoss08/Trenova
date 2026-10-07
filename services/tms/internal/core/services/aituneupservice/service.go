package aituneupservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	agentPageSize = 200
	maxAgents     = 2000
	secondsPerDay = 24 * 60 * 60
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	Repo             repositories.AITuneUpRepository
	Freshness        repositories.AITuneUpFreshness
	Control          repositories.AgentControlRepository
	Definitions      repositories.AgentDefinitionRepository
	AgentDefinitions services.AgentDefinitionService
	Providers        repositories.AIProviderRepository
	ProviderService  services.AIProviderService
	Trust            services.AgentTrustService
	Shadow           services.AgentShadowService
	Usage            repositories.AIUsageRepository
}

type Service struct {
	l                *zap.Logger
	repo             repositories.AITuneUpRepository
	freshness        repositories.AITuneUpFreshness
	control          repositories.AgentControlRepository
	definitions      repositories.AgentDefinitionRepository
	agentDefinitions services.AgentDefinitionService
	providers        repositories.AIProviderRepository
	providerService  services.AIProviderService
	trust            services.AgentTrustService
	shadow           services.AgentShadowService
	usage            repositories.AIUsageRepository
	computing        singleflight.Group
	now              func() int64
}

var _ services.AITuneUpService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:                p.Logger.Named("service.aituneup"),
		repo:             p.Repo,
		freshness:        p.Freshness,
		control:          p.Control,
		definitions:      p.Definitions,
		agentDefinitions: p.AgentDefinitions,
		providers:        p.Providers,
		providerService:  p.ProviderService,
		trust:            p.Trust,
		shadow:           p.Shadow,
		usage:            p.Usage,
		now:              timeutils.NowUnix,
	}
}

func (s *Service) List(ctx context.Context, tenantInfo pagination.TenantInfo) (*services.AITuneUpList, error) {
	s.ensureFresh(ctx, tenantInfo)

	stored, err := s.repo.List(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now()
	visible := make([]*aituneup.TuneUp, 0, len(stored))
	var computedAt *int64
	for _, tuneUp := range stored {
		if computedAt == nil || tuneUp.ComputedAt > *computedAt {
			at := tuneUp.ComputedAt
			computedAt = &at
		}
		if tuneUp.Visible(now) {
			visible = append(visible, tuneUp)
		}
	}

	items, err := s.describe(ctx, tenantInfo, visible)
	if err != nil {
		return nil, err
	}
	return &services.AITuneUpList{Items: items, ComputedAt: computedAt, WindowDays: aituneup.WindowDays}, nil
}

func (s *Service) Get(ctx context.Context, req repositories.GetAITuneUpRequest) (*aituneup.TuneUp, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ensureFresh(ctx context.Context, tenantInfo pagination.TenantInfo) {
	fresh, err := s.freshness.Fresh(ctx, tenantInfo)
	if err != nil {
		s.l.Warn("could not read whether tune-ups are fresh; serving what is stored", zap.Error(err))
		return
	}
	if fresh {
		return
	}
	key := tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String()
	_, err, _ = s.computing.Do(key, func() (any, error) {
		return s.Compute(ctx, tenantInfo)
	})
	if err != nil {
		s.l.Warn("could not compute tune-ups on read; serving what is stored",
			zap.String("organization", tenantInfo.OrgID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) Compute(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error) {
	now := s.now()
	inputs, err := s.inputs(ctx, tenantInfo, now)
	if err != nil {
		return 0, err
	}

	candidates := aituneup.Suggest(inputs)
	existing, err := s.repo.List(ctx, tenantInfo)
	if err != nil {
		return 0, err
	}

	plan := aituneup.Reconcile(existing, candidates, now)
	if err = s.repo.Save(ctx, tenantInfo, &plan); err != nil {
		return 0, err
	}

	if err = s.freshness.Mark(ctx, tenantInfo, now, aituneup.FreshForSeconds*time.Second); err != nil {
		s.l.Warn("could not mark tune-ups computed", zap.Error(err))
	}
	return len(candidates), nil
}

func (s *Service) inputs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (*aituneup.Inputs, error) {
	control, err := s.control.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	agents, err := s.allAgents(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	enabled, err := s.providers.ListEnabled(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	since := now - aituneup.WindowDays*secondsPerDay
	usage, err := s.usage.ProviderTaskTotals(ctx, repositories.AIUsageProviderTaskRequest{
		TenantInfo: tenantInfo,
		Since:      since,
	})
	if err != nil {
		return nil, err
	}

	inputs := &aituneup.Inputs{
		Now:            now,
		EarnedAutonomy: control.EarnedAutonomy,
		Paused:         control.ShadowMode,
		Agents:         agents,
		Enabled:        enabled,
		Usage:          usageRows(usage),
		Shadow:         map[pulid.ID]*agentshadow.Report{},
	}

	if !control.EarnedAutonomy {
		ready, readyErr := s.trust.PromotionCandidates(ctx, &services.PromoteReadyRequest{
			TenantInfo: tenantInfo,
			Threshold:  control.PromotionThreshold,
		})
		if readyErr != nil {
			return nil, readyErr
		}
		inputs.Ready = readyTools(ready)
	}

	if !control.ShadowMode {
		for _, definition := range agents {
			if !definition.Enabled || !definition.ShadowMode {
				continue
			}
			report, reportErr := s.shadow.Report(ctx, &services.AgentShadowReportRequest{
				TenantInfo: tenantInfo,
				AgentID:    definition.ID,
				Days:       aituneup.WindowDays,
			})
			if reportErr != nil {
				return nil, reportErr
			}
			inputs.Shadow[definition.ID] = report
		}
	}

	return inputs, nil
}

func (s *Service) allAgents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agentdefinition.Definition, error) {
	agents := make([]*agentdefinition.Definition, 0, agentPageSize)
	for offset := 0; offset < maxAgents; offset += agentPageSize {
		page, err := s.definitions.List(ctx, &repositories.ListAgentDefinitionRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: agentPageSize, Offset: offset},
			},
		})
		if err != nil {
			return nil, err
		}
		agents = append(agents, page.Items...)
		if len(page.Items) < agentPageSize {
			break
		}
	}
	return agents, nil
}

func usageRows(rows []repositories.AIUsageProviderTaskTotals) []aituneup.ProviderTaskUsage {
	out := make([]aituneup.ProviderTaskUsage, 0, len(rows))
	for idx := range rows {
		row := &rows[idx]
		out = append(out, aituneup.ProviderTaskUsage{
			ProviderID: row.ProviderID,
			Task:       row.Task,
			Calls:      row.Calls,
			Failed:     row.Failed,
			Rescued:    row.Rescued,
		})
	}
	return out
}

func readyTools(promotions []services.ToolPromotion) []aituneup.ReadyTool {
	out := make([]aituneup.ReadyTool, 0, len(promotions))
	for idx := range promotions {
		promotion := &promotions[idx]
		out = append(out, aituneup.ReadyTool{
			AgentDefinitionID: promotion.AgentDefinitionID,
			ToolName:          promotion.ToolName,
			From:              promotion.From,
			To:                promotion.To,
			Streak:            promotion.Streak,
			Approvals:         promotion.Approvals,
			Rejections:        promotion.Rejections,
			Since:             promotion.TrackedSince,
		})
	}
	return out
}

func (s *Service) describe(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tuneUps []*aituneup.TuneUp,
) ([]*services.AITuneUpView, error) {
	if len(tuneUps) == 0 {
		return []*services.AITuneUpView{}, nil
	}

	agentIDs := make([]pulid.ID, 0, len(tuneUps))
	needsProviders := false
	for _, tuneUp := range tuneUps {
		if tuneUp.AgentDefinitionID.IsNotNil() {
			agentIDs = append(agentIDs, tuneUp.AgentDefinitionID)
		}
		if tuneUp.ProviderID.IsNotNil() || tuneUp.OtherProviderID.IsNotNil() {
			needsProviders = true
		}
	}

	agents := map[pulid.ID]*agentdefinition.Definition{}
	if len(agentIDs) > 0 {
		found, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
			IDs:        agentIDs,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return nil, err
		}
		for _, definition := range found {
			agents[definition.ID] = definition
		}
	}

	providers := map[pulid.ID]*aiprovider.Provider{}
	if needsProviders {
		found, err := s.providers.ListOrdered(ctx, tenantInfo)
		if err != nil {
			return nil, err
		}
		for _, provider := range found {
			providers[provider.ID] = provider
		}
	}

	views := make([]*services.AITuneUpView, 0, len(tuneUps))
	for _, tuneUp := range tuneUps {
		view, ok := viewOf(tuneUp, agents, providers)
		if ok {
			views = append(views, view)
		}
	}
	return views, nil
}

func viewOf(
	tuneUp *aituneup.TuneUp,
	agents map[pulid.ID]*agentdefinition.Definition,
	providers map[pulid.ID]*aiprovider.Provider,
) (*services.AITuneUpView, bool) {
	view := &services.AITuneUpView{TuneUp: tuneUp}
	if tuneUp.AgentDefinitionID.IsNotNil() {
		definition, ok := agents[tuneUp.AgentDefinitionID]
		if !ok {
			return nil, false
		}
		view.Agent = &services.AITuneUpAgent{
			ID:     definition.ID,
			Name:   definition.Name,
			Icon:   definition.Icon,
			Accent: definition.Accent,
		}
	}
	if tuneUp.ProviderID.IsNotNil() {
		provider, ok := providers[tuneUp.ProviderID]
		if !ok {
			return nil, false
		}
		view.Provider = providerOf(provider)
	}
	if tuneUp.OtherProviderID.IsNotNil() {
		provider, ok := providers[tuneUp.OtherProviderID]
		if !ok {
			return nil, false
		}
		view.OtherProvider = providerOf(provider)
	}
	return view, true
}

func providerOf(provider *aiprovider.Provider) *services.AITuneUpProvider {
	return &services.AITuneUpProvider{ID: provider.ID, Name: provider.Name, Kind: provider.Kind}
}
