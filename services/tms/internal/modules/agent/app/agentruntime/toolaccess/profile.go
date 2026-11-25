package toolaccess

import (
	"context"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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
		held = append(held, "delegate_task")
	}

	return profile{Mode: mode, Held: held}
}

func (s *Access) profileHolds(
	req *agentruntimeports.RunRequest,
	mode ProfileMode,
	name string,
) bool {
	switch name {
	case "agent_open_page":
		return req.Delegation == nil
	case "agent_raise_exception":
		return req.RunID.IsNotNil()
	case "agent_guide_profile":
		return mode == ProfileConversation && req.Actor.PersonUserID().IsNotNil()
	case "agent_create_schedule":
		return mode == ProfileConversation && req.Actor.PersonUserID().IsNotNil() &&
			req.ThreadID.IsNotNil()
	}
	if mode != ProfileFanOut {
		return true
	}

	return s.LooksUp(name)
}

func (s *Access) LooksUp(name string) bool {
	tool, ok := s.queryTools.Get(name)

	return ok && agenttoolports.EffectOf(tool) == agenttooltrust.ToolEffectLookup
}

func reachesBeyond(req *agentruntimeports.RunRequest, mode ProfileMode) bool {
	return mode == ProfileConversation && req.ThreadID.IsNotNil() &&
		req.Actor != nil && req.Actor.PrincipalType == permissionports.PrincipalTypeUser
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
