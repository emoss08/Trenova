package config

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/spf13/viper"
)

type Section interface {
	Name() string
	Paths() []string
	PlatformModes() []PlatformMode
	SetDefaults(v *viper.Viper, envPrefix string)
	Decode(v *viper.Viper) (any, error)
	Validate(cfg *Config, env string) error
}

var editionPlatformModes = []PlatformMode{PlatformModeCloud}

var editionSectionPaths = []string{
	"platform.cloud",
	"platform.controlPlane",
	"system.networkPulse",
	"aiRetraining",
}

func validateSections(sections []Section) error {
	for idx, section := range sections {
		if err := validateSection(section, sections[:idx]); err != nil {
			return err
		}
	}

	return nil
}

func validateSection(section Section, existing []Section) error {
	if section == nil || strings.TrimSpace(section.Name()) == "" {
		return ErrSectionNameRequired
	}

	paths := section.Paths()
	if len(paths) == 0 {
		return fmt.Errorf("%w: %q", ErrSectionPathsRequired, section.Name())
	}

	for _, other := range existing {
		if strings.EqualFold(other.Name(), section.Name()) {
			return fmt.Errorf("%w: %q", ErrSectionAlreadyRegistered, section.Name())
		}
	}

	for _, path := range paths {
		if basePathExists(path) {
			return fmt.Errorf("%w: %q", ErrSectionPathConflict, path)
		}
		for _, other := range existing {
			for _, owned := range other.Paths() {
				if pathsOverlap(path, owned) {
					return fmt.Errorf("%w: %q", ErrSectionPathConflict, path)
				}
			}
		}
	}

	return nil
}

func splitPath(path string) []string {
	return strings.Split(strings.ToLower(strings.TrimSpace(path)), ".")
}

func pathsOverlap(a, b string) bool {
	left, right := splitPath(a), splitPath(b)
	shortest := min(len(left), len(right))

	return slices.Equal(left[:shortest], right[:shortest])
}

func basePathExists(path string) bool {
	current := reflect.TypeFor[Config]()
	for _, segment := range splitPath(path) {
		if current.Kind() != reflect.Struct {
			return false
		}

		field, ok := fieldByMapstructureKey(current, segment)
		if !ok {
			return false
		}

		current = field.Type
		for current.Kind() == reflect.Pointer {
			current = current.Elem()
		}
	}

	return true
}

func fieldByMapstructureKey(structType reflect.Type, key string) (reflect.StructField, bool) {
	for idx := range structType.NumField() {
		field := structType.Field(idx)
		if !field.IsExported() {
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("mapstructure"), ",")
		if name == "" {
			name = field.Name
		}
		if strings.EqualFold(name, key) {
			return field, true
		}
	}

	return reflect.StructField{}, false
}

func extractPath(settings map[string]any, path string) (any, bool) {
	segments := splitPath(path)
	current := settings
	for _, segment := range segments[:len(segments)-1] {
		next, ok := current[segment].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}

	last := segments[len(segments)-1]
	value, ok := current[last]
	if ok {
		delete(current, last)
	}

	return value, ok
}

func insertPath(settings map[string]any, path string, value any) {
	segments := splitPath(path)
	current := settings
	for _, segment := range segments[:len(segments)-1] {
		next, ok := current[segment].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[segment] = next
		}
		current = next
	}

	current[segments[len(segments)-1]] = value
}

func DecodeSection[T any](v *viper.Viper) (*T, error) {
	target := new(T)
	if err := v.UnmarshalExact(target); err != nil {
		return nil, fmt.Errorf("decode configuration section: %w", err)
	}

	return target, nil
}

func (c *Config) Extension(name string) (any, bool) {
	if c == nil || c.extensions == nil {
		return nil, false
	}

	value, ok := c.extensions[name]

	return value, ok
}

func (c *Config) SetExtension(name string, value any) {
	if c.extensions == nil {
		c.extensions = make(map[string]any)
	}

	c.extensions[name] = value
}

func IsProductionLike(env string) bool {
	return isProductionLike(env)
}
