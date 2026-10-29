package inboundmailbox

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/zap"
)

type testService struct {
	l           *zap.Logger
	mailboxRepo inboundmessageports.InboundMailboxRepository
	encryption  SecretKeeper
	audit       auditports.AuditService

	mailboxState inboundmessageports.TriageMailboxStateRepository
	email        emailports.EmailService
	now          func() int64
}

func (t *testService) mailbox() *Service {
	return &Service{
		audit:        t.audit,
		email:        t.email,
		encryption:   t.encryption,
		l:            t.l,
		mailboxRepo:  t.mailboxRepo,
		mailboxState: t.mailboxState,
		now:          t.now,
	}
}
