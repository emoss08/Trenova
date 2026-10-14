package formulatypes

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

type LanguageVersion int16

const (
	LanguageVersion1       LanguageVersion = 1
	CurrentLanguageVersion                 = LanguageVersion1
)

func (v LanguageVersion) OrCurrent() LanguageVersion {
	if v == 0 {
		return CurrentLanguageVersion
	}
	return v
}

func (v LanguageVersion) IsSupported() bool {
	return v >= LanguageVersion1 && v <= CurrentLanguageVersion
}

func ValidateLanguageVersion(
	field string,
	version LanguageVersion,
	multiErr *errortypes.MultiError,
) {
	if version == 0 || version.IsSupported() {
		return
	}
	multiErr.Add(
		field,
		errortypes.ErrInvalid,
		"Formula language version {0} is not supported; this server reads version {1}",
		int(version),
		int(CurrentLanguageVersion),
	)
}
