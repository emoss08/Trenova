package sessionports

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
)

type SessionStanding struct {
	ID           pulid.ID
	Ended        bool
	SecretsKnown bool
	SecretHashes []string
}

type SessionLiveness interface {
	Standings(ctx context.Context, sessionIDs []pulid.ID) ([]SessionStanding, error)
}
