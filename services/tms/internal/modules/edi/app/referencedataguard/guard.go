package referencedataguard

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/platformconfig"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	PlatformConfig *platformconfig.Config
	Edition        ediports.EditionInfo
}

type Guard struct {
	requireListed bool
	stewards      map[pulid.ID]struct{}
}

func New(p Params) (*Guard, error) {
	return FromPlatform(p.PlatformConfig, p.Edition)
}

func FromPlatform(platform *platformconfig.Config, edition ediports.EditionInfo) (*Guard, error) {
	stewards := make(map[pulid.ID]struct{}, len(platform.ReferenceDataStewards))
	for _, raw := range platform.ReferenceDataStewards {
		id, err := pulid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf(
				"platform.referenceDataStewards has an invalid organization ID %q: %w",
				raw,
				err,
			)
		}
		stewards[id] = struct{}{}
	}

	return &Guard{
		requireListed: (edition != nil && edition.SharedTenancy()) || len(stewards) > 0,
		stewards:      stewards,
	}, nil
}

func (g *Guard) RequireSteward(organizationID pulid.ID) error {
	if g == nil || organizationID.IsNil() {
		return errStewardRequired()
	}

	if !g.requireListed {
		return nil
	}

	if _, ok := g.stewards[organizationID]; ok {
		return nil
	}

	return errStewardRequired()
}

func errStewardRequired() error {
	return errortypes.NewAuthorizationError(
		"Shared reference data can only be changed by the platform operator",
	)
}
