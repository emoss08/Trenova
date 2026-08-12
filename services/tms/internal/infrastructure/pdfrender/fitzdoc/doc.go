package fitzdoc

import (
	"errors"
	"image"
)

var ErrUnavailable = errors.New(
	"PDF rendering unavailable: binary was built with the 'nofitz' build tag",
)

type Document interface {
	Close() error
	NumPage() int
	Image(pageNumber int) (*image.RGBA, error)
	ImagePNG(pageNumber int, dpi float64) ([]byte, error)
	Text(pageNumber int) (string, error)
}
