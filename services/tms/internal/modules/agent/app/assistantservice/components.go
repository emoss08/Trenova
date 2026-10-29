package assistantservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/agentaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistantproviders"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/chaptertitles"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/contextusage"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/decisionnotes"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/memorynotes"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposaloutcomes"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnattachments"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnlimits"
)

func (s *Service) attribution() *attribution.Attributor {
	return &attribution.Attributor{
		Logger:      s.logger,
		Definitions: s.definitions,
		Runs:        s.runs,
		Users:       s.users,
	}
}

func (s *Service) access() *agentaccess.Access {
	return &agentaccess.Access{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Queries:     s.queries,
	}
}

func (s *Service) limits() *turnlimits.Limits {
	return &turnlimits.Limits{
		Budgets:       s.budgets,
		Conversations: s.conversations,
		AgentControls: s.agentControls,
		Quota:         s.quota,
	}
}

func (s *Service) memoryNotes() *memorynotes.Describer {
	return &memorynotes.Describer{Logger: s.logger, Memories: s.memories}
}

func (s *Service) outcomes() *proposaloutcomes.Reader {
	return &proposaloutcomes.Reader{
		Logger:    s.logger,
		Proposals: s.proposals,
		Decisions: s.decisions,
	}
}

func (s *Service) chapterList() *chaptertitles.Index {
	return &chaptertitles.Index{Logger: s.logger, Chapters: s.chapters}
}

func (s *Service) notes() *decisionnotes.Notes {
	return &decisionnotes.Notes{
		Anchors:   s.anchors,
		Proposals: s.proposals,
		Plans:     s.plans,
		Decisions: s.decisions,
	}
}

func (s *Service) usage() *contextusage.Usage {
	return &contextusage.Usage{Logger: s.logger, Conversations: s.conversations}
}

func (s *Service) providerChoice() *assistantproviders.Options {
	return &assistantproviders.Options{Providers: s.providers}
}

func (s *Service) attachmentReads() *turnattachments.Reader {
	return &turnattachments.Reader{Logger: s.logger, Documents: s.documents, Contents: s.contents}
}
