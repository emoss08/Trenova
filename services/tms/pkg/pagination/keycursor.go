package pagination

func EncodeKeyCursor(binding CursorBinding, key string) string {
	return sealCursor(cursorKindKey, binding, []byte(key))
}

func DecodeKeyCursor(binding CursorBinding, encoded string) (string, error) {
	raw, err := openCursor(cursorKindKey, binding, encoded)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}
