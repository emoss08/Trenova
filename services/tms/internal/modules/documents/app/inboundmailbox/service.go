package inboundmailbox

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	// Audit records mailbox configuration changes.
	Audit        auditports.AuditService `optional:"true"`
	Email        emailports.EmailService `optional:"true"`
	Encryption   SecretKeeper
	Logger       *zap.Logger
	MailboxRepo  inboundmessageports.InboundMailboxRepository
	MailboxState inboundmessageports.TriageMailboxStateRepository
	Now          func() int64 `optional:"true"`
}

type Service struct {
	audit        auditports.AuditService
	email        emailports.EmailService
	encryption   SecretKeeper
	l            *zap.Logger
	mailboxRepo  inboundmessageports.InboundMailboxRepository
	mailboxState inboundmessageports.TriageMailboxStateRepository
	now          func() int64
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	now := p.Now
	if now == nil {
		now = timeutils.NowUnix
	}
	return &Service{
		audit:        p.Audit,
		email:        p.Email,
		encryption:   p.Encryption,
		l:            p.Logger.Named("service.inbound-message"),
		mailboxRepo:  p.MailboxRepo,
		mailboxState: p.MailboxState,
		now:          now,
	}
}

// secretKeeper is the sliver of the encryption service this needs: sealing a
// mailbox's signing secret when it is set, and opening it to check a delivery.
// Taking the interface rather than the concrete service keeps both paths
// testable without standing up a key manager.
type SecretKeeper interface {
	EncryptStringWithAAD(value string, aad encryptionservice.AAD) (string, error)
	DecryptStringWithLegacy(
		value string,
		aad encryptionservice.AAD,
	) (plaintext string, legacy bool, err error)
}

func AsSecretKeeper(s *encryptionservice.Service) SecretKeeper {
	return s
}
