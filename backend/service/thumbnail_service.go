package service

import (
	"image"
	"os"

	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"github.com/nfnt/resize"
)

func GenerateThumbnail(inputPath, outputPath string, width, height uint) error {
	// Open the file.

	file, err := os.Open(inputPath)

	if err != nil {
		return err
	}

	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Reisze the image to specified width and heigh using Lanczos resampling
	// and preserve aspect ratio
	thumbnail := resize.Thumbnail(width, height, img, resize.Lanczos3)

	// Create the output file.
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	options, err := encoder.NewLosslessEncoderOptions(encoder.PresetDefault, 75)
	if err != nil {
		return err
	}

	// Encode the image using the WebP encoder.
	if err := webp.Encode(out, thumbnail, options); err != nil {
		return err
	}

	return nil
}
