package paperworkcontract_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/paperworkjobs/paperworkcontract"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type startedWorkflows struct {
	kernelports.WorkflowStarter

	ids []string
}

func (s *startedWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	s.ids = append(s.ids, options.ID)
	return nil, nil
}

func paperwork() *document.Document {
	return &document.Document{
		ID:                pulid.MustNew("doc_"),
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		ProcessingProfile: document.ProcessingProfileShipmentPaperwork,
	}
}

func TestPaperworkIsNotReadWithoutDocumentIntelligence(t *testing.T) {
	t.Parallel()

	workflows := &startedWorkflows{}
	requester := paperworkcontract.NewRequester(paperworkcontract.RequesterParams{
		Workflows: workflows,
		Features:  plantest.Without(platformcatalog.FeatureDocumentIntelligence),
		Logger:    zap.NewNop(),
	})

	requester.RequestRead(t.Context(), paperwork(), pulid.MustNew("usr_"))

	assert.Empty(t, workflows.ids, "no read is queued for an add-on the plan leaves out")
}

func TestPaperworkIsReadWithDocumentIntelligence(t *testing.T) {
	t.Parallel()

	workflows := &startedWorkflows{}
	requester := paperworkcontract.NewRequester(paperworkcontract.RequesterParams{
		Workflows: workflows,
		Features:  plantest.AllFeatures(),
		Logger:    zap.NewNop(),
	})
	doc := paperwork()

	requester.RequestRead(t.Context(), doc, pulid.MustNew("usr_"))

	assert.Equal(t, []string{paperworkcontract.WorkflowID(doc.ID)}, workflows.ids)
}
