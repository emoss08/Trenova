package modeladapter

// FileReport is which of a request's files were sent as themselves, and which
// were sent as their text instead.
type FileReport struct {
	Sent      []string
	Fallbacks []FileFallback
}

// FileFallback is a file sent as its text, and why.
type FileFallback struct {
	DocumentID string
	Reason     string
}
