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

func TestSearchMentions_AllLeavesOutTheKindsOnlyAPickerAsksFor(t *testing.T) {
	t.Parallel()

	repo := &mentionConversations{}
	service := &Service{
		conversations: repo,
		permissions: &mentionPermissions{readable: map[string]bool{
			permission.ResourceInvoice.String():        true,
			permission.ResourceInvoiceDispute.String(): true,
		}},
	}

	_, err := service.SearchMentions(t.Context(), providerActor(), serviceports.MentionSearchRequest{})
	require.NoError(t, err)

	assert.Equal(t, []string{"invoice"}, repo.captured.Kinds)
}

func TestSearchMentionPage_PagesOneKindAndSaysWhetherMoreFollow(t *testing.T) {
	t.Parallel()

	rows := make([]repositories.MentionRow, 0, 4)
	for _, id := range []string{"idsp_1", "idsp_2", "idsp_3", "idsp_4"} {
		rows = append(rows, repositories.MentionRow{Type: "invoice_dispute", ID: id})
	}
	repo := &mentionConversations{rows: rows}
	service := &Service{
		conversations: repo,
		permissions: &mentionPermissions{readable: map[string]bool{
			permission.ResourceInvoiceDispute.String(): true,
		}},
	}

	page, err := service.SearchMentionPage(t.Context(), providerActor(), serviceports.MentionPageRequest{
		Query:  "INV",
		Kind:   "invoice_dispute",
		Offset: 30,
		Limit:  3,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"invoice_dispute"}, repo.captured.Kinds)
	assert.Equal(t, 4, repo.captured.LimitPerKind, "one row past the page says whether more follow")
	assert.Equal(t, 30, repo.captured.Offset)
	assert.True(t, page.HasMore)
	require.Len(t, page.Results, 3)
	assert.Equal(t, "idsp_3", page.Results[2].ID)

	repo.rows = rows[:2]
	page, err = service.SearchMentionPage(t.Context(), providerActor(), serviceports.MentionPageRequest{
		Kind:  "invoice_dispute",
		Limit: 3,
	})
	require.NoError(t, err)
	assert.False(t, page.HasMore)
	assert.Len(t, page.Results, 2)
}

func TestSearchMentionPage_CapsThePageAndRefusesAnUnknownKind(t *testing.T) {
	t.Parallel()

	repo := &mentionConversations{}
	service := &Service{
		conversations: repo,
		permissions: &mentionPermissions{readable: map[string]bool{
			permission.ResourceShipment.String(): true,
		}},
	}

	_, err := service.SearchMentionPage(t.Context(), providerActor(), serviceports.MentionPageRequest{
		Kind:   "shipment",
		Limit:  5000,
		Offset: -4,
	})
	require.NoError(t, err)
	assert.Equal(t, mentionPageMax+1, repo.captured.LimitPerKind)
	assert.Zero(t, repo.captured.Offset)

	_, err = service.SearchMentionPage(t.Context(), providerActor(), serviceports.MentionPageRequest{
		Kind: "all",
	})
	require.Error(t, err)
}
