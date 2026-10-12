package masterdatatools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const mdCode = "code"

const MdName = "name"

const mdDescription = "description"

const mdAddressLine1 = "addressLine1"

const mdAddressLine2 = "addressLine2"

const mdCity = "city"

const mdState = "state"

const mdPostalCode = "postalCode"

const mdPhone = "phone"

const mdEmail = "email"

const mdNotes = "notes"

const mdDOTNumber = "dotNumber"

const mdMCNumber = "mcNumber"

const mdStatus = "status"

const mdMake = "make"

const mdModel = "model"

const mdYear = "year"

const mdVIN = "vin"

const mdLicensePlate = "licensePlateNumber"

const mdRegistrationNumber = "registrationNumber"

const mdRegistrationExpiry = "registrationExpiry"

const mdEquipmentTypeID = "equipmentTypeId"

const mdManufacturerID = "equipmentManufacturerId"

const mdFleetCodeID = "fleetCodeId"

const mdExternalID = "externalId"

const ActionUpdated = "updated"

const AlertCustomMessage = "customMessage"

const AlertKind = "change alert"

const AlertTableName = "tableName"

const mdMaxName = 255

const mdMaxShortName = 100

const mdMaxCode = 10

const mdMaxAddress = 150

const mdMaxCity = 100

const mdMaxPostalCode = 10

const mdMaxPhone = 20

const mdMaxNumber = 12

const mdMaxNotes = 5000

const mdMaxUnitText = 50

const mdMaxExternalID = 100

const mdMinYear = 1950

const mdMaxYear = 2100

const mdMaxTemperature = 200

const mdMinTemperature = -100

const mdMaxPaymentTerm = 365

const mdMaxConsolidPrio = 10

const mdTaintHold = "A record read from a document or message someone outside sent is proposed, " +
	"since colleagues book and pay against it."

var activeStatuses = agenttoolschema.Source("masterData.status", domaintypes.StatusValues())

var mdInternalAsk = masterPolicy{
	defaultTier: agent.TierPropose,
	maxTier:     agent.TierActWithApproval,
	taintHold:   mdTaintHold,
}

var mdInternalStatus = masterPolicy{
	defaultTier: agent.TierPropose,
	maxTier:     agent.TierActWithApproval,
}

const activeStatusNote = "Active records are offered when people book; Inactive ones are " +
	"kept on file and no longer offered."
