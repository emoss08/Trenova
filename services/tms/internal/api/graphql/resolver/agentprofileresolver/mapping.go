package agentprofileresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
)

func profileToModel(view *agentprofileports.ProfileView) *gqlmodel.AgentProfile {
	if view == nil {
		return &gqlmodel.AgentProfile{Guidance: []*agentprofile.Guidance{}}
	}

	guidance := view.Guidance
	if guidance == nil {
		guidance = []*agentprofile.Guidance{}
	}

	return &gqlmodel.AgentProfile{
		Subject:       view.Ref.Subject,
		Current:       view.Current,
		LatestFailure: view.LatestFailure,
		Guidance:      guidance,
	}
}

func factsOf(profile *agentprofile.Profile) []*agentprofile.Fact {
	if profile == nil {
		return []*agentprofile.Fact{}
	}

	facts := make([]*agentprofile.Fact, 0, len(profile.Facts))
	for idx := range profile.Facts {
		facts = append(facts, &profile.Facts[idx])
	}

	return facts
}

func guidanceAuthor(guidance *agentprofile.Guidance) *string {
	if guidance == nil || guidance.CreatedBy == nil {
		return nil
	}
	name := strings.TrimSpace(guidance.CreatedBy.Name)
	if name == "" {
		return nil
	}

	return &name
}
