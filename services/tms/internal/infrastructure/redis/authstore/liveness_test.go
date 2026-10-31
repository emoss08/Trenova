package authstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionClockFrom_ReadsEitherShapeRedisJSONAnswersIn(t *testing.T) {
	t.Parallel()

	want := sessionClock{createdAt: 1700000000, lastAccessedAt: 1700000500, expiresAt: 1700043200}
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "legacy paths",
			raw:  `{".createdAt":1700000000,".lastAccessedAt":1700000500,".expiresAt":1700043200}`,
		},
		{
			name: "JSONPath paths",
			raw:  `{"$.createdAt":[1700000000],"$.lastAccessedAt":[1700000500],"$.expiresAt":[1700043200]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clock, ok := sessionClockFrom(tt.raw)

			assert.True(t, ok)
			assert.Equal(t, want, clock)
		})
	}
}

func TestSessionClockFrom_RefusesAReplyItCannotTrust(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{name: "not JSON", raw: `not json`},
		{
			name: "a whole session",
			raw:  `{"createdAt":1700000000,"lastAccessedAt":1700000500,"expiresAt":1700043200}`,
		},
		{name: "a missing field", raw: `{".createdAt":1700000000,".expiresAt":1700043200}`},
		{
			name: "a zero timestamp",
			raw:  `{".createdAt":0,".lastAccessedAt":1700000500,".expiresAt":1700043200}`,
		},
		{
			name: "a text timestamp",
			raw:  `{".createdAt":"1700000000",".lastAccessedAt":1700000500,".expiresAt":1700043200}`,
		},
		{
			name: "several matches",
			raw:  `{"$.createdAt":[1,2],"$.lastAccessedAt":[1700000500],"$.expiresAt":[1700043200]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, ok := sessionClockFrom(tt.raw)

			assert.False(t, ok)
		})
	}
}
