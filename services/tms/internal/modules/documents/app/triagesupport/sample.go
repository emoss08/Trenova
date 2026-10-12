package triagesupport

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/shared/referenceutils"
)

func SampleOf(row *triageports.MessageRow) *inboundmessageports.Sample {
	sample := row.Sample()
	if sample.Reference == "" {
		sample.Reference = ReferenceIn(row.Subject)
	}

	return sample
}

func ReferenceIn(subject string) string {
	if candidates := referenceutils.Candidates(1, subject); len(candidates) > 0 {
		return candidates[0]
	}

	return ""
}

func SamplesOf(rows []*triageports.MessageRow) []*inboundmessageports.Sample {
	samples := make([]*inboundmessageports.Sample, 0, len(rows))
	for _, row := range rows {
		samples = append(samples, SampleOf(row))
	}

	return samples
}
