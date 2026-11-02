package agentguard

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"go.uber.org/zap"
)

// SetCompletionForTest wires a stub classifier onto a Service built as a
// literal. The field is unexported because nothing outside this package has
// any business swapping the classifier at runtime; a test that needs a slow
// or failing one does.
func SetCompletionForTest(s *Service, completion llmports.CompletionService) {
	s.completion = completion
	if s.logger == nil {
		s.logger = zap.NewNop()
	}
	if s.verdicts == nil {
		s.verdicts = newVerdictCache(nil, 0, s.logger)
	}
}
