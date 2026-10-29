package completionengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestSurfaceFor(t *testing.T) {
	t.Parallel()

	live := aiusageports.AIUsageAttribution{RunID: pulid.MustNew("ar_")}
	byRun := aiusageports.AIUsageAttribution{RunID: pulid.MustNew("aeval_")}
	byPurpose := aiusageports.AIUsageAttribution{Purpose: aiusageports.AIUsagePurposeEvaluation}

	assert.Equal(t, aiusage.SurfaceChat, SurfaceFor(aiusage.SurfaceChat, &live))
	assert.Equal(t, aiusage.SurfaceStructured, SurfaceFor(aiusage.SurfaceStructured, &live))
	assert.Equal(t, aiusage.SurfaceEvaluation, SurfaceFor(aiusage.SurfaceChat, &byRun))
	assert.Equal(t, aiusage.SurfaceEvaluation, SurfaceFor(aiusage.SurfaceStructured, &byPurpose))
}
