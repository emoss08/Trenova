package helpers_test

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
	"github.com/emoss08/trenova/shared/i18n"
	val "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pathAddress struct {
	City string `json:"city"`
}

func (a pathAddress) Validate() error {
	return val.ValidateStruct(&a, val.Field(&a.City, val.Required))
}

type pathStop struct {
	Name string `json:"name"`
}

func (s pathStop) Validate() error {
	return val.ValidateStruct(&s, val.Field(&s.Name, val.Required.Error("Stop name is required")))
}

type pathShipment struct {
	Code    string      `json:"code"`
	Tags    []string    `json:"tags"`
	Address pathAddress `json:"address"`
	Stops   []pathStop  `json:"stops"`
}

func TestFieldErrorsTravelAsStructuredPaths(t *testing.T) {
	t.Parallel()

	shipment := pathShipment{
		Code:  "TOO-LONG-CODE",
		Tags:  []string{"", "ok"},
		Stops: []pathStop{{Name: "Dallas"}, {}},
	}

	multiErr := errortypes.NewMultiError()
	multiErr.AddOzzoError(val.ValidateStruct(&shipment,
		val.Field(&shipment.Code, val.Length(1, 5)),
		val.Field(&shipment.Tags, val.Each(val.Required)),
		val.Field(&shipment.Address),
		val.Field(&shipment.Stops),
	))
	multiErr.WithIndex("moves", 3).Add("stops[1].status", errortypes.ErrInvalid, "Stop is not done")

	require.NoError(t, multiErr.Internal())
	require.Error(t, multiErr.Err())

	type expectation struct {
		path    fieldpath.Path
		code    errortypes.ErrorCode
		message string
		args    []any
	}
	want := []expectation{
		{
			path:    fieldpath.Of(fieldpath.Key("address"), fieldpath.Key("city")),
			code:    errortypes.ErrRequired,
			message: "This field is required",
		},
		{
			path:    fieldpath.Of(fieldpath.Key("code")),
			code:    errortypes.ErrInvalidLength,
			message: "Must be between {0} and {1} characters",
			args:    []any{1, 5},
		},
		{
			path: fieldpath.Of(
				fieldpath.Key("stops"),
				fieldpath.Index(1),
				fieldpath.Key("name"),
			),
			code:    errortypes.ErrRequired,
			message: "Stop name is required",
		},
		{
			path:    fieldpath.Of(fieldpath.Key("tags"), fieldpath.Index(0)),
			code:    errortypes.ErrRequired,
			message: "This field is required",
		},
		{
			path: fieldpath.Of(
				fieldpath.Key("moves"),
				fieldpath.Index(3),
				fieldpath.Key("stops"),
				fieldpath.Index(1),
				fieldpath.Key("status"),
			),
			code:    errortypes.ErrInvalid,
			message: "Stop is not done",
		},
	}
	require.Len(t, multiErr.Errors, len(want))
	for i, expected := range want {
		got := multiErr.Errors[i]
		assert.True(
			t,
			expected.path.Equal(got.Path),
			"error %d path: want %s, got %s",
			i,
			expected.path,
			got.Path,
		)
		assert.Equal(t, expected.code, got.Code, "error %d code", i)
		assert.Equal(t, expected.message, got.Message, "error %d message", i)
		assert.Equal(t, expected.args, got.Args, "error %d args", i)
	}
	assert.Equal(t, "stops[1].name", multiErr.Errors[2].Field())
	assert.True(t, multiErr.WithIndex("stops", 1).HasErrors())
	assert.False(t, multiErr.WithIndex("stops", 0).HasErrors())
	assert.True(t, multiErr.HasErrorAt("code"))

	sanitizer := helpers.NewSanitizer(false)
	problemType := helpers.NewDefaultClassifier().Classify(multiErr)
	require.Equal(t, helpers.ProblemTypeValidation, problemType)

	problem := helpers.NewProblemBuilder("https://api.trenova.app/problems/").
		WithLocale(i18n.Default).
		WithType(problemType).
		WithCode(helpers.ErrorCodeOf(multiErr, problemType)).
		WithErrors(sanitizer.ExtractErrors(multiErr)).
		Build()

	body, err := sonic.Marshal(problem)
	require.NoError(t, err)

	var wire struct {
		Code   string `json:"code"`
		Errors []struct {
			Path    []any  `json:"path"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	require.NoError(t, sonic.Unmarshal(body, &wire))
	assert.Equal(t, string(errortypes.ErrInvalid), wire.Code)
	require.Len(t, wire.Errors, len(want))
	assert.Equal(t, []any{"stops", float64(1), "name"}, wire.Errors[2].Path)
	assert.Equal(t, []any{"moves", float64(3), "stops", float64(1), "status"}, wire.Errors[4].Path)
	assert.Equal(t, "Must be between 1 and 5 characters", wire.Errors[1].Message)
	assert.Equal(t, string(errortypes.ErrInvalidLength), wire.Errors[1].Code)
	assert.NotContains(t, string(body), `"field"`)

	assert.True(t, fieldpath.Parse("stops.1.name").Equal(fieldpath.Parse("stops[1].name")))
}

func TestAValidationThatCouldNotRunIsNotAFieldRefusal(t *testing.T) {
	t.Parallel()

	shipment := pathShipment{}
	stray := ""

	multiErr := errortypes.NewMultiError()
	multiErr.AddOzzoError(val.ValidateStruct(&shipment, val.Field(&stray, val.Required)))

	require.Error(t, multiErr.Internal())
	assert.True(t, multiErr.HasErrors())
	require.Error(t, multiErr.Err())
	assert.False(t, errortypes.IsMultiError(multiErr))

	classifier := helpers.NewDefaultClassifier()
	assert.Equal(t, helpers.ProblemTypeInternal, classifier.Classify(multiErr.Err()))
	assert.Empty(t, helpers.NewSanitizer(false).ExtractErrors(multiErr.Err()))
	assert.Equal(
		t,
		"An unexpected error occurred. Please try again later.",
		helpers.NewSanitizer(false).
			SanitizeMessage(multiErr.Err(), helpers.ProblemTypeInternal).
			Localize(i18n.Default),
	)
}
