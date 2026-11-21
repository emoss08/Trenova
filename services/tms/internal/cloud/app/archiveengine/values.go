package archiveengine

import (
	"encoding/binary"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/copybinary"
	"github.com/google/uuid"
)

const jsonbVersion = byte(1)

var errKeyType = errors.New("a primary key column of a type the archive cannot compare")

func baseType(typ string) string {
	if i := strings.IndexByte(typ, '('); i >= 0 {
		return typ[:i]
	}
	return typ
}

func textOf(row *copybinary.Row, i int, typ string) (string, bool) {
	if row.Null(i) {
		return "", false
	}
	raw := row.Bytes(i)
	if typ == "jsonb" {
		if len(raw) == 0 || raw[0] != jsonbVersion {
			return "", false
		}
		raw = raw[1:]
	}
	return string(raw), true
}

func setText(row *copybinary.Row, i int, typ, value string) {
	if typ == "jsonb" {
		row.Set(i, append([]byte{jsonbVersion}, value...))
		return
	}
	row.Set(i, []byte(value))
}

func intOf(row *copybinary.Row, i int, typ string) (int64, bool) {
	if row.Null(i) {
		return 0, false
	}
	raw := row.Bytes(i)
	switch {
	case typ == "bigint" && len(raw) == 8:
		return int64(binary.BigEndian.Uint64(raw)), true //nolint:gosec // the wire format is signed
	case typ == "integer" && len(raw) == 4:
		value := int32(binary.BigEndian.Uint32(raw)) //nolint:gosec // the wire format is signed
		return int64(value), true
	case typ == "smallint" && len(raw) == 2:
		value := int16(binary.BigEndian.Uint16(raw)) //nolint:gosec // the wire format is signed
		return int64(value), true
	default:
		return 0, false
	}
}

func keyText(row *copybinary.Row, i int, typ string) (string, error) {
	base := baseType(typ)
	switch base {
	case "character varying", "text", "character":
		value, _ := textOf(row, i, base)
		return value, nil
	case "bigint", "integer", "smallint":
		value, _ := intOf(row, i, base)
		return strconv.FormatInt(value, 10), nil
	case "uuid":
		parsed, err := uuid.FromBytes(row.Bytes(i))
		if err != nil {
			return "", err
		}
		return parsed.String(), nil
	default:
		return "", errKeyType
	}
}

func encodeNatural(values []string) string {
	escaped := make([]string, 0, len(values))
	for _, v := range values {
		escaped = append(escaped, url.PathEscape(v))
	}
	return strings.Join(escaped, "/")
}

func decodeNatural(encoded string, parts int) ([]string, bool) {
	split := strings.Split(encoded, "/")
	if len(split) != parts {
		return nil, false
	}
	out := make([]string, 0, parts)
	for _, s := range split {
		v, err := url.PathUnescape(s)
		if err != nil {
			return nil, false
		}
		out = append(out, v)
	}
	return out, true
}

func joinKey(values []string) string {
	return strings.Join(values, "\x1f")
}
