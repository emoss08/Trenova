package fileutils

import "mime"

func MediaType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}

	return mediaType
}
