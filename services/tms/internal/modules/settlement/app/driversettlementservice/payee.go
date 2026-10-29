package driversettlementservice

import (
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
)

func classificationForWorker(profile *driverpay.PayProfile) driverpay.PayeeClassification {
	if profile != nil {
		return profile.Classification
	}
	return driverpay.PayeeClassificationCompanyDriver
}
