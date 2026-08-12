//go:build nofitz

package fitzdoc

func Available() bool { return false }

func NewFromMemory(_ []byte) (Document, error) {
	return nil, ErrUnavailable
}
