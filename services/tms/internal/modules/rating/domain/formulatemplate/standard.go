package formulatemplate

// Standard rating method template names, seeded for every organization.
//
// Every rate in Trenova is priced by a formula template. These are the
// templates that cover the ordinary shapes — a flat amount, a per-mile rate,
// a hundredweight tariff — so a pricing analyst picks one by name and types a
// number, and only writes an expression for a shape no standard template
// covers. The rate sheet importer resolves the method column against these
// names, which is why they are constants rather than strings scattered at the
// call sites.
const (
	StandardFlatRate          = "Flat Rate"
	StandardPerMile           = "Per Mile"
	StandardPerStop           = "Per Stop"
	StandardPerPound          = "Per Pound"
	StandardPerPallet         = "Per Pallet"
	StandardPerPiece          = "Per Piece"
	StandardPerLinearFoot     = "Per Linear Foot"
	StandardPerCwt            = "Per CWT (Hundredweight)"
	StandardPerHour           = "Per Hour"
	StandardPercentOfSellRate = "Percent of Sell Rate"
)
