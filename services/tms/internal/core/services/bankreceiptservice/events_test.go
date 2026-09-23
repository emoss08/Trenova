package bankreceiptservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingEvents struct {
	published []serviceports.AgentEvent
}

func (r *recordingEvents) Publish(_ context.Context, event serviceports.AgentEvent) {
	r.published = append(r.published, event)
}

// An import that ends in an exception tells the agents, with the receipt as
// the subject, so a cash application agent can start on it. The repository
// reports "no open work item" as a not-found, and that is read as none: the
// item is opened rather than skipped.
func TestImportAnnouncesAnExceptionAndOpensItsWorkItem(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	receiptRepo := mocks.NewMockBankReceiptRepository(t)
	paymentRepo := mocks.NewMockCustomerPaymentRepository(t)
	workItemRepo := mocks.NewMockBankReceiptWorkItemRepository(t)
	accountingRepo := mocks.NewMockAccountingControlRepository(t)

	var stored *bankreceipt.BankReceipt
	receiptRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			created := *entity
			created.ID = pulid.MustNew("brcpt_")
			stored = &created

			return &created, nil
		}).
		Once()
	receiptRepo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *bankreceipt.BankReceipt) (*bankreceipt.BankReceipt, error) {
			updated := *entity
			stored = &updated

			return &updated, nil
		}).
		Once()
	paymentRepo.EXPECT().
		FindSuggestedMatchCandidates(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()
	workItemRepo.EXPECT().
		GetActiveByReceiptID(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("bank receipt work item not found")).
		Once()
	var opened *bankreceiptworkitem.WorkItem
	workItemRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, item *bankreceiptworkitem.WorkItem) (*bankreceiptworkitem.WorkItem, error) {
			opened = item

			return item, nil
		}).
		Once()
	accountingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.AccountingControl{ReconciliationMode: tenant.ReconciliationModeWarnOnly}, nil).
		Once()

	events := &recordingEvents{}
	svc := &Service{
		l:              zap.NewNop(),
		repo:           receiptRepo,
		paymentRepo:    paymentRepo,
		workItemRepo:   workItemRepo,
		accountingRepo: accountingRepo,
		auditService:   &mocks.NoopAuditService{},
		events:         events,
	}

	receipt, err := svc.Import(
		t.Context(),
		&serviceports.ImportBankReceiptRequest{
			ReceiptDate:     100,
			AmountMinor:     125_000,
			ReferenceNumber: "ACH 4471",
			TenantInfo:      pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)
	require.NoError(t, err)
	assert.Equal(t, bankreceipt.StatusException, receipt.Status)

	require.NotNil(t, opened, "the exception opens a queue entry")
	assert.Equal(t, stored.ID, opened.BankReceiptID)
	assert.Equal(t, bankreceiptworkitem.StatusOpen, opened.Status)

	require.Len(t, events.published, 1)
	assert.Equal(t, agent.EventBankReceiptException, events.published[0].Kind)
	assert.Equal(t, stored.ID, events.published[0].SubjectID)
	assert.Equal(t, pagination.TenantInfo{OrgID: orgID, BuID: buID}, events.published[0].TenantInfo)
}
