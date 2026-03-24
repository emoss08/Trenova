package sequenceconfigservice

import (
	"context"
	"slices"
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

	if len(doc.Configs) != 4 {
		multiErr.Add(
			"configs",
			errortypes.ErrInvalid,
			"Exactly four sequence configurations are required",
		)
	}

	requiredTypes := map[tenant.SequenceType]bool{
		tenant.SequenceTypeProNumber:     false,
		tenant.SequenceTypeConsolidation: false,
		tenant.SequenceTypeInvoice:       false,
		tenant.SequenceTypeWorkOrder:     false,
	}

	for i, cfg := range doc.Configs {
		cfgErr := multiErr.WithIndex("configs", i)
		if cfg == nil {
			cfgErr.Add("", errortypes.ErrRequired, "Sequence config is required")
			continue
		}

		if _, ok := requiredTypes[cfg.SequenceType]; !ok {
			cfgErr.Add("sequenceType", errortypes.ErrInvalid, "Invalid sequence type")
		} else if requiredTypes[cfg.SequenceType] {
			cfgErr.Add("sequenceType", errortypes.ErrDuplicate, "Sequence type can only appear once")
		} else {
			requiredTypes[cfg.SequenceType] = true
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
	}

	for requiredType, seen := range requiredTypes {
		if !seen {
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
