package fileutils

import (
	"crypto/rand"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
)

func ReadFileData(file *multipart.FileHeader) ([]byte, error) {
	fileData, err := file.Open()
	if err != nil {
		return nil, eris.Wrap(err, "open file")
	}
	defer func(fData multipart.File) {
		err = fData.Close()
		if err != nil {
			log.Error().Err(err).Msg("failed to close file")
		}
	}(fileData)

	return io.ReadAll(fileData)
}

func RenameFile(file *multipart.FileHeader, objectID string) (string, error) {
	randomName, err := ulid.New(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0))
	if err != nil {
		return "", eris.Wrap(err, "generate random name")
	}

	fileExt := filepath.Ext(file.Filename)
	if fileExt == "" {
		return "", eris.New("file has no extension")
	}

	newFileName := objectID + "_" + randomName.String() + fileExt

	return newFileName, nil
}
