package toolschema

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const schemaURL = "tool.json"

// Validate checks arguments against a tool's parameter schema and reports
// what does not fit as field-level errors, so a form can show each one
// beside its value. A schema that declares nothing accepts anything.
func Validate(schema, args map[string]any) error {
	if len(schema) == 0 {
		return nil
	}

	compiled, err := compile(schema)
	if err != nil {
		return err
	}

	normalized, err := normalize(args)
	if err != nil {
		return fmt.Errorf("normalize tool arguments: %w", err)
	}

	err = compiled.Validate(normalized)
	if err == nil {
		return nil
	}

	var invalid *jsonschema.ValidationError
	if !errors.As(err, &invalid) {
		return fmt.Errorf("validate tool arguments: %w", err)
	}

	multiErr := errortypes.NewMultiError()
	collect(invalid, multiErr, message.NewPrinter(language.English))
	if !multiErr.HasErrors() {
		multiErr.Add("params", errortypes.ErrInvalid, invalid.Error())
	}

	return multiErr
}

// compile builds the schema from its Go shape. The map is round-tripped
// through JSON first, since the validator reads JSON-decoded values and a
// schema written in Go holds []string where it expects []any.
func compile(schema map[string]any) (*jsonschema.Schema, error) {
	document, err := normalize(schema)
	if err != nil {
		return nil, fmt.Errorf("normalize tool schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err = compiler.AddResource(schemaURL, document); err != nil {
		return nil, fmt.Errorf("load tool schema: %w", err)
	}

	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("compile tool schema: %w", err)
	}

	return compiled, nil
}

func normalize(value map[string]any) (any, error) {
	if value == nil {
		value = map[string]any{}
	}
	encoded, err := sonic.ConfigStd.Marshal(value)
	if err != nil {
		return nil, err
	}

	var decoded any
	if err = sonic.ConfigStd.Unmarshal(encoded, &decoded); err != nil {
		return nil, err
	}

	return decoded, nil
}

// collect walks the validator's tree to its leaves and files each one under
// the field it is about. A missing required value is filed under the value
// that is missing, not under the object that lacks it, which is where a
// form has a place to say so.
func collect(invalid *jsonschema.ValidationError, multiErr *errortypes.MultiError, printer *message.Printer) {
	if len(invalid.Causes) > 0 {
		for _, cause := range invalid.Causes {
			collect(cause, multiErr, printer)
		}

		return
	}

	path := fieldPath(invalid.InstanceLocation)
	switch failure := invalid.ErrorKind.(type) {
	case *kind.Required:
		for _, missing := range failure.Missing {
			multiErr.Add(join(path, missing), errortypes.ErrRequired, "This value is required")
		}
	case *kind.AdditionalProperties:
		for _, property := range failure.Properties {
			multiErr.Add(join(path, property), errortypes.ErrInvalid, "This tool does not take this value")
		}
	case *kind.Schema:
		// The root saying "does not validate" restates its causes.
	default:
		if path == "" {
			path = "params"
		}
		multiErr.Add(path, errortypes.ErrInvalid, failure.LocalizedString(printer))
	}
}

// fieldPath writes an instance location the way the client addresses form
// values: names joined by dots, indices in brackets.
func fieldPath(location []string) string {
	var path strings.Builder
	for _, segment := range location {
		if _, err := strconv.Atoi(segment); err == nil {
			path.WriteString("[" + segment + "]")

			continue
		}
		if path.Len() > 0 {
			path.WriteByte('.')
		}
		path.WriteString(segment)
	}

	return path.String()
}

func join(path, name string) string {
	if path == "" {
		return name
	}

	return path + "." + name
}
