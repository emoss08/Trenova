package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveryURLConfigFieldIdentifiesTheTarget(t *testing.T) {
	t.Parallel()

	for typ, spec := range ConfigSpecs {
		for _, field := range spec.Fields {
			if field.Type == ConfigFieldTypeURL {
				assert.True(t, field.Target, "%s %s", typ, field.Key)
			}
		}
	}
}

func TestSFTPFuelFeedsMarkHostPortAndHostKeyAsTheTarget(t *testing.T) {
	t.Parallel()

	for _, typ := range []Type{TypeWEXFuel, TypeComdataFuel} {
		targets := make([]string, 0, 3)
		for _, field := range ConfigSpecs[typ].Fields {
			if field.Target {
				targets = append(targets, field.Key)
			}
		}
		assert.ElementsMatch(
			t,
			[]string{ConfigKeyFuelHost, ConfigKeyFuelPort, ConfigKeyFuelKnownHostKey},
			targets,
			string(typ),
		)
	}
}
