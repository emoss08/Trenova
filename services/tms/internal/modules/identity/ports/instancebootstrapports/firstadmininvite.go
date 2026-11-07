package instancebootstrapports

import (
	"context"
	"errors"
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
}
