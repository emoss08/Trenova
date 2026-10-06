package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestSectionsOptionSuppliesTheConfigSectionsGroup(t *testing.T) {
	t.Parallel()

	first := newFakeSection()
	second := fakeSection{name: "second", paths: []string{"platform.second"}}

	var got []Section
	app := fx.New(
		fx.NopLogger,
		SectionsOption(first, second),
		fx.Invoke(func(p ProvideConfigParams) {
			got = p.Sections
		}),
	)
	require.NoError(t, app.Err())

	names := make([]string, 0, len(got))
	for _, section := range got {
		names = append(names, section.Name())
	}
	assert.ElementsMatch(t, []string{"example", "second"}, names)
}

func TestSectionsOptionWithoutSectionsLeavesTheGroupEmpty(t *testing.T) {
	t.Parallel()

	var got []Section
	app := fx.New(
		fx.NopLogger,
		SectionsOption(),
		fx.Invoke(func(p ProvideConfigParams) {
			got = p.Sections
		}),
	)
	require.NoError(t, app.Err())
	assert.Empty(t, got)
}
