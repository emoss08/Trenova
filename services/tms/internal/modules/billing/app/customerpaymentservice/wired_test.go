package customerpaymentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/identitytest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerledgerportsmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func wired(t *testing.T, s *Service) *Service {
	t.Helper()

	ledger := customerledgerportsmock.NewMockCustomerLedgerProjectionRepository(t)
	ledger.EXPECT().AppendEntries(mock.Anything, mock.Anything).Return(nil).Maybe()

	s.l = zap.NewNop()
	s.customerLedgerRepo = ledger
	s.realtime = &mocks.NoopRealtimeService{}
	s.triage = &mocks.NoopTriageEventSink{}
	s.stamper = identitytest.NewFixedForTest(t, "USD", decimal.NewFromInt(1))
	s.receiptMatches = &recordingReceiptReleaser{}

	return s
}
