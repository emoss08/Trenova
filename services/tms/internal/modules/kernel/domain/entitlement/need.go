package entitlement

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

var (
	ErrNoNeed = errors.New(
		"work across tenants must name the feature it belongs to (entitlement.For) or that it serves every active tenant (entitlement.AnyActive)",
	)
	ErrUnknownFeature = errors.New("work across tenants names a feature the catalog does not know")
	ErrNoWork         = errors.New(
		"a sweep across organizations on a shared server runs only for the feature of the schedule that started it, and the context carries none",
	)
)

type Need struct {
	feature   platformcatalog.FeatureKey
	anyActive bool
}

func For(feature platformcatalog.FeatureKey) Need {
	return Need{feature: feature}
}

func AnyActive() Need {
	return Need{anyActive: true}
}

func (n Need) Validate() error {
	if n.anyActive {
		return nil
	}
	if n.feature == "" {
		return ErrNoNeed
	}
	if _, known := platformcatalog.Lookup(n.feature); !known {
		return fmt.Errorf("%w: %q", ErrUnknownFeature, n.feature)
	}
	return nil
}

func (n Need) Feature() (platformcatalog.FeatureKey, bool) {
	return n.feature, !n.anyActive && n.feature != ""
}

func (n Need) AddOn() bool {
	feature, ok := n.Feature()
	if !ok {
		return false
	}
	known, found := platformcatalog.Lookup(feature)
	return found && !known.IsBase()
}

func (n Need) String() string {
	if n.anyActive {
		return "any active plan"
	}
	return string(n.feature)
}

type workKey struct{}

func WithWork(ctx context.Context, need Need) context.Context {
	return context.WithValue(ctx, workKey{}, need)
}

func WorkFrom(ctx context.Context) (Need, bool) {
	need, ok := ctx.Value(workKey{}).(Need)
	return need, ok && need.Validate() == nil
}

const anyActiveText = "*"

func (n Need) MarshalJSON() ([]byte, error) {
	text := string(n.feature)
	if n.anyActive {
		text = anyActiveText
	}
	return []byte(strconv.Quote(text)), nil
}

func (n *Need) UnmarshalJSON(data []byte) error {
	text, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("decode a work need: %w", err)
	}
	if text == anyActiveText {
		*n = AnyActive()
		return nil
	}
	*n = For(platformcatalog.FeatureKey(text))
	return nil
}
