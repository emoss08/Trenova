package audit

type Category string

const (
	CategorySystem = Category("System")
	CategoryUser   = Category("User")
)
