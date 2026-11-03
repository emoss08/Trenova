package masterdatakit

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const ParamCode = "code"

const ParamName = "name"

const ParamAddressLine1 = "addressLine1"

const ParamAddressLine2 = "addressLine2"

const ParamCity = "city"

const ParamState = "state"

const ParamPostalCode = "postalCode"

const ParamPhone = "phone"

const ParamEmail = "email"

const ParamDOTNumber = "dotNumber"

const ParamMCNumber = "mcNumber"

const ParamStatus = "status"

const ParamMake = "make"

const ParamModel = "model"

const ParamYear = "year"

const ParamVIN = "vin"

const ParamLicensePlateNumber = "licensePlateNumber"

const ParamRegistrationNumber = "registrationNumber"

const ParamRegistrationExpiry = "registrationExpiry"

const ParamEquipmentTypeID = "equipmentTypeId"

const ParamEquipmentManufacturerID = "equipmentManufacturerId"

const ParamFleetCodeID = "fleetCodeId"

const ParamExternalID = "externalId"

const ActionUpdated = "updated"

const MdMaxName = 255

const MdMaxShortName = 100

const MdMaxCode = 10

const MdMaxAddress = 150

const MdMaxCity = 100

const MdMaxPostalCode = 10

const MdMaxPhone = 20

const MdMaxNumber = 12

const MdMaxNotes = 5000

const MdMaxUnitText = 50

const MdMaxExternalID = 100

const MdMinYear = 1950

const MdMaxYear = 2100

const MdMaxTemperature = 200

const MdMinTemperature = -100

const MdMaxPaymentTerm = 365

const MdMaxConsolidPrio = 10

const MdTaintHold = "A record read from a document or message someone outside sent is proposed, " +
	"since colleagues book and pay against it."

var ActiveStatuses = agenttoolschema.Source("masterData.status", domaintypes.StatusValues())

var MdInternalAsk = Policy{
	DefaultTier: agent.TierPropose,
	MaxTier:     agent.TierActWithApproval,
	TaintHold:   MdTaintHold,
}

var MdInternalStatus = Policy{
	DefaultTier: agent.TierPropose,
	MaxTier:     agent.TierActWithApproval,
}

const ActiveStatusNote = "Active records are offered when people book; Inactive ones are " +
	"kept on file and no longer offered."
