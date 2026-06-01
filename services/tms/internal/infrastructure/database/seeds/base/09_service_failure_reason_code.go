package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type ServiceFailureReasonCodeSeed struct {
	seedhelpers.BaseSeed
}

func NewServiceFailureReasonCodeSeed() *ServiceFailureReasonCodeSeed {
	seed := &ServiceFailureReasonCodeSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"ServiceFailureReasonCode",
		"1.0.0",
		"Creates default service failure reason codes for tenant operations",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

type serviceFailureReasonCodeSeedDefinition struct {
	code                 string
	label                string
	description          string
	category             servicefailure.ReasonCategory
	appliesTo            servicefailure.ReasonCodeAppliesTo
	defaultStatusCode    string
	defaultReasonCode    string
	defaultExceptionCode string
	defaultNote          string
	sortOrder            int32
}

var defaultServiceFailureReasonCodes = []serviceFailureReasonCodeSeedDefinition{
	{
		code:              "LATE_PICKUP",
		label:             "Late Pickup",
		description:       "Pickup completed after the scheduled cutoff and service failure grace period.",
		category:          servicefailure.ReasonCategoryCarrier,
		appliesTo:         servicefailure.ReasonCodeAppliesToPickup,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Late pickup service failure.",
		sortOrder:         10,
	},
	{
		code:              "LATE_DELIVERY",
		label:             "Late Delivery",
		description:       "Delivery completed after the scheduled cutoff and service failure grace period.",
		category:          servicefailure.ReasonCategoryCarrier,
		appliesTo:         servicefailure.ReasonCodeAppliesToDelivery,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Late delivery service failure.",
		sortOrder:         20,
	},
	{
		code:              "FACILITY_DELAY",
		label:             "Facility Delay",
		description:       "Facility, dock, or appointment conditions caused the service failure.",
		category:          servicefailure.ReasonCategoryFacility,
		appliesTo:         servicefailure.ReasonCodeAppliesToBoth,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Facility-driven service failure.",
		sortOrder:         30,
	},
	{
		code:              "WEATHER_DELAY",
		label:             "Weather Delay",
		description:       "Weather conditions caused or contributed to the service failure.",
		category:          servicefailure.ReasonCategoryWeather,
		appliesTo:         servicefailure.ReasonCodeAppliesToBoth,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Weather-related service failure.",
		sortOrder:         40,
	},
	{
		code:              "EQUIPMENT_FAILURE",
		label:             "Equipment Failure",
		description:       "Power, trailer, or related equipment issue caused the service failure.",
		category:          servicefailure.ReasonCategoryEquipment,
		appliesTo:         servicefailure.ReasonCodeAppliesToBoth,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Equipment-related service failure.",
		sortOrder:         50,
	},
	{
		code:              "CUSTOMER_DELAY",
		label:             "Customer Delay",
		description:       "Customer action, availability, or instruction caused the service failure.",
		category:          servicefailure.ReasonCategoryCustomer,
		appliesTo:         servicefailure.ReasonCodeAppliesToBoth,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Customer-driven service failure.",
		sortOrder:         60,
	},
	{
		code:              "DOCUMENT_DELAY",
		label:             "Documentation Delay",
		description:       "Documentation issue caused or contributed to the service failure.",
		category:          servicefailure.ReasonCategoryDocumentation,
		appliesTo:         servicefailure.ReasonCodeAppliesToBoth,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Documentation-related service failure.",
		sortOrder:         70,
	},
	{
		code:              "OTHER_SERVICE_FAILURE",
		label:             "Other Service Failure",
		description:       "Operational service failure that does not fit another default reason.",
		category:          servicefailure.ReasonCategoryOther,
		appliesTo:         servicefailure.ReasonCodeAppliesToBoth,
		defaultStatusCode: "SD",
		defaultReasonCode: "NS",
		defaultNote:       "Service failure.",
		sortOrder:         100,
	},
	{
		code:              "X12_AF_ACCIDENT",
		label:             "X12 AF Accident",
		description:       "X12 214 AF accident service failure reason.",
		category:          servicefailure.ReasonCategoryOther,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "AF",
		sortOrder:         210,
	},
	{
		code:              "X12_AH_DRIVER_RELATED",
		label:             "X12 AH Driver Related",
		description:       "X12 214 AH driver related service failure reason.",
		category:          servicefailure.ReasonCategoryDriver,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "AH",
		sortOrder:         220,
	},
	{
		code:              "X12_AI_MECHANICAL_BREAKDOWN",
		label:             "X12 AI Mechanical Breakdown",
		description:       "X12 214 AI mechanical breakdown service failure reason.",
		category:          servicefailure.ReasonCategoryEquipment,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "AI",
		sortOrder:         230,
	},
	{
		code:              "X12_AO_WEATHER",
		label:             "X12 AO Weather",
		description:       "X12 214 AO weather service failure reason.",
		category:          servicefailure.ReasonCategoryWeather,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "AO",
		sortOrder:         240,
	},
	{
		code:              "X12_BE_ROAD_CONDITIONS",
		label:             "X12 BE Road Conditions",
		description:       "X12 214 BE road conditions service failure reason.",
		category:          servicefailure.ReasonCategoryWeather,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "BE",
		sortOrder:         250,
	},
	{
		code:              "X12_D1_DISPATCH_ERROR",
		label:             "X12 D1 Dispatch Error",
		description:       "X12 214 D1 dispatch error service failure reason.",
		category:          servicefailure.ReasonCategoryCarrier,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "D1",
		sortOrder:         260,
	},
	{
		code:              "X12_D2_DRIVER_NOT_AVAILABLE",
		label:             "X12 D2 Driver Not Available",
		description:       "X12 214 D2 driver not available service failure reason.",
		category:          servicefailure.ReasonCategoryDriver,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "D2",
		sortOrder:         270,
	},
	{
		code:              "X12_T3_TRAILER_NOT_AVAILABLE",
		label:             "X12 T3 Trailer Not Available",
		description:       "X12 214 T3 trailer not available service failure reason.",
		category:          servicefailure.ReasonCategoryEquipment,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "T3",
		sortOrder:         280,
	},
	{
		code:              "X12_T7_INSUFFICIENT_DELIVERY_TIME",
		label:             "X12 T7 Insufficient Delivery Time",
		description:       "X12 214 T7 insufficient delivery time service failure reason.",
		category:          servicefailure.ReasonCategoryAppointment,
		appliesTo:         servicefailure.ReasonCodeAppliesToDelivery,
		defaultStatusCode: "SD",
		defaultReasonCode: "T7",
		sortOrder:         290,
	},
	{
		code:              "X12_UG_WAITING_PICKUP",
		label:             "X12 UG Waiting Pickup",
		description:       "X12 214 UG waiting pickup service failure reason.",
		category:          servicefailure.ReasonCategoryShipper,
		appliesTo:         servicefailure.ReasonCodeAppliesToPickup,
		defaultStatusCode: "SD",
		defaultReasonCode: "UG",
		sortOrder:         300,
	},
	{
		code:              "X12_UH_WAITING_UNLOAD",
		label:             "X12 UH Waiting Unload",
		description:       "X12 214 UH waiting unload service failure reason.",
		category:          servicefailure.ReasonCategoryConsignee,
		appliesTo:         servicefailure.ReasonCodeAppliesToDelivery,
		defaultStatusCode: "SD",
		defaultReasonCode: "UH",
		sortOrder:         310,
	},
	{
		code:              "X12_AL_PREVIOUS_STOP",
		label:             "X12 AL Previous Stop",
		description:       "X12 214 AL previous stop service failure reason.",
		category:          servicefailure.ReasonCategoryAppointment,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "AL",
		sortOrder:         320,
	},
	{
		code:              "X12_AM_SHIPPER_RELATED",
		label:             "X12 AM Shipper Related",
		description:       "X12 214 AM shipper related service failure reason.",
		category:          servicefailure.ReasonCategoryShipper,
		appliesTo:         servicefailure.ReasonCodeAppliesToPickup,
		defaultStatusCode: "SD",
		defaultReasonCode: "AM",
		sortOrder:         330,
	},
	{
		code:              "X12_AG_CONSIGNEE_RELATED",
		label:             "X12 AG Consignee Related",
		description:       "X12 214 AG consignee related service failure reason.",
		category:          servicefailure.ReasonCategoryConsignee,
		appliesTo:         servicefailure.ReasonCodeAppliesToDelivery,
		defaultStatusCode: "SD",
		defaultReasonCode: "AG",
		sortOrder:         340,
	},
	{
		code:              "X12_B1_CONSIGNEE_CLOSED",
		label:             "X12 B1 Consignee Closed",
		description:       "X12 214 B1 consignee closed service failure reason.",
		category:          servicefailure.ReasonCategoryConsignee,
		appliesTo:         servicefailure.ReasonCodeAppliesToDelivery,
		defaultStatusCode: "SD",
		defaultReasonCode: "B1",
		sortOrder:         350,
	},
	{
		code:              "X12_B8_UNLOADING_FACILITY_EQUIPMENT",
		label:             "X12 B8 Unloading Facility Equipment",
		description:       "X12 214 B8 unloading facility equipment service failure reason.",
		category:          servicefailure.ReasonCategoryFacility,
		appliesTo:         servicefailure.ReasonCodeAppliesToDelivery,
		defaultStatusCode: "SD",
		defaultReasonCode: "B8",
		sortOrder:         360,
	},
	{
		code:              "X12_B9_RECEIVING_TIME_RESTRICTED",
		label:             "X12 B9 Receiving Time Restricted",
		description:       "X12 214 B9 receiving time restricted service failure reason.",
		category:          servicefailure.ReasonCategoryAppointment,
		appliesTo:         servicefailure.ReasonCodeAppliesToDelivery,
		defaultStatusCode: "SD",
		defaultReasonCode: "B9",
		sortOrder:         370,
	},
	{
		code:              "X12_BG_OTHER",
		label:             "X12 BG Other",
		description:       "X12 214 BG other service failure reason.",
		category:          servicefailure.ReasonCategoryOther,
		appliesTo:         servicefailure.ReasonCodeAppliesToAll,
		defaultStatusCode: "SD",
		defaultReasonCode: "BG",
		sortOrder:         380,
	},
}

func (s *ServiceFailureReasonCodeSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			var orgs []tenant.Organization
			if err := tx.NewSelect().Model(&orgs).Order("created_at ASC").Scan(ctx); err != nil {
				return fmt.Errorf("get organizations: %w", err)
			}

			if len(orgs) == 0 {
				return fmt.Errorf("no organizations found")
			}

			var createdCount int
			var createdOrgCount int
			for i := range orgs {
				count, err := s.createMissingReasonCodes(ctx, tx, sc, &orgs[i])
				if err != nil {
					return fmt.Errorf(
						"create default service failure reason codes for org %s: %w",
						orgs[i].Name,
						err,
					)
				}
				if count == 0 {
					continue
				}
				createdCount += count
				createdOrgCount++
			}

			if createdCount > 0 {
				seedhelpers.LogSuccess(
					"Created service failure reason code fixtures",
					fmt.Sprintf("- Created defaults for %d organizations", createdOrgCount),
					fmt.Sprintf("- Created %d service failure reason codes", createdCount),
				)
			}

			return nil
		},
	)
}

func (s *ServiceFailureReasonCodeSeed) createMissingReasonCodes(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	org *tenant.Organization,
) (int, error) {
	existingCodes, err := s.existingCodes(ctx, tx, org.ID, org.BusinessUnitID)
	if err != nil {
		return 0, err
	}

	entities := make([]*servicefailure.ReasonCode, 0, len(defaultServiceFailureReasonCodes))
	for _, def := range defaultServiceFailureReasonCodes {
		if existingCodes[strings.ToUpper(def.code)] {
			continue
		}

		entities = append(entities, &servicefailure.ReasonCode{
			ID:                   pulid.MustNew("sfrc_"),
			OrganizationID:       org.ID,
			BusinessUnitID:       org.BusinessUnitID,
			Code:                 def.code,
			Label:                def.label,
			Description:          def.description,
			Category:             def.category,
			AppliesTo:            def.appliesTo,
			DefaultStatusCode:    def.defaultStatusCode,
			DefaultReasonCode:    def.defaultReasonCode,
			DefaultExceptionCode: def.defaultExceptionCode,
			DefaultNote:          def.defaultNote,
			Active:               true,
			SortOrder:            def.sortOrder,
		})
	}

	if len(entities) == 0 {
		return 0, nil
	}

	if _, err = tx.NewInsert().Model(&entities).Exec(ctx); err != nil {
		return 0, fmt.Errorf("insert default service failure reason codes: %w", err)
	}

	for _, entity := range entities {
		if err = sc.TrackCreated(ctx, "service_failure_reason_codes", entity.ID, s.Name()); err != nil {
			return 0, fmt.Errorf("track service failure reason code: %w", err)
		}
	}

	return len(entities), nil
}

func (s *ServiceFailureReasonCodeSeed) existingCodes(
	ctx context.Context,
	tx bun.Tx,
	orgID pulid.ID,
	buID pulid.ID,
) (map[string]bool, error) {
	codes := make([]string, 0, len(defaultServiceFailureReasonCodes))
	for _, def := range defaultServiceFailureReasonCodes {
		codes = append(codes, def.code)
	}

	existingRows := make([]string, 0, len(defaultServiceFailureReasonCodes))
	if err := tx.NewSelect().
		Model((*servicefailure.ReasonCode)(nil)).
		Column("code").
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("code IN (?)", bun.In(codes)).
		Scan(ctx, &existingRows); err != nil {
		return nil, fmt.Errorf("get existing service failure reason codes: %w", err)
	}

	existing := make(map[string]bool, len(existingRows))
	for _, code := range existingRows {
		existing[strings.ToUpper(code)] = true
	}

	return existing, nil
}

func (s *ServiceFailureReasonCodeSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *ServiceFailureReasonCodeSeed) CanRollback() bool {
	return true
}
