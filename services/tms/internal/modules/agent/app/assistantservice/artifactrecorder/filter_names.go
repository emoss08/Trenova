package artifactrecorder

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/filterwords"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"go.uber.org/zap"
)

// nameFilters writes a table's filters with the names of the records they
// narrow by. A read that fails leaves them counting their records.
func (r *Recorder) nameFilters(artifact *assistantartifact.Artifact) {
	filters, _ := artifact.Payload[artifactpayload.PayloadFilters].([]filterwords.Filter)
	if len(filters) == 0 || r.reads == nil {
		return
	}
	if err := filterwords.NameRecords(r.ctx, r.reads.Labels, r.tenant, filters); err != nil {
		r.logger.Warn("failed to name the records a table was narrowed by", zap.Error(err))
	}
}
