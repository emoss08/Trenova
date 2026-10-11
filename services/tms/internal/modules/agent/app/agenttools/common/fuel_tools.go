package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const ParamTractorID = "tractorId"

const ParamFuelType = "fuelType"

const KeepWhenLeftOut = " Leave it out to keep it."

var FuelTypes = agenttoolschema.Source("ifta.fuelType", domaintypes.IFTAFuelTypeValues())

func FuelInternalSpec(spec *ReceivableSpec) *ReceivableSpec {
	spec.Egress = agent.EgressInternal
	if spec.DefaultTier == "" {
		spec.DefaultTier = agent.TierPropose
	}
	if spec.MaxTier == "" {
		spec.MaxTier = agent.TierActWithApproval
	}

	return spec
}
