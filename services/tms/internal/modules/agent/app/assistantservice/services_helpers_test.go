package assistantservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistantthreads"
)

func threadsOf(s *Service) *assistantthreads.Service {
	return assistantthreads.New(assistantthreads.Params{
		Logger:        s.logger,
		Runtime:       s.runtime,
		Conversations: s.conversations,
		Definitions:   s.definitions,
		Permissions:   s.permissions,
		Queries:       s.queries,
	})
}
