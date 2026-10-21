package fileutils

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
)

var ErrMultipartSizeMismatch = errors.New("uploaded file is larger than its declared size")

func ReadMultipart(header *multipart.FileHeader) ([]byte, error) {
	if header == nil {
		return nil, errors.New("uploaded file is missing")
	}
	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("open uploaded file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, header.Size+1))
	if err != nil {
		return nil, fmt.Errorf("read uploaded file: %w", err)
	}
	if int64(len(content)) > header.Size {
		return nil, ErrMultipartSizeMismatch
	}
	return content, nil
}
