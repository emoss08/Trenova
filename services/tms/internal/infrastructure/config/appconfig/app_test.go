package appconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppConfig_IsDevelopment(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvDevelopment}
		assert.True(t, c.IsDevelopment())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvProduction}
		assert.False(t, c.IsDevelopment())
	})
}

func TestAppConfig_IsProduction(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvProduction}
		assert.True(t, c.IsProduction())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvDevelopment}
		assert.False(t, c.IsProduction())
	})
}

func TestAppConfig_IsStaging(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvStaging}
		assert.True(t, c.IsStaging())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvDevelopment}
		assert.False(t, c.IsStaging())
	})
}

func TestAppConfig_IsTest(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvTest}
		assert.True(t, c.IsTest())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &Config{Env: EnvDevelopment}
		assert.False(t, c.IsTest())
	})
}

func TestAppConfig_GetProblemTypeBaseURI(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, "https://api.trenova.app/problems/", c.GetProblemTypeBaseURI())
	})

	t.Run("custom with trailing slash", func(t *testing.T) {
		t.Parallel()
		c := &Config{ProblemTypeBaseURI: "https://example.com/errors/"}
		assert.Equal(t, "https://example.com/errors/", c.GetProblemTypeBaseURI())
	})

	t.Run("custom without trailing slash", func(t *testing.T) {
		t.Parallel()
		c := &Config{ProblemTypeBaseURI: "https://example.com/errors"}
		assert.Equal(t, "https://example.com/errors/", c.GetProblemTypeBaseURI())
	})
}

func TestAppConfig_GetWebBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		expected string
	}{
		{name: "unset", base: "", expected: ""},
		{
			name:     "without trailing slash",
			base:     "https://app.example.com",
			expected: "https://app.example.com",
		},
		{
			name:     "with trailing slash",
			base:     "https://app.example.com/",
			expected: "https://app.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Config{WebBaseURL: tt.base}
			assert.Equal(t, tt.expected, c.GetWebBaseURL())
		})
	}
}
