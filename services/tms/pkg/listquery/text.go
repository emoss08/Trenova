package listquery

import "github.com/emoss08/keel"

func TextContainsFold(e keel.Expr, s string) keel.Pred {
	return keel.DynOf(e).Cast(keel.TypeString).ContainsFold(s)
}
