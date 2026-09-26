package aicorrection

import (
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

var PredictedFieldKeys = []string{
	"loadNumber", "referenceNumber", "shipper", "consignee",
	"rate", "equipmentType", "commodity",
	"pickupDate", "deliveryDate", "pickupWindow", "deliveryWindow",
	"pickupNumber", "deliveryNumber",
	"appointmentNumber", "bol", "poNumber", "scac", "proNumber",
	"paymentTerms", "billTo",
	"carrierName", "carrierContact", "containerNumber",
	"trailerNumber", "tractorNumber",
	"fuelSurcharge", "serviceType",
}

var canonicalFieldKeys = buildCanonicalFieldKeys()

func buildCanonicalFieldKeys() map[string]string {
	keys := make(map[string]string, len(PredictedFieldKeys)+len(fieldSpecs)*3)
	register := func(key string) {
		folded := foldFieldKey(key)
		if _, exists := keys[folded]; !exists {
			keys[folded] = key
		}
	}
	for _, key := range PredictedFieldKeys {
		register(key)
	}
	for i := range fieldSpecs {
		register(fieldSpecs[i].key)
		for _, alias := range fieldSpecs[i].aliases {
			register(alias)
		}
	}

	return keys
}

func CanonicalFieldKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if canonical, ok := canonicalFieldKeys[foldFieldKey(trimmed)]; ok {
		return canonical
	}

	return trimmed
}

func foldFieldKey(key string) string {
	return strings.ToLower(stringutils.NormalizeIdentifier(key))
}
