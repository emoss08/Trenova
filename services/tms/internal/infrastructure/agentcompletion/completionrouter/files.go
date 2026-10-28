package completionrouter

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// fileSource reads a request's attached files within its tenant. Nil when no
// reader is configured, and every file is then read as its text.
func (s *Service) fileSource(tenant pagination.TenantInfo) modeladapter.FileSource {
	if s.files == nil {
		return nil
	}

	return func(ctx context.Context, documentID string) (*assistantports.MessageFileContent, error) {
		id, err := pulid.Parse(documentID)
		if err != nil {
			return nil, err
		}

		return s.files.ReadMessageFile(ctx, &assistantports.ReadMessageFileRequest{
			TenantInfo: tenant,
			DocumentID: id,
		})
	}
}

// noteFiles logs each attached file the model was sent as its text rather than
// as itself, and why: the answer still came, from the text, but an operator
// may want to know a file was too large or would not open.
func (s *Service) noteFiles(provider *aiprovider.Provider, report modeladapter.FileReport) {
	for _, fallback := range report.Fallbacks {
		s.logger.Info("an attached file was sent to the model as its text",
			zap.String("provider", provider.ID.String()),
			zap.String("document", fallback.DocumentID),
			zap.String("reason", fallback.Reason),
		)
	}
}
