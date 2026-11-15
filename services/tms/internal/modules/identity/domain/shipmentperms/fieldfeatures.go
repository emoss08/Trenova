package shipmentperms

import "github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"

// shipmentFieldFeatures are the values of other features a shipment holds,
// by API name: the tariff record it was rated by (its rating method, inputs
// and rating detail are base, since every plan prices with the standard
// templates), its tender, its projected arrival and its detention charges
// and totals. A field of a plain struct the shipment carries (an additional
// charge, its rating detail) is named by its own name.
func shipmentFieldFeatures() map[string]platformcatalog.FeatureKey {
	return map[string]platformcatalog.FeatureKey{
		"formulaTemplate": platformcatalog.FeatureRating,
		"tenderStatus":    platformcatalog.FeatureCarrierBrokerage,
		"eta":             platformcatalog.FeatureCustomerTracking,
		"isDetention":     platformcatalog.FeatureDetention,
		"detention":       platformcatalog.FeatureDetention,
	}
}
