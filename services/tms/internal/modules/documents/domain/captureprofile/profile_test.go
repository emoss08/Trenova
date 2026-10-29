package captureprofile_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func errorFields(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestProfileValidate(t *testing.T) {
	t.Parallel()

	valid := func() *captureprofile.CaptureProfile {
		profile := &captureprofile.CaptureProfile{Name: "PODs"}
		profile.ApplyDefaults()

		return profile
	}

	multiErr := errortypes.NewMultiError()
	valid().Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "defaults must be a valid profile")

	fixed := valid()
	fixed.SeparatorStrategies = []captureprofile.SeparatorStrategy{
		captureprofile.SeparatorFixedPageCount,
	}
	multiErr = errortypes.NewMultiError()
	fixed.Validate(multiErr)
	assert.Equal(t, []string{"fixedPageCount"}, errorFields(multiErr))

	stray := valid()
	stray.FixedPageCount = 3
	multiErr = errortypes.NewMultiError()
	stray.Validate(multiErr)
	assert.Equal(t, []string{"fixedPageCount"}, errorFields(multiErr))

	inactiveDefault := valid()
	inactiveDefault.IsDefault = true
	inactiveDefault.Status = captureprofile.ProfileInactive
	multiErr = errortypes.NewMultiError()
	inactiveDefault.Validate(multiErr)
	assert.Equal(t, []string{"isDefault"}, errorFields(multiErr))

	bad := valid()
	bad.DPI = 1200
	bad.JPEGQuality = 10
	bad.SeparatorStrategies = []captureprofile.SeparatorStrategy{"Staple"}
	multiErr = errortypes.NewMultiError()
	bad.Validate(multiErr)
	assert.ElementsMatch(
		t,
		[]string{"dpi", "jpegQuality", "separatorStrategies"},
		errorFields(multiErr),
	)
}
