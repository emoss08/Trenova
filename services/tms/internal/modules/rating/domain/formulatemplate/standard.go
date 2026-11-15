package formulatemplate

import "github.com/emoss08/trenova/pkg/errortypes"

func (ft *FormulaTemplate) IsStandard() bool {
	return ft != nil && ft.StandardKey != ""
}

func (ft *FormulaTemplate) RefuseStandardChange() error {
	if !ft.IsStandard() {
		return nil
	}

	return errortypes.NewBusinessError(
		"{0} is a standard template, so it cannot be changed. Duplicate or fork it to make a template of your own",
		ft.Name,
	)
}

const (
	StandardFlatRate          = "Flat Rate"
	StandardPerMile           = "Per Mile"
	StandardPerMileOrFlat     = "Per Mile or Flat"
	StandardPerStop           = "Per Stop"
	StandardPerPound          = "Per Pound"
	StandardPerPallet         = "Per Pallet"
	StandardPerPiece          = "Per Piece"
	StandardPerLinearFoot     = "Per Linear Foot"
	StandardPerCwt            = "Per CWT (Hundredweight)"
	StandardPerHour           = "Per Hour"
	StandardPercentOfSellRate = "Percent of Sell Rate"
	StandardWeightBreaks      = "Weight Breaks (LTL)"
)

const (
	StandardInputRate          = "rate"
	StandardInputMinimumCharge = "minimumCharge"
	StandardInputBreaks        = "breaks"
)
