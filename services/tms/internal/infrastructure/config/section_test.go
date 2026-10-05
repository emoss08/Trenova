package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errFakeSectionInvalid = errors.New("fake section refused")

type fakeSettings struct {
	Platform struct {
		Example struct {
			Enabled bool          `mapstructure:"enabled"`
			Timeout time.Duration `mapstructure:"timeout"`
		} `mapstructure:"example"`
	} `mapstructure:"platform"`
}

type fakeSection struct {
	name      string
	paths     []string
	modes     []PlatformMode
	validated *bool
	refuse    bool
}

func (s fakeSection) Name() string                  { return s.name }
func (s fakeSection) Paths() []string               { return s.paths }
func (s fakeSection) PlatformModes() []PlatformMode { return s.modes }

func (s fakeSection) SetDefaults(v *viper.Viper, envPrefix string) {
	v.SetDefault("platform.example.timeout", "5s")
	_ = v.BindEnv("platform.example.enabled", envPrefix+"_EXAMPLE_ENABLED")
}

func (s fakeSection) Decode(v *viper.Viper) (any, error) {
	return DecodeSection[fakeSettings](v)
}

func (s fakeSection) Validate(cfg *Config, _ string) error {
	if s.validated != nil {
		*s.validated = true
	}
	if s.refuse {
		return errFakeSectionInvalid
	}
	if _, ok := cfg.Extension(s.name); !ok {
		return errors.New("section value missing")
	}

	return nil
}

func newFakeSection() fakeSection {
	return fakeSection{
		name:  "example",
		paths: []string{"platform.example"},
		modes: []PlatformMode{PlatformModeCloud},
	}
}

func loadWithSections(t *testing.T, contents string, sections ...Section) (*Config, error) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(contents), 0o600))

	return NewLoader(
		WithConfigPath(dir),
		WithEnvironment(EnvTest),
		WithSections(sections...),
	).Load()
}

func TestLoad_EditionSectionWithoutTheEditionIsRefused(t *testing.T) {
	for _, contents := range []string{
		"platform:\n  cloud:\n    signup:\n      enabled: true\n",
		"platform:\n  controlPlane:\n    enabled: true\n",
		"aiRetraining:\n  enabled: true\n",
	} {
		_, err := loadWithSections(t, validConfigYAML()+contents)
		require.ErrorIs(t, err, ErrSectionRequiresEdition)
	}
}

func TestLoad_EditionPlatformModeWithoutTheEditionIsRefused(t *testing.T) {
	_, err := loadWithSections(t, validConfigYAML()+"platform:\n  mode: cloud\n")
	require.ErrorIs(t, err, ErrPlatformModeRequiresEdition)

	cfg, err := loadWithSections(
		t,
		validConfigYAML()+"platform:\n  mode: cloud\n",
		newFakeSection(),
	)
	require.NoError(t, err)
	assert.True(t, cfg.Platform.IsCloud())
}

func TestLoad_SectionDecodesItsOwnPathsStrictly(t *testing.T) {
	t.Setenv("TRENOVA_EXAMPLE_ENABLED", "true")

	validated := false
	section := newFakeSection()
	section.validated = &validated

	cfg, err := loadWithSections(t, validConfigYAML()+`
platform:
  instanceId: inst_01
  example:
    timeout: 30s
`, section)
	require.NoError(t, err)
	assert.True(t, validated)
	assert.Equal(t, "inst_01", cfg.Platform.InstanceID)

	value, ok := cfg.Extension("example")
	require.True(t, ok)
	settings, ok := value.(*fakeSettings)
	require.True(t, ok)
	assert.True(t, settings.Platform.Example.Enabled)
	assert.Equal(t, 30*time.Second, settings.Platform.Example.Timeout)

	_, err = loadWithSections(t, validConfigYAML()+`
platform:
  example:
    unknown: true
`, newFakeSection())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")

	_, err = loadWithSections(t, validConfigYAML()+`
platform:
  example:
    enabled: true
`)
	require.Error(t, err, "without the section the base decode refuses its keys")
}

func TestLoad_SectionValidationRuns(t *testing.T) {
	section := newFakeSection()
	section.refuse = true

	_, err := loadWithSections(t, validConfigYAML(), section)
	require.ErrorIs(t, err, errFakeSectionInvalid)
}

func TestLoad_RefusesInvalidSections(t *testing.T) {
	tests := []struct {
		name     string
		sections []Section
		want     error
	}{
		{
			name:     "no name",
			sections: []Section{fakeSection{paths: []string{"platform.example"}}},
			want:     ErrSectionNameRequired,
		},
		{
			name:     "no paths",
			sections: []Section{fakeSection{name: "example"}},
			want:     ErrSectionPathsRequired,
		},
		{
			name: "owns a base path",
			sections: []Section{
				fakeSection{name: "example", paths: []string{"platform.mode"}},
			},
			want: ErrSectionPathConflict,
		},
		{
			name: "owns a whole base section",
			sections: []Section{
				fakeSection{name: "example", paths: []string{"database"}},
			},
			want: ErrSectionPathConflict,
		},
		{
			name: "duplicate name",
			sections: []Section{
				newFakeSection(),
				fakeSection{name: "EXAMPLE", paths: []string{"platform.other"}},
			},
			want: ErrSectionAlreadyRegistered,
		},
		{
			name: "overlapping paths",
			sections: []Section{
				newFakeSection(),
				fakeSection{name: "other", paths: []string{"platform.example.nested"}},
			},
			want: ErrSectionPathConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadWithSections(t, validConfigYAML(), tt.sections...)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestConfigExtensions(t *testing.T) {
	t.Parallel()

	var empty *Config
	_, ok := empty.Extension("example")
	assert.False(t, ok)

	cfg := &Config{}
	_, ok = cfg.Extension("example")
	assert.False(t, ok)

	cfg.SetExtension("example", 42)
	value, ok := cfg.Extension("example")
	require.True(t, ok)
	assert.Equal(t, 42, value)
}
