package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
)

var ErrOfficeDocumentUnreadable = errors.New("the office document could not be read")

type OfficeFormat string

const (
	OfficeFormatDOCX OfficeFormat = "docx"
	OfficeFormatXLSX OfficeFormat = "xlsx"
	OfficeFormatPPTX OfficeFormat = "pptx"
	OfficeFormatEPUB OfficeFormat = "epub"
)

var officeContentTypes = map[string]OfficeFormat{
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   OfficeFormatDOCX,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         OfficeFormatXLSX,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": OfficeFormatPPTX,
	"application/epub+zip": OfficeFormatEPUB,
}

func OfficeFormatOf(contentType, fileName string) (OfficeFormat, bool) {
	ct, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(contentType)), ";")
	if format, ok := officeContentTypes[strings.TrimSpace(ct)]; ok {
		return format, true
	}

	switch OfficeFormat(strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")) {
	case OfficeFormatDOCX:
		return OfficeFormatDOCX, true
	case OfficeFormatXLSX:
		return OfficeFormatXLSX, true
	case OfficeFormatPPTX:
		return OfficeFormatPPTX, true
	case OfficeFormatEPUB:
		return OfficeFormatEPUB, true
	default:
		return "", false
	}
}

type OfficeImage struct {
	Name string
	Ext  string
	Data []byte
}

type OfficePage struct {
	Text   string
	Images []OfficeImage
}

type OfficeDocumentReader interface {
	Read(ctx context.Context, data []byte, format OfficeFormat) ([]OfficePage, error)
}
