package listquery

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/stringutils"
)

func TextContainsFold(e keel.Expr, s string) keel.Pred {
	return keel.Raw("?::text ILIKE ?", e, "%"+stringutils.EscapeLikePattern(s)+"%")
}
