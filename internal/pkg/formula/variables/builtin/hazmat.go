package builtin

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// * Hazmat-related variables

// * HasHazmatVar indicates if the shipment contains hazardous materials
var HasHazmatVar = variables.NewVariable(
	"has_hazmat",
	"Whether the shipment contains hazardous materials",
	formula.ValueTypeBoolean,
	variables.CategoryHazmat,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetComputed("computeHasHazmat")
	},
)

// * HazmatClassVar returns the hazmat class if present
var HazmatClassVar = variables.NewVariable(
	"hazmat_class",
	"Hazardous material class (e.g., 3 for Flammable Liquids)",
	formula.ValueTypeString,
	variables.CategoryHazmat,
	hazmatClassResolver,
)

// * HazmatClassesVar returns all hazmat classes in the shipment
var HazmatClassesVar = variables.NewVariable(
	"hazmat_classes",
	"All hazardous material classes in the shipment",
	formula.ValueTypeArray,
	variables.CategoryHazmat,
	hazmatClassesResolver,
)

// * HazmatUNNumberVar returns the UN number if present
var HazmatUNNumberVar = variables.NewVariable(
	"hazmat_un_number",
	"UN identification number for hazardous material",
	formula.ValueTypeString,
	variables.CategoryHazmat,
	hazmatUNNumberResolver,
)

// * HazmatPackingGroupVar returns the packing group if present
var HazmatPackingGroupVar = variables.NewVariable(
	"hazmat_packing_group",
	"Hazmat packing group (I, II, or III)",
	formula.ValueTypeString,
	variables.CategoryHazmat,
	hazmatPackingGroupResolver,
)

// * IsExplosiveVar checks if shipment contains explosives (Class 1)
var IsExplosiveVar = variables.NewVariable(
	"is_explosive",
	"Whether the shipment contains explosive materials (Class 1)",
	formula.ValueTypeBoolean,
	variables.CategoryHazmat,
	func(ctx variables.VariableContext) (any, error) {
		// Check for any Class 1 hazmat
		entity := ctx.GetEntity()
		s, ok := entity.(*shipment.Shipment)
		if !ok {
			return false, fmt.Errorf("entity is not a shipment")
		}
		
		explosiveClasses := []hazardousmaterial.HazardousClass{
			hazardousmaterial.HazardousClass1And1,
			hazardousmaterial.HazardousClass1And2,
			hazardousmaterial.HazardousClass1And3,
			hazardousmaterial.HazardousClass1And4,
			hazardousmaterial.HazardousClass1And5,
			hazardousmaterial.HazardousClass1And6,
		}
		
		for _, sc := range s.Commodities {
			if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
				for _, explosiveClass := range explosiveClasses {
					if sc.Commodity.HazardousMaterial.Class == explosiveClass {
						return true, nil
					}
				}
			}
		}
		
		return false, nil
	},
)

// * IsFlammableVar checks if shipment contains flammable materials (Class 3)
var IsFlammableVar = variables.NewVariable(
	"is_flammable",
	"Whether the shipment contains flammable liquids (Class 3)",
	formula.ValueTypeBoolean,
	variables.CategoryHazmat,
	func(ctx variables.VariableContext) (any, error) {
		return hasHazmatClass(ctx, hazardousmaterial.HazardousClass3)
	},
)

// * IsCorrosiveVar checks if shipment contains corrosive materials (Class 8)
var IsCorrosiveVar = variables.NewVariable(
	"is_corrosive",
	"Whether the shipment contains corrosive materials (Class 8)",
	formula.ValueTypeBoolean,
	variables.CategoryHazmat,
	func(ctx variables.VariableContext) (any, error) {
		return hasHazmatClass(ctx, hazardousmaterial.HazardousClass8)
	},
)

// * hazmatClassResolver gets the first hazmat class found
func hazmatClassResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return "", fmt.Errorf("entity is not a shipment")
	}
	
	// * Check each commodity for hazmat
	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			return string(sc.Commodity.HazardousMaterial.Class), nil
		}
	}
	
	return "", nil
}

// * hazmatClassesResolver gets all unique hazmat classes
func hazmatClassesResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return []string{}, fmt.Errorf("entity is not a shipment")
	}
	
	classMap := make(map[string]bool)
	
	// * Collect all unique hazmat classes
	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			classMap[string(sc.Commodity.HazardousMaterial.Class)] = true
		}
	}
	
	// * Convert to array
	classes := make([]string, 0, len(classMap))
	for class := range classMap {
		classes = append(classes, class)
	}
	
	return classes, nil
}

// * hazmatUNNumberResolver gets the first UN number found
func hazmatUNNumberResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return "", fmt.Errorf("entity is not a shipment")
	}
	
	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			return sc.Commodity.HazardousMaterial.UNNumber, nil
		}
	}
	
	return "", nil
}

// * hazmatPackingGroupResolver gets the first packing group found
func hazmatPackingGroupResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return "", fmt.Errorf("entity is not a shipment")
	}
	
	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			return string(sc.Commodity.HazardousMaterial.PackingGroup), nil
		}
	}
	
	return "", nil
}

// * hasHazmatClass checks if shipment contains a specific hazmat class
func hasHazmatClass(ctx variables.VariableContext, targetClass hazardousmaterial.HazardousClass) (bool, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return false, fmt.Errorf("entity is not a shipment")
	}
	
	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			if sc.Commodity.HazardousMaterial.Class == targetClass {
				return true, nil
			}
		}
	}
	
	return false, nil
}

// * HazmatClassNameVar returns the human-readable hazmat class name
var HazmatClassNameVar = variables.NewVariable(
	"hazmat_class_name",
	"Human-readable name of the hazmat class",
	formula.ValueTypeString,
	variables.CategoryHazmat,
	hazmatClassNameResolver,
)

// * hazmatClassNameResolver converts class to readable name
func hazmatClassNameResolver(ctx variables.VariableContext) (any, error) {
	classStr, err := hazmatClassResolver(ctx)
	if err != nil {
		return "", err
	}
	
	class, ok := classStr.(string)
	if !ok || class == "" {
		return "", nil
	}
	
	// * Convert to hazardousmaterial.HazardousClass and get name
	hazClass := hazardousmaterial.HazardousClass(class)
	
	// * Map class to name
	classNames := map[hazardousmaterial.HazardousClass]string{
		hazardousmaterial.HazardousClass1And1: "Explosives (Division 1.1)",
		hazardousmaterial.HazardousClass1And2: "Explosives (Division 1.2)",
		hazardousmaterial.HazardousClass1And3: "Explosives (Division 1.3)",
		hazardousmaterial.HazardousClass1And4: "Explosives (Division 1.4)",
		hazardousmaterial.HazardousClass1And5: "Explosives (Division 1.5)",
		hazardousmaterial.HazardousClass1And6: "Explosives (Division 1.6)",
		hazardousmaterial.HazardousClass2And1: "Flammable Gas",
		hazardousmaterial.HazardousClass2And2: "Non-Flammable Gas",
		hazardousmaterial.HazardousClass2And3: "Toxic Gas",
		hazardousmaterial.HazardousClass3:     "Flammable Liquids",
		hazardousmaterial.HazardousClass4And1: "Flammable Solids",
		hazardousmaterial.HazardousClass4And2: "Spontaneously Combustible",
		hazardousmaterial.HazardousClass4And3: "Dangerous When Wet",
		hazardousmaterial.HazardousClass5And1: "Oxidizers",
		hazardousmaterial.HazardousClass5And2: "Organic Peroxides",
		hazardousmaterial.HazardousClass6And1: "Toxic Substances",
		hazardousmaterial.HazardousClass6And2: "Infectious Substances",
		hazardousmaterial.HazardousClass7:     "Radioactive Materials",
		hazardousmaterial.HazardousClass8:     "Corrosive Materials",
		hazardousmaterial.HazardousClass9:     "Miscellaneous Dangerous Goods",
	}
	
	if name, found := classNames[hazClass]; found {
		return name, nil
	}
	
	// * Return the class as-is if no mapping found
	return strings.ReplaceAll(string(hazClass), "_", "."), nil
}

// * RegisterHazmatVariables registers all hazmat-related variables
func RegisterHazmatVariables(registry *variables.Registry) {
	registry.MustRegister(HasHazmatVar)
	registry.MustRegister(HazmatClassVar)
	registry.MustRegister(HazmatClassesVar)
	registry.MustRegister(HazmatClassNameVar)
	registry.MustRegister(HazmatUNNumberVar)
	registry.MustRegister(HazmatPackingGroupVar)
	registry.MustRegister(IsExplosiveVar)
	registry.MustRegister(IsFlammableVar)
	registry.MustRegister(IsCorrosiveVar)
}