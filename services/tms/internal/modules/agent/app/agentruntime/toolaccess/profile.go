package toolaccess

import (
	"context"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/usableagents"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/zap"
)

type ProfileMode string

const (
	ProfileConversation = ProfileMode("conversation")
	ProfileUnattended   = ProfileMode("unattended")
	ProfileDelegated    = ProfileMode("delegated")
	ProfileFanOut       = ProfileMode("fan_out")
)

type profile struct {
	Mode   ProfileMode
	Held   []string
	Beyond map[string]agent.AutonomyTier
}

func profileModeOf(req *agentruntimeports.RunRequest) ProfileMode {
	switch {
	case req.Delegation != nil && req.Delegation.ReadOnly:
		return ProfileFanOut
	case req.Delegation != nil:
		return ProfileDelegated
	case req.Unattended:
		return ProfileUnattended
	default:
		return ProfileConversation
	}
}

func (s *Access) ProfileFor(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	granted []string,
) profile {
	mode := profileModeOf(req)
	held := withGrants(s.HeldTools(req.Definition), granted)
	held = slices.DeleteFunc(held, func(name string) bool {
		return !s.profileHolds(req, mode, name)
	})
	if req.MayDelegate() {
		held = append(held, runtimetools.DelegateTaskName)
	}

	return profile{
		Mode:   mode,
		Held:   held,
		Beyond: s.beyondFor(ctx, req, mode, held),
	}
}

func (s *Access) profileHolds(
	req *agentruntimeports.RunRequest,
	mode ProfileMode,
	name string,
) bool {
	switch name {
	case agentdefinition.CoreToolOpenPage:
		return req.Delegation == nil
	case agentdefinition.CoreToolRaiseException:
		return req.RunID.IsNotNil()
	case agentdefinition.CoreToolGuideProfile:
		return mode == ProfileConversation && req.Actor.PersonUserID().IsNotNil()
	case agentdefinition.CoreToolSchedule:
		return mode == ProfileConversation && req.Actor.PersonUserID().IsNotNil() &&
			req.ThreadID.IsNotNil()
	}
	if mode != ProfileFanOut {
		return true
	}

	return s.LooksUp(name)
}

func (s *Access) LooksUp(name string) bool {
	if s.queryTools == nil {
		return false
	}
	tool, ok := s.queryTools.Get(name)

	return ok && agenttoolports.EffectOf(tool) == agenttooltrust.ToolEffectLookup
}

func reachesBeyond(req *agentruntimeports.RunRequest, mode ProfileMode) bool {
	return mode == ProfileConversation && !req.Inline && req.ThreadID.IsNotNil() &&
		req.Actor != nil && req.Actor.PrincipalType == permissionports.PrincipalTypeUser
}

func (s *Access) beyondFor(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	mode ProfileMode,
	held []string,
) map[string]agent.AutonomyTier {
	if !reachesBeyond(req, mode) || s.definitions == nil {
		return nil
	}
	others, err := usableagents.Others(ctx, &usableagents.OthersRequest{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Actor:       req.Actor,
		Self:        req.Definition.ID,
	})
	if err != nil {
		s.logger.Warn(
			"could not read the person's other agents; the turn reaches only its own tools",
			zap.Error(err),
		)

		return nil
	}

	beyond := make(map[string]agent.AutonomyTier, 64)
	for _, other := range others {
		for _, name := range s.HeldTools(other) {
			if agentdefinition.IsCoreTool(name) || slices.Contains(held, name) ||
				slices.Contains(req.Definition.DisabledToolNames, name) {
				continue
			}
			tier, known := s.holderTier(other, name)
			if !known {
				continue
			}
			if current, seen := beyond[name]; !seen || tier.Above(current) {
				beyond[name] = tier
			}
		}
	}
	if len(beyond) == 0 {
		return nil
	}

	return beyond
}

func (s *Access) holderTier(
	holder *agentdefinition.Definition,
	name string,
) (agent.AutonomyTier, bool) {
	if _, ok := s.queryTools.Get(name); ok {
		return agent.TierAutoExecute, true
	}
	tool, ok := s.actionTools.Get(name)
	if !ok || holder.SimulationMode {
		return "", false
	}

	return agenttoolpolicy.StaticTier(holder, tool.Policy()), true
}

func BeyondNames(beyond map[string]agent.AutonomyTier) []string {
	if len(beyond) == 0 {
		return nil
	}

	return slices.Sorted(maps.Keys(beyond))
}

func (s *Access) Reachable(req *agentruntimeports.RunRequest, beyond agent.AutonomyTier) bool {
	return beyond.IsValid() && reachesBeyond(req, profileModeOf(req))
}
