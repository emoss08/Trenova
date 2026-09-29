package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const (
	mdCode               = "code"
	mdName               = "name"
	mdDescription        = "description"
	mdAddressLine1       = "addressLine1"
	mdAddressLine2       = "addressLine2"
	mdCity               = "city"
	mdState              = "state"
	mdPostalCode         = "postalCode"
	mdPhone              = "phone"
	mdEmail              = "email"
	mdNotes              = "notes"
	mdDOTNumber          = "dotNumber"
	mdMCNumber           = "mcNumber"
	mdStatus             = "status"
	mdMake               = "make"
	mdModel              = "model"
	mdYear               = "year"
	mdVIN                = "vin"
	mdLicensePlate       = "licensePlateNumber"
	mdRegistrationNumber = "registrationNumber"
	mdRegistrationExpiry = "registrationExpiry"
	mdEquipmentTypeID    = "equipmentTypeId"
	mdManufacturerID     = "equipmentManufacturerId"
	mdFleetCodeID        = "fleetCodeId"
	mdExternalID         = "externalId"

	actionUpdated      = "updated"
	actionDeleted      = "deleted"
	alertCustomMessage = "customMessage"
	alertKind          = "change alert"
	alertTableName     = "tableName"
	scannedDocument    = "scanned document"

	mdMaxName         = 255
	mdMaxShortName    = 100
	mdMaxCode         = 10
	mdMaxAddress      = 150
	mdMaxCity         = 100
	mdMaxPostalCode   = 10
	mdMaxPhone        = 20
	mdMaxNumber       = 12
	mdMaxNotes        = 5000
	mdMaxUnitText     = 50
	mdMaxExternalID   = 100
	mdMinYear         = 1950
	mdMaxYear         = 2100
	mdMaxTemperature  = 200
	mdMinTemperature  = -100
	mdMaxPaymentTerm  = 365
	mdMaxConsolidPrio = 10

	mdTaintHold = "A record read from a document or message someone outside sent is proposed, " +
		"since colleagues book and pay against it."
)

var (
	activeStatuses = agenttoolschema.Source("masterData.status", domaintypes.StatusValues())

	mdInternalAsk = masterPolicy{
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		taintHold:   mdTaintHold,
	}
	mdInternalStatus = masterPolicy{
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
	}
)

const activeStatusNote = "Active records are offered when people book; Inactive ones are " +
	"kept on file and no longer offered."
