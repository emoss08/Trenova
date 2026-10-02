package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mentionPermissions struct {
	serviceports.PermissionEngine

	readable map[string]bool
}

func (p *mentionPermissions) CheckBatch(
	_ context.Context,
	req *serviceports.BatchPermissionCheckRequest,
) (*serviceports.BatchPermissionCheckResult, error) {
	results := make([]serviceports.PermissionCheckResult, 0, len(req.Checks))
	for _, check := range req.Checks {
		results = append(results, serviceports.PermissionCheckResult{
			Allowed: p.readable[check.Resource] && check.Operation == permission.OpRead,
		})
	}

	return &serviceports.BatchPermissionCheckResult{Results: results}, nil
}

type mentionConversations struct {
	repositories.ConversationRepository

	captured repositories.SearchMentionsRequest
	rows     []repositories.MentionRow
}

func (c *mentionConversations) SearchMentions(
	_ context.Context,
	req repositories.SearchMentionsRequest,
) ([]repositories.MentionRow, error) {
	c.captured = req

	return c.rows, nil
}

func TestSearchMentions_OnlySearchesWhatThePersonMayRead(t *testing.T) {
	t.Parallel()

	repo := &mentionConversations{rows: []repositories.MentionRow{
		{Type: "shipment", ID: "shp_1", Label: "SEED-SHP-001", Subtitle: "New"},
	}}
	service := &Service{
		conversations: repo,
		permissions: &mentionPermissions{readable: map[string]bool{
			permission.ResourceShipment.String(): true,
			permission.ResourceCarrier.String():  true,
		}},
	}

	results, err := service.SearchMentions(
		t.Context(),
		providerActor(),
		serviceports.MentionSearchRequest{Query: " seed "},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"shipment", "carrier"}, repo.captured.Kinds)
	assert.Equal(t, "seed", repo.captured.Query)
	assert.Equal(t, mentionLimitAllKinds, repo.captured.LimitPerKind)
	require.Len(t, results, 1)
	assert.Equal(t, "SEED-SHP-001", results[0].Label)
}

func TestSearchMentions_InvoiceTabCoversTheBillingQueue(t *testing.T) {
	t.Parallel()

	repo := &mentionConversations{}
	service := &Service{
		conversations: repo,
		permissions: &mentionPermissions{readable: map[string]bool{
			permission.ResourceInvoice.String():      true,
			permission.ResourceBillingQueue.String(): true,
		}},
	}

	_, err := service.SearchMentions(
		t.Context(),
		providerActor(),
		serviceports.MentionSearchRequest{Kind: "invoice"},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"invoice", "billing_queue_item"}, repo.captured.Kinds)
	assert.Equal(t, mentionLimitOneKind, repo.captured.LimitPerKind)
}

func TestSearchMentions_NothingReadableSearchesNothing(t *testing.T) {
	t.Parallel()

	repo := &mentionConversations{}
	service := &Service{
		conversations: repo,
		permissions:   &mentionPermissions{readable: map[string]bool{}},
	}

	results, err := service.SearchMentions(
		t.Context(),
		providerActor(),
		serviceports.MentionSearchRequest{Query: "acme"},
	)
	require.NoError(t, err)

	assert.Empty(t, results)
	assert.Nil(t, repo.captured.Kinds)
}
