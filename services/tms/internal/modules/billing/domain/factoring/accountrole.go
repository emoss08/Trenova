package factoring

type AccountRole string

const (
	AccountRoleAR            = AccountRole("AccountsReceivable")
	AccountRoleCash          = AccountRole("Cash")
	AccountRoleDueFromFactor = AccountRole("DueFromFactor")
	AccountRoleFeeExpense    = AccountRole("FeeExpense")
	AccountRoleLiability     = AccountRole("Liability")
)
