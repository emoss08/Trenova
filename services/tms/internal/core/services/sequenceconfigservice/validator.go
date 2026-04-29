package sequenceconfigservice

import (
	"context"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateUpdate(
	_ context.Context,
	doc *tenant.SequenceConfigDocument,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if doc == nil {
		multiErr.Add("", errortypes.ErrRequired, "Sequence configuration payload is required")
		return multiErr
	}

	if doc.OrganizationID.IsNil() {
		multiErr.Add("organizationId", errortypes.ErrRequired, "Organization ID is required")
	}
	if doc.BusinessUnitID.IsNil() {
		multiErr.Add("businessUnitId", errortypes.ErrRequired, "Business unit ID is required")
	}

	requiredTypes := tenant.RequiredSequenceTypes()
	if len(doc.Configs) != len(requiredTypes) {
		multiErr.Add(
			"configs",
			errortypes.ErrInvalid,
			"Exactly "+strconv.Itoa(len(requiredTypes))+" sequence configurations are required",
		)
	}

	requiredTypeSet := make(map[tenant.SequenceType]struct{}, len(requiredTypes))
	seenTypes := make(map[tenant.SequenceType]bool, len(requiredTypes))
	for _, sequenceType := range requiredTypes {
		requiredTypeSet[sequenceType] = struct{}{}
		seenTypes[sequenceType] = false
	}

	for i, cfg := range doc.Configs {
		cfgErr := multiErr.WithIndex("configs", i)
		if cfg == nil {
			cfgErr.Add("", errortypes.ErrRequired, "Sequence config is required")
			continue
		}

		if _, ok := requiredTypeSet[cfg.SequenceType]; !ok {
			cfgErr.Add("sequenceType", errortypes.ErrInvalid, "Invalid sequence type")
		} else if seenTypes[cfg.SequenceType] {
			cfgErr.Add("sequenceType", errortypes.ErrDuplicate, "Sequence type can only appear once")
		} else {
			seenTypes[cfg.SequenceType] = true
		}

		if strings.TrimSpace(cfg.Prefix) == "" {
			cfgErr.Add("prefix", errortypes.ErrRequired, "Prefix is required")
		}
		if cfg.SequenceDigits < 1 || cfg.SequenceDigits > 10 {
			cfgErr.Add(
				"sequenceDigits",
				errortypes.ErrInvalid,
				"Sequence digits must be between 1 and 10",
			)
		}
		if cfg.IncludeYear && (cfg.YearDigits < 2 || cfg.YearDigits > 4) {
			cfgErr.Add(
				"yearDigits",
				errortypes.ErrInvalid,
				"Year digits must be between 2 and 4 when include year is enabled",
			)
		}
		if cfg.IncludeRandomDigits && (cfg.RandomDigitsCount < 1 || cfg.RandomDigitsCount > 10) {
			cfgErr.Add(
				"randomDigitsCount",
				errortypes.ErrInvalid,
				"Random digits count must be between 1 and 10 when include random digits is enabled",
			)
		}
		if !cfg.IncludeRandomDigits {
			cfg.RandomDigitsCount = 0
		}
		if cfg.UseSeparators {
			if strings.TrimSpace(cfg.SeparatorChar) == "" {
				cfgErr.Add(
					"separatorChar",
					errortypes.ErrRequired,
					"Separator character is required when separators are enabled",
				)
			} else if !slices.Contains([]string{"-", "_", "/", "."}, cfg.SeparatorChar) {
				cfgErr.Add("separatorChar", errortypes.ErrInvalid, "Separator must be one of '-', '_', '/', '.'")
			}
		} else {
			cfg.SeparatorChar = ""
		}
		if cfg.AllowCustomFormat && strings.TrimSpace(cfg.CustomFormat) == "" {
			cfgErr.Add(
				"customFormat",
				errortypes.ErrRequired,
				"Custom format is required when custom format is enabled",
			)
		}
		if !cfg.AllowCustomFormat {
			cfg.CustomFormat = ""
		}
		if cfg.SequenceType == tenant.SequenceTypeLocationCode {
			cfg.LocationCodeStrategy = tenant.EffectiveLocationCodeStrategy(cfg.LocationCodeStrategy)
			if err := cfg.LocationCodeStrategy.Validate(); err != nil {
				cfgErr.Add(
					"locationCodeStrategy",
					errortypes.ErrInvalid,
					err.Error(),
				)
			}
		} else {
			cfg.LocationCodeStrategy = nil
		}
	}

	sort.Slice(requiredTypes, func(i, j int) bool {
		return tenant.SequenceTypeSortOrder(requiredTypes[i]) < tenant.SequenceTypeSortOrder(requiredTypes[j])
	})

	for _, requiredType := range requiredTypes {
		if !seenTypes[requiredType] {
			multiErr.Add(
				"configs",
				errortypes.ErrRequired,
				"Missing sequence type: "+string(requiredType),
			)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
