package helpers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
)

func TestDefaultClassifier_PlanErrors(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	quota := fmt.Errorf("create: %w",
		errortypes.NewQuotaExceededError("customers.total", 8, 8, "free_demo"))
	assert.Equal(t, helpers.ProblemTypeQuotaExceeded, classifier.Classify(quota))

	restricted := fmt.Errorf("send: %w",
		errortypes.NewPlanRestrictionError("email.outbound", "", "free_demo"))
	assert.Equal(t, helpers.ProblemTypePlanRestricted, classifier.Classify(restricted))
}

func TestProblemTypeInfo_PlanErrors(t *testing.T) {
	t.Parallel()

	quota := helpers.ProblemTypeQuotaExceeded.Info()
	assert.Equal(t, http.StatusPaymentRequired, quota.StatusCode)
	assert.Equal(t, "Plan Limit Reached", quota.Title)
	assert.False(t, quota.ShouldLog)
	assert.False(t, helpers.ProblemTypeQuotaExceeded.IsInternal())

	restricted := helpers.ProblemTypePlanRestricted.Info()
	assert.Equal(t, http.StatusForbidden, restricted.StatusCode)
	assert.Equal(t, "Not Available on Your Plan", restricted.Title)
	assert.False(t, restricted.ShouldLog)
	assert.False(t, helpers.ProblemTypePlanRestricted.IsInternal())
}

func TestSanitizer_ExtractParams_PlanErrors(t *testing.T) {
	t.Parallel()

	sanitizer := helpers.NewSanitizer(false)

	assert.Equal(t, map[string]string{
		"meter": "documents.uploads",
		"limit": "25",
		"used":  "25",
		"plan":  "free_demo",
	}, sanitizer.ExtractParams(errortypes.NewQuotaExceededError("documents.uploads", 25, 25, "free_demo")))

	assert.Equal(t, map[string]string{
		"capability": "",
		"reason":     "subscription_read_only",
		"plan":       "free_demo",
	}, sanitizer.ExtractParams(errortypes.NewPlanRestrictionError(
		"",
		errortypes.PlanRestrictionReasonReadOnly,
		"free_demo",
	)))
}

func TestProblemBuilder_QuotaExceeded(t *testing.T) {
	t.Parallel()

	err := errortypes.NewQuotaExceededError("shipments.total", 12, 12, "free_demo")
	sanitizer := helpers.NewSanitizer(false)

	problem := helpers.NewProblemBuilder("https://api.test/problems/").
		WithLocale(i18n.EN).
		WithType(helpers.ProblemTypeQuotaExceeded).
		WithDetail(sanitizer.SanitizeMessage(err, helpers.ProblemTypeQuotaExceeded)).
		WithParams(sanitizer.ExtractParams(err)).
		Build()

	assert.Equal(t, "https://api.test/problems/quota-exceeded", problem.Type)
	assert.Equal(t, http.StatusPaymentRequired, problem.Status)
	assert.Equal(t, "This organization has reached the limit of its plan", problem.Detail)
	assert.Equal(t, "12", problem.Params["limit"])
	assert.Equal(t, "shipments.total", problem.Params["meter"])
}
