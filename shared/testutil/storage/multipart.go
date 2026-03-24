package storage

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
)

type MockFileHeader struct {
	Filename    string
	Size        int64
	ContentType string
	Data        []byte
}

func NewMockFileHeader(filename string, data []byte, contentType string) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, _ := writer.CreatePart(h)
	_, _ = part.Write(data)
	_ = writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(int64(len(data)) + 1024)

	if files, ok := form.File["file"]; ok && len(files) > 0 {
		return files[0]
	}

	return &multipart.FileHeader{
		Filename: filename,
		Size:     int64(len(data)),
		Header:   textproto.MIMEHeader{"Content-Type": []string{contentType}},
	}
}

func NewMockFileHeaders(files []MockFileHeader) []*multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, f := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="files"; filename="`+f.Filename+`"`)
		h.Set("Content-Type", f.ContentType)

		part, _ := writer.CreatePart(h)
		_, _ = part.Write(f.Data)
	}
	_ = writer.Close()

	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(10 * 1024 * 1024)

	return form.File["files"]
}

type MockMultipartFile struct {
	*bytes.Reader
	filename string
}

func NewMockMultipartFile(data []byte, filename string) *MockMultipartFile {
	return &MockMultipartFile{
		Reader:   bytes.NewReader(data),
		filename: filename,
	}
}

func (m *MockMultipartFile) Close() error {
	return nil
}

func (m *MockMultipartFile) ReadAt(p []byte, off int64) (n int, err error) {
	return m.Reader.ReadAt(p, off)
}

func (m *MockMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return m.Reader.Seek(offset, whence)
}

var _ io.ReadCloser = (*MockMultipartFile)(nil)
var _ io.ReaderAt = (*MockMultipartFile)(nil)
var _ io.Seeker = (*MockMultipartFile)(nil)
