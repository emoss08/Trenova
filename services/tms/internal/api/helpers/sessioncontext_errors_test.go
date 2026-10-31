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

func TestProblemBuilder_SessionContextStale_CarriesTheCurrentSession(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf(
		"guard: %w",
		errortypes.NewSessionContextStaleError("usr_current", "org_current", "bu_current"),
	)
	sanitizer := helpers.NewSanitizer(false)
	problemType := helpers.NewDefaultClassifier().Classify(err)

	problem := helpers.NewProblemBuilder("https://api.test/problems/").
		WithLocale(i18n.EN).
		WithType(problemType).
		WithDetail(sanitizer.SanitizeMessage(err, problemType)).
		WithParams(sanitizer.ExtractParams(err)).
		Build()

	assert.Equal(t, "https://api.test/problems/session-context-stale", problem.Type)
	assert.Equal(t, http.StatusConflict, problem.Status)
	assert.Equal(
		t,
		"Your session changed in another tab. Reload this page to continue.",
		problem.Detail,
	)
	assert.Equal(t, map[string]string{
		"userId":         "usr_current",
		"organizationId": "org_current",
		"businessUnitId": "bu_current",
	}, problem.Params)
}
