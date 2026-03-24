package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dothazmatreference"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type DotHazmatReferencesSeed struct {
	seedhelpers.BaseSeed
}

func NewDotHazmatReferencesSeed() *DotHazmatReferencesSeed {
	seed := &DotHazmatReferencesSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DotHazmatReferences",
		"1.0.0",
		"Creates DOT hazardous materials reference data from 49 CFR 172.101",
		[]common.Environment{
			common.EnvProduction,
			common.EnvStaging,
			common.EnvDevelopment,
			common.EnvTest,
		},
	)
	return seed
}

func (s *DotHazmatReferencesSeed) Run(ctx context.Context, tx bun.Tx) error {
	var count int
	err := tx.NewSelect().
		Model((*dothazmatreference.DotHazmatReference)(nil)).
		ColumnExpr("count(*)").
		Scan(ctx, &count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	loader := seedhelpers.NewDataLoader("./internal/infrastructure/database/seeds/base/data")

	var data struct {
		References []struct {
			UnNumber            string `json:"un_number"`
			ProperShippingName  string `json:"proper_shipping_name"`
			HazardClass         string `json:"hazard_class"`
			SubsidiaryHazard    string `json:"subsidiary_hazard"`
			PackingGroup        string `json:"packing_group"`
			SpecialProvisions   string `json:"special_provisions"`
			PackagingExceptions string `json:"packaging_exceptions"`
			PackagingNonBulk    string `json:"packaging_non_bulk"`
			PackagingBulk       string `json:"packaging_bulk"`
			QuantityPassenger   string `json:"quantity_passenger"`
			QuantityCargo       string `json:"quantity_cargo"`
			VesselStowage       string `json:"vessel_stowage"`
			ErgGuide            string `json:"erg_guide"`
			Symbols             string `json:"symbols"`
		} `json:"references"`
	}

	if err := loader.LoadYAML("dot_hazmat_references.yaml", &data); err != nil {
		return err
	}

	const batchSize = 500
	refs := make([]dothazmatreference.DotHazmatReference, len(data.References))
	for i, ref := range data.References {
		refs[i] = dothazmatreference.DotHazmatReference{
			UnNumber:            ref.UnNumber,
			ProperShippingName:  ref.ProperShippingName,
			HazardClass:         ref.HazardClass,
			SubsidiaryHazard:    ref.SubsidiaryHazard,
			PackingGroup:        ref.PackingGroup,
			SpecialProvisions:   ref.SpecialProvisions,
			PackagingExceptions: ref.PackagingExceptions,
			PackagingNonBulk:    ref.PackagingNonBulk,
			PackagingBulk:       ref.PackagingBulk,
			QuantityPassenger:   ref.QuantityPassenger,
			QuantityCargo:       ref.QuantityCargo,
			VesselStowage:       ref.VesselStowage,
			ErgGuide:            ref.ErgGuide,
			Symbols:             ref.Symbols,
		}
	}

	for i := 0; i < len(refs); i += batchSize {
		end := min(i+batchSize, len(refs))

		batch := refs[i:end]
		if _, err := tx.NewInsert().Model(&batch).Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}
