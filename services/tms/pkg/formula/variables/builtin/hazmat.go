package builtin

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/formula/variables"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

var HasHazmatVar = variables.NewVariable(
	"has_hazmat",
	"Whether the shipment contains hazardous materials",
	formulatypes.ValueTypeBoolean,
	variables.SourceHazmat,
	func(ctx variables.VariableContext) (any, error) {
		return ctx.GetComputed("computeHasHazmat")
	},
)

var HazmatClassVar = variables.NewVariable(
	"hazmat_class",
	"Hazardous material class (e.g., 3 for Flammable Liquids)",
	formulatypes.ValueTypeString,
	variables.SourceHazmat,
	hazmatClassResolver,
)

var HazmatClassesVar = variables.NewVariable(
	"hazmat_classes",
	"All hazardous material classes in the shipment",
	formulatypes.ValueTypeArray,
	variables.SourceHazmat,
	hazmatClassesResolver,
)

var HazmatUNNumberVar = variables.NewVariable(
	"hazmat_un_number",
	"UN identification number for hazardous material",
	formulatypes.ValueTypeString,
	variables.SourceHazmat,
	hazmatUNNumberResolver,
)

var HazmatPackingGroupVar = variables.NewVariable(
	"hazmat_packing_group",
	"Hazmat packing group (I, II, or III)",
	formulatypes.ValueTypeString,
	variables.SourceHazmat,
	hazmatPackingGroupResolver,
)

var IsExplosiveVar = variables.NewVariable(
	"is_explosive",
	"Whether the shipment contains explosive materials (Class 1)",
	formulatypes.ValueTypeBoolean,
	variables.SourceHazmat,
	func(ctx variables.VariableContext) (any, error) {
		entity := ctx.GetEntity()
		s, ok := entity.(*shipment.Shipment)
		if !ok {
			return false, ErrEntityNotShipment
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

var IsFlammableVar = variables.NewVariable(
	"is_flammable",
	"Whether the shipment contains flammable liquids (Class 3)",
	formulatypes.ValueTypeBoolean,
	variables.SourceHazmat,
	func(ctx variables.VariableContext) (any, error) {
		return hasHazmatClass(ctx, hazardousmaterial.HazardousClass3)
	},
)

var IsCorrosiveVar = variables.NewVariable(
	"is_corrosive",
	"Whether the shipment contains corrosive materials (Class 8)",
	formulatypes.ValueTypeBoolean,
	variables.SourceHazmat,
	func(ctx variables.VariableContext) (any, error) {
		return hasHazmatClass(ctx, hazardousmaterial.HazardousClass8)
	},
)

func hazmatClassResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return "", ErrEntityNotShipment
	}

	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			return string(sc.Commodity.HazardousMaterial.Class), nil
		}
	}

	return "", nil
}

func hazmatClassesResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return []string{}, ErrEntityNotShipment
	}

	classMap := make(map[string]bool)

	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			classMap[string(sc.Commodity.HazardousMaterial.Class)] = true
		}
	}

	classes := make([]string, 0, len(classMap))
	for class := range classMap {
		classes = append(classes, class)
	}

	return classes, nil
}

func hazmatUNNumberResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return "", ErrEntityNotShipment
	}

	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			return sc.Commodity.HazardousMaterial.UNNumber, nil
		}
	}

	return "", nil
}

func hazmatPackingGroupResolver(ctx variables.VariableContext) (any, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return "", ErrEntityNotShipment
	}

	for _, sc := range s.Commodities {
		if sc.Commodity != nil && sc.Commodity.HazardousMaterial != nil {
			return string(sc.Commodity.HazardousMaterial.PackingGroup), nil
		}
	}

	return "", nil
}

func hasHazmatClass(
	ctx variables.VariableContext,
	targetClass hazardousmaterial.HazardousClass,
) (bool, error) {
	entity := ctx.GetEntity()
	s, ok := entity.(*shipment.Shipment)
	if !ok {
		return false, ErrEntityNotShipment
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

var HazmatClassNameVar = variables.NewVariable(
	"hazmat_class_name",
	"Human-readable name of the hazmat class",
	formulatypes.ValueTypeString,
	variables.SourceHazmat,
	hazmatClassNameResolver,
)

func hazmatClassNameResolver(ctx variables.VariableContext) (any, error) {
	classStr, err := hazmatClassResolver(ctx)
	if err != nil {
		return "", err
	}

	class, ok := classStr.(string)
	if !ok || class == "" {
		return "", nil
	}

	hazClass := hazardousmaterial.HazardousClass(class)

	classNames := map[hazardousmaterial.HazardousClass]string{
		hazardousmaterial.HazardousClass1:     "Explosives (Division 1)",
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

	return strings.ReplaceAll(string(hazClass), "_", "."), nil
}

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
