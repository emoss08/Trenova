package qrutils

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

const pngDataURIPrefix = "data:image/png;base64,"

var ErrEmptyContent = errors.New("qr content must not be empty")

func PNGDataURI(content string, size int) (string, error) {
	if content == "" {
		return "", ErrEmptyContent
	}
	if size <= 0 {
		return "", fmt.Errorf("qr size must be positive, got %d", size)
	}

	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return "", fmt.Errorf("encode qr code: %w", err)
	}

	scaled, err := barcode.Scale(code, size, size)
	if err != nil {
		return "", fmt.Errorf("scale qr code: %w", err)
	}

	var buf bytes.Buffer
	if err = png.Encode(&buf, scaled); err != nil {
		return "", fmt.Errorf("render qr code: %w", err)
	}

	return pngDataURIPrefix + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
