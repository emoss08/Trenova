package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type PasswordResetTokenRepository interface {
	// Create stores a freshly minted token.
	Create(ctx context.Context, token *tenant.PasswordResetToken) error

	// FindRedeemableByHash returns the token behind a link, or a not-found error when
	// the digest is unknown, already used, superseded, or past its expiry. The four
	// cases are deliberately indistinguishable to the caller: telling them apart in a
	// response would let somebody probe which links once existed.
	FindRedeemableByHash(
		ctx context.Context,
		tokenHash string,
		now int64,
	) (*tenant.PasswordResetToken, error)

	// MarkUsed redeems a token, and reports whether it was still redeemable at the
	// moment of the write. Two requests racing on the same link must not both win, so
	// the used_at guard lives in the UPDATE rather than in a prior read.
	MarkUsed(ctx context.Context, tokenID pulid.ID, now int64) (bool, error)

	// InvalidateOutstanding retires every unused token for a user. Called when a new
	// link is issued, so only the newest email works, and again after a successful
	// reset so nothing else is left redeemable.
	InvalidateOutstanding(ctx context.Context, userID pulid.ID, now int64) error

	// CountSince counts the tokens issued to a user since a cutoff, so a request can
	// be refused before it becomes a way to flood somebody's inbox.
	CountSince(ctx context.Context, userID pulid.ID, since int64) (int, error)
}
