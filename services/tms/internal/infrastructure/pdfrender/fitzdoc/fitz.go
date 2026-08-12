//go:build !nofitz

package fitzdoc

import (
	"fmt"

	"github.com/gen2brain/go-fitz"
)

func Available() bool { return true }

func NewFromMemory(data []byte) (Document, error) {
	doc, err := fitz.NewFromMemory(data)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}

	return doc, nil
}
