package sequenceconfigresolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/intutils"
)

func sequenceConfigDocumentFromInput(
	input *gqlmodel.SequenceConfigDocumentInput,
	authCtx *authctx.AuthContext,
) (*sequenceconfig.SequenceConfigDocument, error) {
	doc := &sequenceconfig.SequenceConfigDocument{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		Configs:        make([]*sequenceconfig.SequenceConfig, 0, len(input.Configs)),
	}

	for i, item := range input.Configs {
		if item == nil {
			continue
		}
		id, err := gqlid.OptionalScopedID(fmt.Sprintf("configs[%d].id", i), item.ID)
		if err != nil {
			return nil, err
		}

		cfg := &sequenceconfig.SequenceConfig{
			ID:                      id,
			OrganizationID:          authCtx.OrganizationID,
			BusinessUnitID:          authCtx.BusinessUnitID,
			SequenceType:            item.SequenceType,
			Prefix:                  item.Prefix,
			IncludeYear:             item.IncludeYear,
			YearDigits:              intutils.SafeToInt16(item.YearDigits),
			IncludeMonth:            item.IncludeMonth,
			IncludeWeekNumber:       item.IncludeWeekNumber,
			IncludeDay:              item.IncludeDay,
			SequenceDigits:          intutils.SafeToInt16(item.SequenceDigits),
			IncludeLocationCode:     item.IncludeLocationCode,
			IncludeRandomDigits:     item.IncludeRandomDigits,
			RandomDigitsCount:       intutils.SafeToInt16(item.RandomDigitsCount),
			IncludeCheckDigit:       item.IncludeCheckDigit,
			IncludeBusinessUnitCode: item.IncludeBusinessUnitCode,
			UseSeparators:           item.UseSeparators,
			SeparatorChar:           item.SeparatorChar,
			AllowCustomFormat:       item.AllowCustomFormat,
			CustomFormat:            item.CustomFormat,
			LocationCodeStrategy:    locationCodeStrategyFromInput(item.LocationCodeStrategy),
		}
		if item.Version != nil {
			cfg.Version = int64(*item.Version)
		}
		doc.Configs = append(doc.Configs, cfg)
	}

	return doc, nil
}

func locationCodeStrategyFromInput(
	input *gqlmodel.LocationCodeStrategyInput,
) *sequenceconfig.LocationCodeStrategy {
	if input == nil {
		return nil
	}

	return &sequenceconfig.LocationCodeStrategy{
		Components:     append([]sequenceconfig.LocationCodeComponent(nil), input.Components...),
		ComponentWidth: intutils.SafeToInt16(input.ComponentWidth),
		SequenceDigits: intutils.SafeToInt16(input.SequenceDigits),
		Separator:      input.Separator,
		Casing:         input.Casing,
		FallbackPrefix: input.FallbackPrefix,
	}
}
