package masterdatakit

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const MdCode = "code"

const MdName = "name"

const MdDescription = "description"

const MdAddressLine1 = "addressLine1"

const MdAddressLine2 = "addressLine2"

const MdCity = "city"

const MdState = "state"

const MdPostalCode = "postalCode"

const MdPhone = "phone"

const MdEmail = "email"

const MdNote = "note"

const MdDOTNumber = "dotNumber"

const MdMCNumber = "mcNumber"

const MdStatus = "status"

const MdMake = "make"

const MdModel = "model"

const MdYear = "year"

const MdVIN = "vin"

const MdLicensePlate = "licensePlateNumber"

const MdRegistrationNumber = "registrationNumber"

const MdRegistrationExpiry = "registrationExpiry"

const MdEquipmentTypeID = "equipmentTypeId"

const MdManufacturerID = "equipmentManufacturerId"

const MdFleetCodeID = "fleetCodeId"

const MdExternalID = "externalId"

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
