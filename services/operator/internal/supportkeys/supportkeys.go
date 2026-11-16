package supportkeys

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

type TMS interface {
	SupportKeys(ctx context.Context) (tms.HeldSupportKeys, error)
	PutSupportKeys(ctx context.Context, keys tms.SupportKeys) error
}

type Relay struct {
	tms    TMS
	logger *slog.Logger
}

func New(t TMS, logger *slog.Logger) *Relay {
	return &Relay{tms: t, logger: logger}
}

func (r *Relay) Sync(ctx context.Context, desired *instanceapi.SpecSupport) error {
	if desired == nil {
		return nil
	}
	want := keyIDs(desired.Keys)
	held, err := r.tms.SupportKeys(ctx)
	if err != nil {
		return err
	}
	if held.InstanceID == desired.InstanceId &&
		held.StaffDomain == desired.StaffDomain &&
		sameSet(held.KeyIDs, want) {
		return nil
	}
	keys := make([]tms.SupportKey, 0, len(desired.Keys))
	for _, k := range desired.Keys {
		keys = append(keys, tms.SupportKey{
			Kid: k.Kid,
			Kty: k.Kty,
			Crv: k.Crv,
			Alg: k.Alg,
			Use: k.Use,
			X:   k.X,
		})
	}
	if err = r.tms.PutSupportKeys(ctx, tms.SupportKeys{
		InstanceID:  desired.InstanceId,
		StaffDomain: desired.StaffDomain,
		Keys:        keys,
	}); err != nil {
		return fmt.Errorf("key ids %s: %w", strings.Join(want, ","), err)
	}
	r.logger.Info(
		"support access keys sent to the TMS",
		"staffDomain", desired.StaffDomain,
		"keyIds", want,
		"replacedKeyIds", held.KeyIDs,
	)
	return nil
}

func keyIDs(keys []instanceapi.SupportKey) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k.Kid)
	}
	return out
}

func sameSet(held, want []string) bool {
	a, b := slices.Clone(held), slices.Clone(want)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(slices.Compact(a), slices.Compact(b))
}
