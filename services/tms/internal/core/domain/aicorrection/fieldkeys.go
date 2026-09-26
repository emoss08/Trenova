package aicorrection

import (
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

type PredictedField struct {
	Key   string
	Label string
}

var predictedFields = []PredictedField{
	{Key: "loadNumber", Label: "Load Number"},
	{Key: "referenceNumber", Label: "Reference Number"},
	{Key: "shipper", Label: "Shipper"},
	{Key: "consignee", Label: "Consignee"},
	{Key: "rate", Label: "Rate"},
	{Key: "equipmentType", Label: "Equipment Type"},
	{Key: "commodity", Label: "Commodity"},
	{Key: "pickupDate", Label: "Pickup Date"},
	{Key: "deliveryDate", Label: "Delivery Date"},
	{Key: "pickupWindow", Label: "Pickup Window"},
	{Key: "deliveryWindow", Label: "Delivery Window"},
	{Key: "pickupNumber", Label: "Pickup Number"},
	{Key: "deliveryNumber", Label: "Delivery Number"},
	{Key: "appointmentNumber", Label: "Appointment Number"},
	{Key: "bol", Label: "BOL"},
	{Key: "poNumber", Label: "PO Number"},
	{Key: "scac", Label: "SCAC"},
	{Key: "proNumber", Label: "Pro Number"},
	{Key: "paymentTerms", Label: "Payment Terms"},
	{Key: "billTo", Label: "Bill To"},
	{Key: "carrierName", Label: "Carrier Name"},
	{Key: "carrierContact", Label: "Carrier Contact"},
	{Key: "containerNumber", Label: "Container Number"},
	{Key: "trailerNumber", Label: "Trailer Number"},
	{Key: "tractorNumber", Label: "Tractor Number"},
	{Key: "fuelSurcharge", Label: "Fuel Surcharge"},
	{Key: "serviceType", Label: "Service Type"},
}

var PredictedFieldKeys = buildPredictedFieldKeys()

var predictedFieldLabels = buildPredictedFieldLabels()

func buildPredictedFieldKeys() []string {
	keys := make([]string, 0, len(predictedFields))
	for _, field := range predictedFields {
		keys = append(keys, field.Key)
	}

	return keys
}

func buildPredictedFieldLabels() map[string]string {
	labels := make(map[string]string, len(predictedFields))
	for _, field := range predictedFields {
		labels[field.Key] = field.Label
	}

	return labels
}

func FieldLabel(key string) string {
	if label, ok := predictedFieldLabels[CanonicalFieldKey(key)]; ok {
		return label
	}

	return strings.TrimSpace(key)
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
