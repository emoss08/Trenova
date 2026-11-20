package billingcontrol

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

var queueReviewFeatures = []platformcatalog.FeatureKey{
	platformcatalog.FeatureAIDesk,
	platformcatalog.FeatureAgentAutomation,
}

func QueueReviewFeatures() []platformcatalog.FeatureKey {
	return slices.Clone(queueReviewFeatures)
}

func (m QueueAutoApproveMode) OnPlan(queueReviewOnPlan bool) QueueAutoApproveMode {
	if m == QueueAutoApproveModeWhenAgentAgrees && !queueReviewOnPlan {
		return QueueAutoApproveModeOff
	}
	return m
}

func (bc *BillingControl) ApplyQueueReviewPlan(queueReviewOnPlan bool) {
	bc.QueueAutoApproveMode = bc.QueueAutoApproveMode.OnPlan(queueReviewOnPlan)
}
