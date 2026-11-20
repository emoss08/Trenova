package instancebootstrapports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/shared/pulid"
)

var ErrNotBootstrapped = errors.New("the instance has not been bootstrapped yet")

type FirstAdministratorInvite struct {
	TokenSHA256 string
	ExpiresAt   int64
}

type FirstAdministratorInviteResult struct {
	SetUp             bool
	InviteTokenSHA256 string
}

type FirstAdministratorInviteService interface {
	Install(
		ctx context.Context,
		invite *FirstAdministratorInvite,
	) (*FirstAdministratorInviteResult, error)
	InstallFor(
		ctx context.Context,
		adminUserID pulid.ID,
		invite *FirstAdministratorInvite,
	) (*FirstAdministratorInviteResult, error)
}
