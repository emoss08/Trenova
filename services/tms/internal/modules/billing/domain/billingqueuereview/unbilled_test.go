package billingqueuereview

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindingsRaiseAReceiptTheInvoiceDoesNotBill(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	lumper := pulid.MustNew("acc_")
	issues := Findings(&FindingInput{
		CustomerName: "Acme Foods",
		UnbilledReceipts: []*UnbilledReceipt{{
			DocumentID:          documentID,
			Reference:           "LMP-5521",
			AccessorialChargeID: lumper,
			Label:               "Lumper",
			Amount:              dec("185"),
		}},
	})

	require.Len(t, issues, 1)
	issue := issues[0]
	assert.Equal(t, billingqueue.CheckCharges, issue.CheckKey)
	assert.Equal(t, billingqueue.IssueUnbilledAccessorial, issue.Code)
	assert.Equal(t, documentID.String(), issue.SubjectKey)
	assert.Equal(t, "Lumper $185.00 isn't billed", issue.Summary)
	assert.Equal(
		t,
		"Receipt LMP-5521 for $185.00 is on the shipment, but no lumper charge bills it.",
		issue.Reasoning,
	)
	require.Len(t, issue.Options, 2)
	assert.Equal(t, "add", issue.Options[0].Key)
	assert.Equal(t, "Bill it · $185.00", issue.Options[0].Label)
	assert.Equal(t, billingqueue.EffectAdd, issue.Options[0].Effect.Kind)
	assert.Equal(t, lumper, issue.Options[0].Effect.AccessorialChargeID)
	assert.True(t, issue.Options[0].Effect.Amount.Decimal.Equal(dec("185")))
	assert.Equal(t, "Lumper", issue.Options[0].Effect.Label, "the line it adds is named for the list")
	assert.Equal(t, "skip", issue.Options[1].Key)
	assert.Equal(t, billingqueue.EffectAccept, issue.Options[1].Effect.Kind)
	assert.Equal(t, billingqueue.IssueSourceDeterministic, issue.Source)
}

func TestFindingsCreditNovaWithAReceiptItReadAndAPersonWithOneTheyEntered(t *testing.T) {
	t.Parallel()

	read := &UnbilledReceipt{
		DocumentID:          pulid.MustNew("doc_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Label:               "Lumper",
		Amount:              dec("185"),
		ReadByNova:          true,
	}
	entered := &UnbilledReceipt{
		DocumentID:          pulid.MustNew("doc_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Label:               "Scale ticket",
		Amount:              dec("12"),
	}

	issues := Findings(&FindingInput{UnbilledReceipts: []*UnbilledReceipt{read, entered}})

	require.Len(t, issues, 2)
	assert.Equal(t, billingqueue.IssueSourceAgent, issues[0].Source)
	assert.Equal(t, billingqueue.IssueSourceDeterministic, issues[1].Source)
}

func TestFindingsAskForThePONumberTheCustomerRequires(t *testing.T) {
	t.Parallel()

	issues := Findings(&FindingInput{CustomerName: "Acme Foods", MissingPONumber: true})

	require.Len(t, issues, 1)
	assert.Equal(t, billingqueue.CheckTerms, issues[0].CheckKey)
	assert.Equal(t, billingqueue.IssuePONumberMissing, issues[0].Code)
	assert.Equal(t, "Acme Foods needs a PO number on the invoice", issues[0].Summary)
	assert.Empty(t, issues[0].Options, "only entering the PO settles it")
}

func TestAMissingPONumberBlocksApproval(t *testing.T) {
	t.Parallel()

	item := reviewItem(billingqueue.StatusInReview, true)
	issues := Findings(&FindingInput{CustomerName: "Acme Foods", MissingPONumber: true})
	review := BuildReadiness(item, issues, CheckFacts{})

	assert.False(t, review.Ready)
	assert.Equal(t, billingqueue.BlockerIssue, review.Blocker)
}

func TestAnUnbilledReceiptOnlySuggestsAndLeavesTheItemReadyToApprove(t *testing.T) {
	t.Parallel()

	item := reviewItem(billingqueue.StatusInReview, true)
	issues := Findings(&FindingInput{
		CustomerName: "Acme Foods",
		UnbilledReceipts: []*UnbilledReceipt{{
			DocumentID:          pulid.MustNew("doc_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Label:               "Lumper",
			Amount:              dec("185"),
			ReadByNova:          true,
		}},
	})
	for _, issue := range issues {
		issue.ID = pulid.MustNew("bqis_")
	}

	review := BuildReadiness(item, issues, CheckFacts{})

	assert.True(t, review.Ready)
	assert.Zero(t, review.NeedsCount)
	assert.Equal(t, billingqueue.BlockerNone, review.Blocker)
	for _, check := range review.Checks {
		assert.NotEqual(t, billingqueue.CheckCodeIssue, check.Code, "no check waits on a suggestion")
	}
	assert.Len(t, review.Issues, 1, "the suggestion is still on the item, where the charges show it")
}

func TestAnUnbilledReceiptBesideARealIssueCountsOnlyTheIssue(t *testing.T) {
	t.Parallel()

	item := reviewItem(billingqueue.StatusInReview, true)
	issues := Findings(&FindingInput{
		CustomerName:    "Acme Foods",
		MissingPONumber: true,
		UnbilledReceipts: []*UnbilledReceipt{{
			DocumentID:          pulid.MustNew("doc_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Label:               "Lumper",
			Amount:              dec("185"),
		}},
	})

	review := BuildReadiness(item, issues, CheckFacts{})

	assert.False(t, review.Ready)
	assert.Equal(t, 1, review.NeedsCount)
}
