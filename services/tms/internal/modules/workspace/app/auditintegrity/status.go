package auditintegrity

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ChainStatus struct {
	Signed                 bool
	LastSeq                int64
	SealedThrough          int64
	LastVerifiedAt         *int64
	LastVerificationStatus audit.VerificationStatus
	LastVerificationDetail string
}

type Status struct {
	seals   auditports.SealRepository
	keyring *Keyring
}

func NewStatus(seals auditports.SealRepository, keyring *Keyring) *Status {
	return &Status{seals: seals, keyring: keyring}
}

func (s *Status) Get(ctx context.Context, tenant pagination.TenantInfo) (*ChainStatus, error) {
	head, err := s.seals.GetHead(ctx, tenant)
	if err != nil {
		return nil, err
	}
	return &ChainStatus{
		Signed:                 s.keyring.Signed(),
		LastSeq:                head.LastSeq,
		SealedThrough:          head.SealedThrough,
		LastVerifiedAt:         head.LastVerifiedAt,
		LastVerificationStatus: head.LastVerificationStatus,
		LastVerificationDetail: head.LastVerificationDetail,
	}, nil
}
