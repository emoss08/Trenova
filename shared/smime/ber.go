package smime

import (
	"encoding/asn1"
	"errors"
)

const (
	maxBERDepth       = 32
	maxEnvelopeFields = 8
)

var errMalformedEnvelope = errors.New("as2: malformed enveloped data")

type berElement struct {
	class       int
	tag         int
	constructed bool
	content     []byte
	depth       int
}

func (e berElement) is(class, tag int, constructed bool) bool {
	return e.class == class && e.tag == tag && e.constructed == constructed
}

func (e berElement) children(limit int) ([]berElement, error) {
	if !e.constructed {
		return nil, errMalformedEnvelope
	}
	children := make([]berElement, 0, limit)
	rest := e.content
	for len(rest) > 0 {
		if len(children) == limit {
			break
		}
		child, remaining, err := readBERElement(rest, e.depth+1)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
		rest = remaining
	}
	return children, nil
}

func (e berElement) objectIdentifier() (asn1.ObjectIdentifier, error) {
	if !e.is(asn1.ClassUniversal, asn1.TagOID, false) || len(e.content) == 0 ||
		len(e.content) > 127 {
		return nil, errMalformedEnvelope
	}
	encoded := make([]byte, 0, len(e.content)+2)
	length := byte(len(e.content)) //nolint:gosec // length is at most 127
	encoded = append(encoded, asn1.TagOID, length)
	encoded = append(encoded, e.content...)
	var oid asn1.ObjectIdentifier
	if _, err := asn1.Unmarshal(encoded, &oid); err != nil {
		return nil, errMalformedEnvelope
	}
	return oid, nil
}

func readBERElement(data []byte, depth int) (berElement, []byte, error) {
	if depth > maxBERDepth || len(data) < 2 {
		return berElement{}, nil, errMalformedEnvelope
	}
	identifier := data[0]
	element := berElement{
		class:       int(identifier >> 6),
		constructed: identifier&0x20 != 0,
		tag:         int(identifier & 0x1F),
		depth:       depth,
	}
	offset := 1
	if element.tag == 0x1F {
		element.tag = 0
		for {
			if offset >= len(data) || offset > 4 {
				return berElement{}, nil, errMalformedEnvelope
			}
			b := data[offset]
			offset++
			element.tag = element.tag<<7 | int(b&0x7F)
			if b&0x80 == 0 {
				break
			}
		}
	}
	if offset >= len(data) {
		return berElement{}, nil, errMalformedEnvelope
	}
	lengthByte := data[offset]
	offset++

	if lengthByte == 0x80 {
		if !element.constructed {
			return berElement{}, nil, errMalformedEnvelope
		}
		return readIndefinite(data, offset, element)
	}

	length := int(lengthByte)
	if lengthByte&0x80 != 0 {
		count := int(lengthByte & 0x7F)
		if count == 0 || count > 4 || offset+count > len(data) {
			return berElement{}, nil, errMalformedEnvelope
		}
		length = 0
		for _, b := range data[offset : offset+count] {
			length = length<<8 | int(b)
		}
		offset += count
	}
	if length < 0 || length > len(data)-offset {
		return berElement{}, nil, errMalformedEnvelope
	}
	element.content = data[offset : offset+length]
	return element, data[offset+length:], nil
}

func readIndefinite(
	data []byte,
	offset int,
	element berElement,
) (berElement, []byte, error) {
	rest := data[offset:]
	consumed := 0
	for {
		if len(rest) < 2 {
			return berElement{}, nil, errMalformedEnvelope
		}
		if rest[0] == 0 && rest[1] == 0 {
			element.content = data[offset : offset+consumed]
			return element, rest[2:], nil
		}
		_, remaining, err := readBERElement(rest, element.depth+1)
		if err != nil {
			return berElement{}, nil, err
		}
		consumed += len(rest) - len(remaining)
		rest = remaining
	}
}
