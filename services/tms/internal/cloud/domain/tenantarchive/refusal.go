package tenantarchive

import (
	"errors"
	"unicode/utf8"
)

type RefusalCode string

const (
	RefusalManifest  = RefusalCode("manifest")
	RefusalChecksum  = RefusalCode("checksum")
	RefusalKey       = RefusalCode("key")
	RefusalSchema    = RefusalCode("schema")
	RefusalPurpose   = RefusalCode("purpose")
	RefusalInstance  = RefusalCode("instance")
	RefusalExists    = RefusalCode("exists")
	RefusalReference = RefusalCode("reference")
	RefusalTaken     = RefusalCode("taken")
	RefusalTenant    = RefusalCode("tenant")
	RefusalLocation  = RefusalCode("location")
	RefusalStorage   = RefusalCode("storage")
	RefusalKeyAbsent = RefusalCode("key_unknown")
	RefusalFailed    = RefusalCode("failed")

	maxRefusalReason = 500
)

type RefusalError struct {
	Code   RefusalCode
	Reason string
}

func (r *RefusalError) Error() string {
	return r.Reason
}

func Refuse(code RefusalCode, reason string) *RefusalError {
	if utf8.RuneCountInString(reason) > maxRefusalReason {
		runes := []rune(reason)
		reason = string(runes[:maxRefusalReason-1]) + "…"
	}
	return &RefusalError{Code: code, Reason: reason}
}

func RefusalOf(err error) (*RefusalError, bool) {
	var r *RefusalError
	if errors.As(err, &r) {
		return r, true
	}
	return nil, false
}
