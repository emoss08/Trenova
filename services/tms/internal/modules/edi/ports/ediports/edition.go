package ediports

type EditionInfo interface {
	Name() string
	SharedTenancy() bool
}
