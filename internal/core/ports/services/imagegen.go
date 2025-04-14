package services

import "context"

// ImageGenerationService handles conversion of various document types to images
type ImageGenerationService interface {
	// ConvertToImage converts a document to an image
	ConvertToImage(ctx context.Context, req *ConvertToImageRequest) (*ConvertToImageResponse, error)

	// ResizeImage resizes an image to specified dimensions
	ResizeImage(ctx context.Context, req *ResizeImageRequest) (*ResizeImageResponse, error)

	// OptimizeImage compresses an image with specified quality settings
	OptimizeImage(ctx context.Context, req *OptimizeImageRequest) (*OptimizeImageResponse, error)
}

// ConvertToImageRequest contains parameters for document to image conversion
type ConvertToImageRequest struct {
	FilePath   string             // Path to the input file
	OutputPath string             // Path where the output image should be saved
	Options    *ConversionOptions // Optional conversion parameters
}

// ConversionOptions provides customization for the conversion process
type ConversionOptions struct {
	PageNumber *int    // Specific page to convert (for multi-page documents)
	Quality    *int    // Output quality (1-100)
	MaxWidth   *int    // Maximum width in pixels
	MaxHeight  *int    // Maximum height in pixels
	Format     *string // Output format (webp, jpg, png)
}

// ConvertToImageResponse contains the result of a conversion operation
type ConvertToImageResponse struct {
	OutputPath string // Path to the generated image
	FileSize   int64  // Size of the generated image in bytes
}

// ResizeImageRequest contains parameters for image resizing
type ResizeImageRequest struct {
	ImagePath  string
	OutputPath string
	Width      int
	Height     int
	Maintain   bool // Maintain aspect ratio
}

// ResizeImageResponse contains the result of a resize operation
type ResizeImageResponse struct {
	OutputPath string
	Width      int
	Height     int
	FileSize   int64
}

// OptimizeImageRequest contains parameters for image optimization
type OptimizeImageRequest struct {
	ImagePath  string
	OutputPath string
	Quality    int
	Format     string // webp, jpg, png
}

// OptimizeImageResponse contains the result of an optimization operation
type OptimizeImageResponse struct {
	OutputPath     string
	FileSize       int64
	CompressionPct float64 // Percentage of size reduction
}
