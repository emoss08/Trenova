package storage

import (
	"strings"
)

func SumUploadedPartSizes(parts []UploadedPart) int64 {
	var total int64
	for _, part := range parts {
		total += part.Size
	}
	return total
}

func IsMissingMultipartUploadError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "multipart upload does not exist")
}

func ToDomainParts(parts []UploadedPart) []UploadedPart {
	result := make([]UploadedPart, 0, len(parts))
	for _, part := range parts {
		result = append(result, UploadedPart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
			Size:       part.Size,
		})
	}
	return result
}
