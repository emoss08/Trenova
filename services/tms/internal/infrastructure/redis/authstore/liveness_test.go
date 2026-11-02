package authstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionVitalsFrom_ReadsEitherShapeRedisJSONAnswersIn(t *testing.T) {
	t.Parallel()

	want := sessionVitals{
		createdAt:           1700000000,
		lastAccessedAt:      1700000500,
		expiresAt:           1700043200,
		secretHash:          "current",
		previousSecretHash:  "previous",
		previousSecretUntil: 1700000560,
	}
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "legacy paths",
			raw: `{".createdAt":1700000000,".lastAccessedAt":1700000500,".expiresAt":1700043200,` +
				`".secretHash":"current",".previousSecretHash":"previous",` +
				`".previousSecretUntil":1700000560}`,
		},
		{
			name: "JSONPath paths",
			raw: `{"$.createdAt":[1700000000],"$.lastAccessedAt":[1700000500],` +
				`"$.expiresAt":[1700043200],"$.secretHash":["current"],` +
				`"$.previousSecretHash":["previous"],"$.previousSecretUntil":[1700000560]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vitals, ok := sessionVitalsFrom(tt.raw)

			assert.True(t, ok)
			assert.Equal(t, want, vitals)
		})
	}
}

func TestSessionVitalsFrom_ReadsASessionThatWasNeverRotated(t *testing.T) {
	t.Parallel()

	vitals, ok := sessionVitalsFrom(
		`{".createdAt":1700000000,".lastAccessedAt":1700000500,".expiresAt":1700043200,` +
			`".secretHash":"current",".previousSecretHash":"",".previousSecretUntil":0}`,
	)

	assert.True(t, ok)
	assert.Empty(t, vitals.previousSecretHash)
	assert.Zero(t, vitals.previousSecretUntil)
}

func TestSessionVitalsFrom_RefusesAReplyItCannotTrust(t *testing.T) {
	t.Parallel()

	const secrets = `,".secretHash":"current",".previousSecretHash":"",".previousSecretUntil":0}`
	tests := []struct {
		name string
		raw  string
	}{
		{name: "not JSON", raw: `not json`},
		{
			name: "a whole session",
			raw:  `{"createdAt":1700000000,"lastAccessedAt":1700000500,"expiresAt":1700043200}`,
		},
		{
			name: "a missing field",
			raw:  `{".createdAt":1700000000,".expiresAt":1700043200` + secrets,
		},
		{
			name: "a zero timestamp",
			raw:  `{".createdAt":0,".lastAccessedAt":1700000500,".expiresAt":1700043200` + secrets,
		},
		{
			name: "a text timestamp",
			raw: `{".createdAt":"1700000000",".lastAccessedAt":1700000500,` +
				`".expiresAt":1700043200` + secrets,
		},
		{
			name: "several matches",
			raw: `{"$.createdAt":[1,2],"$.lastAccessedAt":[1700000500],"$.expiresAt":[1700043200],` +
				`"$.secretHash":["current"],"$.previousSecretHash":[""],"$.previousSecretUntil":[0]}`,
		},
		{
			name: "no secret",
			raw: `{".createdAt":1700000000,".lastAccessedAt":1700000500,".expiresAt":1700043200,` +
				`".secretHash":"",".previousSecretHash":"",".previousSecretUntil":0}`,
		},
		{
			name: "a numeric secret",
			raw: `{".createdAt":1700000000,".lastAccessedAt":1700000500,".expiresAt":1700043200,` +
				`".secretHash":7,".previousSecretHash":"",".previousSecretUntil":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, ok := sessionVitalsFrom(tt.raw)

			assert.False(t, ok)
		})
	}
}
