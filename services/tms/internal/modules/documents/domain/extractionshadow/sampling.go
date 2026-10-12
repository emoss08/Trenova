package extractionshadow

import (
	"strconv"

	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func Sampled(documentID pulid.ID, extractedAt int64, percent int) bool {
	return hashutils.InPercentSample(
		documentID.String()+":"+strconv.FormatInt(extractedAt, 10),
		percent,
	)
}
