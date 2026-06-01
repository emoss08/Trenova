package encodingutils

// EncodedBase64Size returns the number of bytes required to Base64-encode a
// payload of size bytes with standard padding.
func EncodedBase64Size(size int64) int64 {
	if size <= 0 {
		return 0
	}
	return ((size + 2) / 3) * 4
}
