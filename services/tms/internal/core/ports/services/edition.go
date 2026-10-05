package services

type EditionInfo interface {
	Name() string
	SharedTenancy() bool
}
