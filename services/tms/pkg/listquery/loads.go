package listquery

import (
	"github.com/emoss08/keel"
)

type Loads struct {
	list []keel.Loader
	err  error
}

func (l *Loads) Add(include bool, build func() (keel.Loader, error)) {
	if !include || l.err != nil {
		return
	}

	loader, err := build()
	if err != nil {
		l.err = err
		return
	}
	l.list = append(l.list, loader)
}

func (l *Loads) Result() ([]keel.Loader, error) {
	return l.list, l.err
}

func Load[T any](q *keel.Query[T], columns []string, nested ...keel.Loader) (keel.Loader, error) {
	picked, err := Pick(q, columns)
	if err != nil {
		return nil, err
	}
	if len(nested) == 0 {
		return picked, nil
	}

	return picked.Load(nested...), nil
}

func When(include bool, loaders ...keel.Loader) []keel.Loader {
	if !include {
		return nil
	}

	return loaders
}
