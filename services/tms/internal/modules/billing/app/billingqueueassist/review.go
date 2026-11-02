package billingqueueassist

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/deliveryread"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	reviewMaxTokens      = 1500
	reviewMaxCalls       = 3
	reviewSummaryRunes   = 600
	reviewMaxDocuments   = 8
	reviewMaxCatalog     = 80
	reviewRecentInvoices = 12
	reviewMaxComments    = 10
	shipmentResource     = "shipment"
)

func (s *Service) ReviewItem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
) (*billingqueueports.AgentReviewOutcome, error) {
	available, err := s.Availability(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !available.Review {
		return &billingqueueports.AgentReviewOutcome{}, nil
	}

	evidence, err := s.gatherEvidence(ctx, tenantInfo, itemID)
	if err != nil || evidence == nil {
		return &billingqueueports.AgentReviewOutcome{}, err
	}
	item := evidence.Item
	if item.AgentReviewHash == reviewprompt.Hash(evidence) {
		outcome := &billingqueueports.AgentReviewOutcome{}
		if item.AgentVerdict != nil {
			outcome.Verdict = *item.AgentVerdict
		}
		return outcome, nil
	}

	answer := &reviewprompt.Answer{}
	reads := &readLog{}
	task := s.reviewTask(tenantInfo, evidence)
	task.Reads = s.reviewReads(tenantInfo, evidence, reads)
	_, err = s.nova.Run(ctx, task, answer)
	if err != nil && !errors.Is(err, llmports.ErrOutsideSchema) {
		return nil, fmt.Errorf("review billing queue item: %w", err)
	}

	return s.settleReview(ctx, tenantInfo, evidence, reads.taken(), answer, err)
}

// settleReview records Nova's answer to the evidence, with what it read
// (what it was shown, and taken: what it read further). An answer that is not
// its shape, or that the queue cannot show, is recorded as this evidence
// reviewed with no verdict, so the same evidence is not paid for again.
func (s *Service) settleReview(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	evidence *reviewprompt.Evidence,
	taken reviewprompt.Taken,
	answer *reviewprompt.Answer,
	answerErr error,
) (*billingqueueports.AgentReviewOutcome, error) {
	item := evidence.Item
	err := answerErr
	if err == nil {
		err = reviewprompt.Validate(answer, item)
	}
	if err != nil {
		s.l.Warn("discarded an unusable billing review",
			zap.String("billingQueueItemId", item.ID.String()), zap.Error(err))

		return &billingqueueports.AgentReviewOutcome{}, s.repo.RecordAgentReview(
			ctx,
			&billingqueueports.RecordAgentReviewRequest{
				TenantInfo: tenantInfo,
				ItemID:     item.ID,
				InputHash:  reviewprompt.Hash(evidence),
				ReviewedAt: timeutils.NowUnix(),
			},
		)
	}

	s.applyDocumentAnswers(ctx, evidence, answer.Documents)
	verdict := billingqueue.AgentVerdict(answer.Verdict)
	if err = s.repo.RecordAgentReview(ctx, &billingqueueports.RecordAgentReviewRequest{
		TenantInfo: tenantInfo,
		ItemID:     item.ID,
		Verdict:    verdict,
		Summary:    stringutils.Ellipsize(strings.TrimSpace(answer.Summary), reviewSummaryRunes),
		Reads:      reviewprompt.Reads(evidence, taken),
		InputHash:  reviewprompt.Hash(evidence),
		ReviewedAt: timeutils.NowUnix(),
	}); err != nil {
		return nil, err
	}

	return &billingqueueports.AgentReviewOutcome{Reviewed: true, Verdict: verdict}, nil
}

func (s *Service) gatherEvidence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
) (*reviewprompt.Evidence, error) {
	item, err := s.queue.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                itemID,
		TenantInfo:            tenantInfo,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, err
	}
	if !billingqueuereview.IsReviewable(item.Status) ||
		item.BillType != billingqueue.BillTypeInvoice ||
		item.Review == nil ||
		item.ShipmentID.IsNil() {
		return nil, nil
	}

	evidence := &reviewprompt.Evidence{Item: item, Excerpts: map[pulid.ID]reviewprompt.Excerpt{}}
	if evidence.Documents, err = s.reviewDocuments(ctx, tenantInfo, item.ShipmentID); err != nil {
		return nil, err
	}
	for _, doc := range evidence.Documents {
		content, contentErr := s.content.GetContent(ctx, doc.ID, tenantInfo)
		if contentErr != nil || content == nil {
			continue
		}
		if parts := reviewprompt.Parts(strings.TrimSpace(content.ContentText)); len(parts) > 0 {
			evidence.Excerpts[doc.ID] = reviewprompt.Excerpt{Text: parts[0], Parts: len(parts)}
		}
	}
	catalog, catalogErr := s.accessorials.ListCatalog(ctx, tenantInfo)
	if catalogErr != nil {
		return nil, catalogErr
	}
	if len(catalog) > reviewMaxCatalog {
		catalog = catalog[:reviewMaxCatalog]
	}
	evidence.Catalog = catalog
	if evidence.Comments, err = s.reviewComments(ctx, tenantInfo, item.ShipmentID); err != nil {
		return nil, err
	}
	evidence.Recent, err = s.repo.RecentCustomerInvoices(
		ctx,
		tenantInfo,
		item.BillToCustomerID,
		reviewRecentInvoices,
	)
	if err != nil {
		return nil, err
	}

	return evidence, nil
}

func (s *Service) reviewDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*document.Document, error) {
	docs, err := s.documents.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo:          tenantInfo,
		ResourceID:          shipmentID.String(),
		ResourceType:        shipmentResource,
		IncludeDocumentType: true,
	})
	if err != nil {
		return nil, err
	}

	current := make([]*document.Document, 0, min(len(docs), reviewMaxDocuments))
	for _, doc := range docs {
		if doc == nil || !doc.IsCurrentVersion || doc.Status == document.StatusRejected ||
			doc.Status == document.StatusArchived {
			continue
		}
		current = append(current, doc)
		if len(current) == reviewMaxDocuments {
			break
		}
	}

	return current, nil
}

func (s *Service) reviewComments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentComment, error) {
	page, err := s.comments.ListByShipmentID(ctx, &shipmentcommentports.ListShipmentCommentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: reviewMaxComments},
		},
		Cursor:     pagination.CursorInfo{Limit: reviewMaxComments},
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("read the shipment's comments: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	return page.Items, nil
}

func (s *Service) applyDocumentAnswers(
	ctx context.Context,
	evidence *reviewprompt.Evidence,
	answers []deliveryread.Answer,
) {
	if len(answers) == 0 {
		return
	}

	byID := make(map[string]*document.Document, len(evidence.Documents))
	for _, doc := range evidence.Documents {
		byID[doc.ID.String()] = doc
	}
	charges := make(map[string]deliveryread.Charge, len(evidence.Catalog))
	for _, charge := range evidence.Catalog {
		label := charge.Description
		if label == "" {
			label = charge.Code
		}
		charges[charge.ID.String()] = deliveryread.Charge{ID: charge.ID, Label: label}
	}

	for _, answer := range answers {
		doc := byID[answer.DocumentID]
		if doc == nil || deliveryread.Kept(doc.DeliveryFacts) {
			continue
		}
		facts := deliveryread.Facts(&answer, charges, timeutils.NowUnix())
		if facts == nil || deliveryread.Same(doc.DeliveryFacts, facts) {
			continue
		}
		written, err := s.documents.RecordDeliveryFacts(
			ctx,
			&documentports.RecordDeliveryFactsRequest{
				ID:         doc.ID,
				TenantInfo: tenantOf(doc),
				Facts:      facts,
				KeepPerson: true,
			},
		)
		if err != nil {
			s.l.Warn("could not record what Nova read from a document",
				zap.String("documentId", doc.ID.String()), zap.Error(err))
			continue
		}
		if written == nil {
			continue
		}
		*doc = *written
	}
}

func (s *Service) reviewTask(
	tenantInfo pagination.TenantInfo,
	e *reviewprompt.Evidence,
) *llmports.NovaTask {
	return &llmports.NovaTask{
		Name:         aitask.TaskBillingDiagnosis,
		Feature:      aiusage.FeatureBillingQueueReview,
		Instructions: reviewprompt.Instructions,
		SchemaName:   "billing_review",
		Schema:       reviewprompt.Schema(),
		MaxTokens:    reviewMaxTokens,
		Context:      llmports.DelimitedContext{Sections: reviewprompt.Sections(e)},
		TenantInfo:   tenantInfo,
		MaxCalls:     reviewMaxCalls,
	}
}

func tenantOf(doc *document.Document) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID}
}
