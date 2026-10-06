package editioninfo

import "github.com/emoss08/trenova/internal/core/ports/services"

const SelfHostedName = "community"

type Static struct {
	name   string
	shared bool
}

var _ services.EditionInfo = Static{}

func NewStatic(name string, sharedTenancy bool) Static {
	return Static{name: name, shared: sharedTenancy}
}

func SelfHosted() services.EditionInfo {
	return NewStatic(SelfHostedName, false)
}

func (s Static) Name() string {
	return s.name
}

func (s Static) SharedTenancy() bool {
	return s.shared
}
