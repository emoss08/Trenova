/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package hazmatsegreationrulevalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// ValidatorParams defines the dependencies required for initializing the Validator.
// This includes the database connection.
type ValidatorParams struct {
	fx.In

	DB                      db.Connection
	ValidationEngineFactory framework.ValidationEngineFactory
}

// Validator is a struct that contains the database connection and the validator.
// It provides methods to validate hazmat segregation rules and shipments.
type Validator struct {
	db  db.Connection
	vef framework.ValidationEngineFactory
}

// NewValidator initializes a new Validator with the provided database connection.
//
// Parameters:
//   - p: ValidatorParams containing the database connection.
//
// Returns:
//   - *Validator: A new Validator instance.
func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		db:  p.DB,
		vef: p.ValidationEngineFactory,
	}
}

// Validate validates a hazmat segregation rule.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - hsr: The hazmat segregation rule to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	hsr *hazmatsegregationrule.HazmatSegregationRule,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// * Basic Validation
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				hsr.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// * Validate Uniqueness
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				return v.ValidateUniqueness(ctx, valCtx, hsr, multiErr)
			},
		),
	)

	return engine.Validate(ctx)
}

// ValidateUniqueness validates the uniqueness of a hazmat segregation rule.
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - hsr: The hazmat segregation rule to validate.
//   - multiErr: The multi-error to add validation errors to.
//
// Returns:
//   - error: An error if the validation fails.
func (v *Validator) ValidateUniqueness(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	hsr *hazmatsegregationrule.HazmatSegregationRule,
	multiErr *errors.MultiError,
) error {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	vb := queryutils.NewUniquenessValidator(hsr.GetTableName()).
		WithTenant(hsr.OrganizationID, hsr.BusinessUnitID).
		WithModelName("HazmatSegregationRule").
		WithFieldAndTemplate("name", hsr.Name,
			"Hazmat Segregation Rule with name ':value' already exists in the organization.",
			map[string]string{
				"value": hsr.Name,
			})

	if valCtx.IsCreate {
		vb.WithOperation(queryutils.OperationCreate)
	} else {
		vb.WithOperation(queryutils.OperationUpdate).
			WithPrimaryKey("id", hsr.GetID())
	}

	queryutils.CheckFieldUniqueness(ctx, dba, vb.Build(), multiErr)

	return nil
}

// ValidateShipment checks if a shipment's commodities violate hazmat segregation rules
//
// Parameters:
//   - ctx: The context of the request.
//   - shp: The shipment to validate.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
//   - []*SegregationViolation: A list of segregation violations.
func (v *Validator) ValidateShipment(
	ctx context.Context,
	shp *shipment.Shipment,
) (*errors.MultiError, []*SegregationViolation) {
	multiErr := errors.NewMultiError()

	// Skip validation if shipment has no commodities or only one commodity
	if !shp.HasCommodities() || len(shp.Commodities) < 2 {
		return nil, nil
	}

	// Get commodities with hazmat and their IDs
	commoditiesWithHazmat, hazmatIDs := v.extractHazmatCommodities(shp)

	// Skip if no hazmat materials or only one
	if len(hazmatIDs) < 2 {
		return nil, nil
	}

	// Fetch hazmat data and rules
	hazmatMap, ruleMap, err := v.fetchHazmatDataAndRules(ctx, shp, hazmatIDs)
	if err != nil {
		multiErr.Add("hazmatSegregation", errors.ErrSystemError, err.Error())
		return multiErr, nil
	}

	// Attach hazmat data to commodities
	v.attachHazmatDataToCommodities(commoditiesWithHazmat, hazmatMap)

	// Check for violations
	violations := checkCommodityPairs(commoditiesWithHazmat, ruleMap)

	// Add validation errors for any violations
	if len(violations) > 0 {
		for i, violation := range violations {
			fieldName := fmt.Sprintf("commodities[%d].commodityId", i)
			multiErr.Add(fieldName, errors.ErrInvalid, violation.Message)
		}
		return multiErr, violations
	}

	return nil, nil
}

// extractHazmatCommodities extracts commodities with hazmat and their IDs
func (v *Validator) extractHazmatCommodities(
	shp *shipment.Shipment,
) ([]*shipment.ShipmentCommodity, []pulid.ID) {
	commoditiesWithHazmat := make([]*shipment.ShipmentCommodity, 0)
	hazmatIDs := make([]pulid.ID, 0)

	for _, com := range shp.Commodities {
		if com.Commodity != nil && com.Commodity.HazardousMaterialID != nil {
			hazmatIDs = append(hazmatIDs, *com.Commodity.HazardousMaterialID)
			commoditiesWithHazmat = append(commoditiesWithHazmat, com)
		}
	}

	return commoditiesWithHazmat, hazmatIDs
}

// fetchHazmatDataAndRules fetches hazmat data and rules from database
func (v *Validator) fetchHazmatDataAndRules(
	ctx context.Context,
	shp *shipment.Shipment,
	hazmatIDs []pulid.ID,
) (hazmatMap map[string]*hazardousmaterial.HazardousMaterial, ruleMap map[hazmatPair]*hazmatsegregationrule.HazmatSegregationRule, err error) {
	dba, err := v.db.DB(ctx)
	if err != nil {
		return nil, nil, eris.Wrap(err, "get database connection")
	}

	// Fetch hazmat materials
	hazmatMaterials := make([]*hazardousmaterial.HazardousMaterial, 0)
	err = dba.NewSelect().
		Model(&hazmatMaterials).
		Where("hm.id IN (?)", bun.In(hazmatIDs)).
		Where("hm.organization_id = ?", shp.OrganizationID).
		Where("hm.business_unit_id = ?", shp.BusinessUnitID).
		Scan(ctx)
	if err != nil {
		return nil, nil, eris.Wrap(err, "fetch hazardous materials")
	}

	// Create a map for quick lookup
	hazmatMap = make(map[string]*hazardousmaterial.HazardousMaterial)
	for _, hm := range hazmatMaterials {
		hazmatMap[hm.ID.String()] = hm
	}

	// Get all segregation rules
	rules := make([]*hazmatsegregationrule.HazmatSegregationRule, 0)
	err = dba.NewSelect().
		Model(&rules).
		Where("hsr.organization_id = ? AND hsr.business_unit_id = ?", shp.OrganizationID, shp.BusinessUnitID).
		Where("hsr.status = ?", "Active").
		Scan(ctx)
	if err != nil {
		return nil, nil, eris.Wrap(err, "fetch segregation rules")
	}

	// Build rule map
	ruleMap = buildRuleMap(rules)

	return hazmatMap, ruleMap, nil
}

// attachHazmatDataToCommodities attaches hazmat data to commodities
func (v *Validator) attachHazmatDataToCommodities(
	commodities []*shipment.ShipmentCommodity,
	hazmatMap map[string]*hazardousmaterial.HazardousMaterial,
) {
	for _, com := range commodities {
		if com.Commodity != nil && com.Commodity.HazardousMaterialID != nil {
			hazmatID := com.Commodity.HazardousMaterialID.String()
			if hazmat, ok := hazmatMap[hazmatID]; ok {
				com.Commodity.HazardousMaterial = hazmat
			}
		}
	}
}

// Helper function to build rule lookup map
func buildRuleMap(
	rules []*hazmatsegregationrule.HazmatSegregationRule,
) map[hazmatPair]*hazmatsegregationrule.HazmatSegregationRule {
	ruleMap := make(map[hazmatPair]*hazmatsegregationrule.HazmatSegregationRule)

	for _, rule := range rules {
		// Add class-level rules
		ruleMap[hazmatPair{
			classA: rule.ClassA,
			classB: rule.ClassB,
		}] = rule

		// Add the reversed pair too for easier lookup
		ruleMap[hazmatPair{
			classA: rule.ClassB,
			classB: rule.ClassA,
		}] = rule

		// Add specific material rules if applicable
		if rule.HazmatAID != nil && rule.HazmatBID != nil {
			ruleMap[hazmatPair{
				classA:    rule.ClassA,
				classB:    rule.ClassB,
				hazmatAID: rule.HazmatAID.String(),
				hazmatBID: rule.HazmatBID.String(),
			}] = rule

			// Add the reversed pair
			ruleMap[hazmatPair{
				classA:    rule.ClassB,
				classB:    rule.ClassA,
				hazmatAID: rule.HazmatBID.String(),
				hazmatBID: rule.HazmatAID.String(),
			}] = rule
		}
	}

	return ruleMap
}

// Helper function to check pairs of commodities for violations
func checkCommodityPairs(
	commodities []*shipment.ShipmentCommodity,
	ruleMap map[hazmatPair]*hazmatsegregationrule.HazmatSegregationRule,
) []*SegregationViolation {
	violations := make([]*SegregationViolation, 0)

	for i := range commodities {
		for j := i + 1; j < len(commodities); j++ {
			comA := commodities[i]
			comB := commodities[j]

			// * Skip if either commodity doesn't have hazmat data
			if comA.Commodity == nil || comA.Commodity.HazardousMaterial == nil ||
				comB.Commodity == nil || comB.Commodity.HazardousMaterial == nil {
				continue
			}

			hazA := comA.Commodity.HazardousMaterial
			hazB := comB.Commodity.HazardousMaterial

			// * First check for specific material rules
			specificPair := hazmatPair{
				classA:    hazA.Class,
				classB:    hazB.Class,
				hazmatAID: hazA.ID.String(),
				hazmatBID: hazB.ID.String(),
			}

			if rule, exists := ruleMap[specificPair]; exists {
				violations = append(
					violations,
					createViolation(rule, comA.Commodity, comB.Commodity, hazA, hazB),
				)
				continue
			}

			// * Then check for class-level rules
			classPair := hazmatPair{
				classA: hazA.Class,
				classB: hazB.Class,
			}

			if rule, exists := ruleMap[classPair]; exists {
				violations = append(
					violations,
					createViolation(rule, comA.Commodity, comB.Commodity, hazA, hazB),
				)
			}
		}
	}

	return violations
}

// loadCommoditiesWithHazmat ensures the commodities have their hazmat data loaded
//
// Parameters:
//   - ctx: The context of the request.
//   - shp: The shipment to load commodities for.
//
// Returns:
//   - []*shipment.ShipmentCommodity: A list of shipment commodities with hazmat data.
//   - error: An error if the loading fails.
func (v *Validator) loadCommoditiesWithHazmat(
	ctx context.Context,
	shp *shipment.Shipment,
) ([]*shipment.ShipmentCommodity, error) {
	// * If commodities are already fully loaded with hazmat data, return them
	if v.areCommoditiesFullyLoaded(shp.Commodities) {
		return shp.Commodities, nil
	}

	// * Otherwise, load the complete commodity data from the database
	dba, err := v.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	commodities := make([]*shipment.ShipmentCommodity, 0, len(shp.Commodities))

	err = dba.NewSelect().
		Model(&commodities).
		Where("sc.shipment_id = ?", shp.ID).
		Relation("Commodity").
		Relation("Commodity.HazardousMaterial").
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "load shipment commodities with hazmat data")
	}

	return commodities, nil
}

// areCommoditiesFullyLoaded checks if commodities have their hazmat data loaded
//
// Parameters:
//   - commodities: The commodities to check.
//
// Returns:
//   - bool: True if the commodities have their hazmat data loaded, false otherwise.
func (v *Validator) areCommoditiesFullyLoaded(commodities []*shipment.ShipmentCommodity) bool {
	for _, com := range commodities {
		if com.Commodity == nil {
			return false
		}
		// * We need to check if HazardousMaterial is nil, but we also need to check
		// * if HazardousMaterialID is set but HazardousMaterial is not loaded
		if com.Commodity.HazardousMaterialID != nil && com.Commodity.HazardousMaterial == nil {
			return false
		}
	}
	return true
}

// createViolation creates a segregation violation with a formatted message
//
// Parameters:
//   - rule: The rule that was violated.
//   - comA: The first commodity.
//   - comB: The second commodity.
//   - hazA: The first hazardous material.
//   - hazB: The second hazardous material.
//
// Returns:
//   - *SegregationViolation: A segregation violation.
func createViolation(
	rule *hazmatsegregationrule.HazmatSegregationRule,
	comA, comB *commodity.Commodity,
	hazA, hazB *hazardousmaterial.HazardousMaterial,
) *SegregationViolation {
	segregationType := string(rule.SegregationType)
	var message string

	switch rule.SegregationType {
	case hazmatsegregationrule.SegregationTypeProhibited:
		message = fmt.Sprintf(
			"Hazardous materials %s (%s) and %s (%s) cannot be transported together",
			comA.Name,
			hazA.Class,
			comB.Name,
			hazB.Class,
		)
	case hazmatsegregationrule.SegregationTypeBarrier:
		message = fmt.Sprintf(
			"Hazardous materials %s (%s) and %s (%s) must be separated by a barrier",
			comA.Name,
			hazA.Class,
			comB.Name,
			hazB.Class,
		)
	case hazmatsegregationrule.SegregationTypeSeparated:
		message = fmt.Sprintf(
			"Hazardous materials %s (%s) and %s (%s) must be separated from each other",
			comA.Name,
			hazA.Class,
			comB.Name,
			hazB.Class,
		)
	case hazmatsegregationrule.SegregationTypeDistance:
		distance := "unknown distance"
		if rule.MinimumDistance != nil {
			distance = fmt.Sprintf("%.2f %s", *rule.MinimumDistance, rule.DistanceUnit)
		}
		message = fmt.Sprintf(
			"Hazardous materials %s (%s) and %s (%s) must be separated by at least %s",
			comA.Name,
			hazA.Class,
			comB.Name,
			hazB.Class,
			distance,
		)
	default:
		message = fmt.Sprintf("Hazardous materials %s (%s) and %s (%s) require %s segregation",
			comA.Name, hazA.Class, comB.Name, hazB.Class, segregationType)
	}

	if rule.HasExceptions && rule.ExceptionNotes != "" {
		message += fmt.Sprintf(" (Note: %s)", rule.ExceptionNotes)
	}

	return &SegregationViolation{
		Rule:       rule,
		CommodityA: comA,
		CommodityB: comB,
		HazmatA:    hazA,
		HazmatB:    hazB,
		Message:    message,
	}
}

// ValidateShipmentCommodityAddition checks if adding a commodity to a shipment would violate segregation rules
//
// Parameters:
//   - ctx: The context of the request.
//   - shp: The shipment to validate.
//   - newCommodityID: The ID of the new commodity to add.
//
// Returns:
//   - *errors.MultiError: A list of validation errors.
//   - []*SegregationViolation: A list of segregation violations.
func (v *Validator) ValidateShipmentCommodityAddition(
	ctx context.Context, shp *shipment.Shipment, newCommodityID pulid.ID,
) (*errors.MultiError, []*SegregationViolation) {
	multiErr := errors.NewMultiError()

	// * Load the new commodity with hazmat information
	dba, err := v.db.DB(ctx)
	if err != nil {
		multiErr.Add(
			"hazmatSegregation",
			errors.ErrSystemError,
			"Failed to get database connection",
		)
		return multiErr, nil
	}

	// * Load the new commodity
	newCommodity := new(commodity.Commodity)
	err = dba.NewSelect().
		Model(newCommodity).
		Where("com.id = ?", newCommodityID).
		Where("com.organization_id = ?", shp.OrganizationID).
		Where("com.business_unit_id = ?", shp.BusinessUnitID).
		Relation("HazardousMaterial").
		Scan(ctx)
	if err != nil {
		multiErr.Add(
			"hazmatSegregation",
			errors.ErrSystemError,
			fmt.Sprintf("Failed to load commodity: %s", err.Error()),
		)
		return multiErr, nil
	}

	// * Skip validation if new commodity is not hazardous
	if newCommodity.HazardousMaterialID == nil || newCommodity.HazardousMaterial == nil {
		return nil, nil
	}

	// * Load shipment commodities if not already loaded
	commodities, err := v.loadCommoditiesWithHazmat(ctx, shp)
	if err != nil {
		multiErr.Add(
			"hazmatSegregation",
			errors.ErrSystemError,
			fmt.Sprintf("Failed to load shipment commodities: %s", err.Error()),
		)
		return multiErr, nil
	}

	// * Create a temporary shipment commodity for the new commodity
	newShipmentCommodity := &shipment.ShipmentCommodity{
		ShipmentID:     shp.ID,
		CommodityID:    newCommodityID,
		Commodity:      newCommodity,
		OrganizationID: shp.OrganizationID,
		BusinessUnitID: shp.BusinessUnitID,
		Pieces:         1, // * Default values, not important for validation
		Weight:         1,
	}

	// * Create a copy of the shipment with the new commodity added
	commodities = append(commodities, newShipmentCommodity)
	tempShipment := *shp
	tempShipment.Commodities = commodities

	// * Validate the temporary shipment
	return v.ValidateShipment(ctx, &tempShipment)
}

// BatchValidateShipments validates multiple shipments for hazmat segregation violations
//
// Parameters:
//   - ctx: The context of the request.
//   - shipments: The shipments to validate.
//
// Returns:
//   - map[string]*errors.MultiError: A map of shipment IDs to validation errors.
func (v *Validator) BatchValidateShipments(
	ctx context.Context,
	shipments []*shipment.Shipment,
) (errorResults map[string]*errors.MultiError, violationResults map[string][]*SegregationViolation) {
	errorResults = make(map[string]*errors.MultiError)
	violationResults = make(map[string][]*SegregationViolation)

	for _, shp := range shipments {
		multiErr, violations := v.ValidateShipment(ctx, shp)
		if multiErr != nil && multiErr.HasErrors() {
			errorResults[shp.ID.String()] = multiErr
		}
		if len(violations) > 0 {
			violationResults[shp.ID.String()] = violations
		}
	}

	return errorResults, violationResults
}
