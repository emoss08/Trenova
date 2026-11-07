package pagination

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash"
	"strings"
	"sync/atomic"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxCursorLength = 4096

	cursorKeyLabel = "trenova list cursor key v1"
	cursorTagSize  = 16
)

var ErrCursorInvalid = errors.New("cursor is invalid")

type cursorKind string

const (
	cursorKindKeyset   cursorKind = "keyset"
	cursorKindOffset   cursorKind = "offset"
	cursorKindKey      cursorKind = "key"
	cursorKindPosition cursorKind = "position"
	cursorKindRecord   cursorKind = "record"
)

var cursorKey atomic.Pointer[[]byte]

func init() {
	key := make([]byte, sha256.Size)
	_, _ = rand.Read(key)
	cursorKey.Store(&key)
}

func InstallCursorKey(secret string) {
	key := hashutils.DeriveKey(secret, cursorKeyLabel)
	cursorKey.Store(&key)
}

type CursorBinding struct {
	OrgID pulid.ID
	BuID  pulid.ID
	Query string
}

func BindCursor(tenant TenantInfo, resource string, sort []CursorSortField) CursorBinding {
	var query strings.Builder
	query.WriteString(resource)
	for _, field := range sort {
		query.WriteByte(0)
		query.WriteString(field.Field)
		query.WriteByte(' ')
		query.WriteString(field.Direction)
	}

	return CursorBinding{OrgID: tenant.OrgID, BuID: tenant.BuID, Query: query.String()}
}

func SortShape(sorts []domaintypes.SortField) []CursorSortField {
	if len(sorts) == 0 {
		return nil
	}

	shape := make([]CursorSortField, 0, len(sorts))
	for _, sort := range sorts {
		shape = append(shape, CursorSortField{Field: sort.Field, Direction: string(sort.Direction)})
	}

	return shape
}

func SealCursor(binding CursorBinding, payload []byte) string {
	return sealCursor(cursorKindRecord, binding, payload)
}

func OpenCursor(binding CursorBinding, encoded string) ([]byte, error) {
	return openCursor(cursorKindRecord, binding, encoded)
}

func sealCursor(kind cursorKind, binding CursorBinding, payload []byte) string {
	sealed := make([]byte, 0, len(payload)+cursorTagSize)
	sealed = append(sealed, payload...)
	sealed = append(sealed, cursorTag(kind, binding, payload)...)

	return base64.RawURLEncoding.EncodeToString(sealed)
}

func openCursor(kind cursorKind, binding CursorBinding, encoded string) ([]byte, error) {
	raw, ok := cursorBytes(encoded)
	if !ok {
		return nil, ErrCursorInvalid
	}

	split := len(raw) - cursorTagSize
	payload, tag := raw[:split], raw[split:]
	if !hmac.Equal(tag, cursorTag(kind, binding, payload)) {
		return nil, ErrCursorInvalid
	}

	return payload, nil
}

func cursorBytes(encoded string) ([]byte, bool) {
	if encoded == "" || len(encoded) > MaxCursorLength {
		return nil, false
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) <= cursorTagSize {
		return nil, false
	}

	return raw, true
}

func cursorTag(kind cursorKind, binding CursorBinding, payload []byte) []byte {
	mac := hmac.New(sha256.New, *cursorKey.Load())
	writeCursorPart(mac, []byte(kind))
	writeCursorPart(mac, []byte(binding.OrgID))
	writeCursorPart(mac, []byte(binding.BuID))
	writeCursorPart(mac, []byte(binding.Query))
	writeCursorPart(mac, payload)

	return mac.Sum(nil)[:cursorTagSize]
}

func writeCursorPart(mac hash.Hash, part []byte) {
	_, _ = mac.Write(binary.AppendUvarint(nil, uint64(len(part))))
	_, _ = mac.Write(part)
}
