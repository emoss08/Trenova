package postgres

import "github.com/emoss08/keel"

var (
	rlsOrgID     = keel.DeclareFunc0[keel.StringExpr]("trenova_rls.org_id")
	rlsReconcile = keel.DeclareFunc0[keel.Int32Expr]("trenova_rls.reconcile")
)

func RLSFuncs() []keel.DeclaredFunc {
	return []keel.DeclaredFunc{rlsOrgID, rlsReconcile}
}
