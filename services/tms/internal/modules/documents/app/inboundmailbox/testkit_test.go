package inboundmailbox

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/mailtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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

type tokenRenderer struct {
	documenttemplateports.DocumentTemplateResolver
}

func (tokenRenderer) RenderMessage(
	_ context.Context,
	req *documenttemplateports.RenderMessageRequest,
) (*documenttemplateports.RenderedMessage, error) {
	data, _ := req.Data.(mailtemplate.MailboxTestEmailContext)
	return &documenttemplateports.RenderedMessage{
		Subject: "Test message " + data.Token,
		Text:    data.MailboxAddress,
		HTML:    "<html>" + data.Token + "</html>",
	}, nil
}

func (t *testService) mailbox() *Service {
	audit := t.audit
	if audit == nil {
		audit = &mocks.NoopAuditService{}
	}

	return &Service{
		audit:        audit,
		email:        t.email,
		renderer:     tokenRenderer{},
		encryption:   t.encryption,
		l:            t.l,
		mailboxRepo:  t.mailboxRepo,
		mailboxState: t.mailboxState,
		now:          t.now,
	}
}
