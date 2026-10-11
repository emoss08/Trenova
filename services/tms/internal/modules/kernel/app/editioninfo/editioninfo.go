package editioninfo

import (
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
)

const SelfHostedName = "community"

type Static struct {
	name   string
	shared bool
}

var _ ediports.EditionInfo = Static{}

func NewStatic(name string, sharedTenancy bool) Static {
	return Static{name: name, shared: sharedTenancy}
}

func SelfHosted() ediports.EditionInfo {
	return NewStatic(SelfHostedName, false)
}

func (s Static) Name() string {
	return s.name
}

func (s Static) SharedTenancy() bool {
	return s.shared
}
