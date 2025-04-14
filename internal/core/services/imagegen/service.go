package imagegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/gen2brain/go-fitz"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Logger *logger.Logger
}

type service struct {
	l *zerolog.Logger
}

func NewService(p Params) services.ImageGenerationService {
	log := p.Logger.With().Str("service", "imagegen").Logger()

	return &service{
		l: &log,
	}
}

// ConvertToImage converts various document types to images
func (s *service) ConvertToImage(ctx context.Context, req *services.ConvertToImageRequest) (*services.ConvertToImageResponse, error) {
	log := s.l.With().
		Str("operation", "ConvertToImage").
		Str("filePath", req.FilePath).
		Str("outputPath", req.OutputPath).
		Logger()

	// Get file extension to determine converter to use
	ext := filepath.Ext(req.FilePath)
	ext = strings.ToLower(ext)

	// Select appropriate converter based on file type
	var err error
	switch ext {
	case ".pdf":
		err = s.convertPDFToImage(req.FilePath, req.OutputPath, req.Options)
	case ".docx", ".doc":
		err = s.convertDocToImage(ctx, req.FilePath, req.OutputPath, req.Options)
	case ".xlsx", ".xls":
		err = s.convertSpreadsheetToImage(ctx, req.FilePath, req.OutputPath, req.Options)
	case ".pptx", ".ppt":
		err = s.convertPresentationToImage(ctx, req.FilePath, req.OutputPath, req.Options)
	default:
		return nil, errors.NewBusinessError(fmt.Sprintf("Unsupported file type: %s", ext))
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to convert file to image")
		return nil, err
	}

	// Get image properties if needed
	fileInfo, err := os.Stat(req.OutputPath)
	if err != nil {
		log.Warn().Err(err).Msg("failed to get output file info")
	}

	return &services.ConvertToImageResponse{
		OutputPath: req.OutputPath,
		FileSize:   fileInfo.Size(),
	}, nil
}

// convertPDFToImage converts a PDF document to an image
func (s *service) convertPDFToImage(pdfPath, outputPath string, options *services.ConversionOptions) error {
	// Open the PDF document
	doc, err := fitz.New(pdfPath)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to open PDF document")
		return err
	}
	defer doc.Close()

	// Check if the document has at least one page
	if doc.NumPage() < 1 {
		s.l.Error().Msg("document has no pages")
		return errors.NewBusinessError("Document has no pages")
	}

	// Determine which page to use
	pageNum := 0
	if options != nil && options.PageNumber != nil {
		// Ensure page number is valid
		if *options.PageNumber >= 0 && *options.PageNumber < doc.NumPage() {
			pageNum = *options.PageNumber
		} else {
			s.l.Warn().Int("requestedPage", *options.PageNumber).Int("totalPages", doc.NumPage()).
				Msg("invalid page number requested, using first page")
		}
	}

	// Extract the specified page as an image
	img, err := doc.Image(pageNum)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to extract page as image")
		return err
	}

	// Create output file
	out, err := os.Create(outputPath)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to create output image file")
		return err
	}
	defer out.Close()

	// Determine quality and format
	quality := 90
	if options != nil && options.Quality != nil {
		quality = *options.Quality
	}

	// Encode to WebP for better compression while maintaining quality
	if err = webp.Encode(out, img, &webp.Options{Quality: float32(quality)}); err != nil {
		s.l.Error().Err(err).Msg("failed to encode image")
		return err
	}

	return nil
}

// convertDocToImage converts a Word document to an image
//
//nolint:revive // Not implemented yet
func (s *service) convertDocToImage(ctx context.Context, docPath, outputPath string, options *services.ConversionOptions) error {
	// Implementation for Word documents
	// This would typically involve a library like unidoc, libreoffice headless, etc.
	return errors.NewBusinessError("Word document conversion not yet implemented")
}

// convertSpreadsheetToImage converts a spreadsheet to an image
//
//nolint:revive // Not implemented yet
func (s *service) convertSpreadsheetToImage(ctx context.Context, spreadsheetPath, outputPath string, options *services.ConversionOptions) error {
	// Implementation for spreadsheets
	return errors.NewBusinessError("Spreadsheet conversion not yet implemented")
}

// convertPresentationToImage converts a presentation to an image
//
//nolint:revive // Not implemented yet
func (s *service) convertPresentationToImage(ctx context.Context, presentationPath, outputPath string, options *services.ConversionOptions) error {
	// Implementation for presentations
	return errors.NewBusinessError("Presentation conversion not yet implemented")
}

// ResizeImage resizes an image to specified dimensions
//
//nolint:revive // Not implemented yet
func (s *service) ResizeImage(ctx context.Context, req *services.ResizeImageRequest) (*services.ResizeImageResponse, error) {
	// Implementation for image resizing
	// This could use imaging library or similar
	return nil, errors.NewBusinessError("Image resizing not yet implemented")
}

// OptimizeImage compresses an image with specified quality
//
//nolint:revive // Not implemented yet
func (s *service) OptimizeImage(ctx context.Context, req *services.OptimizeImageRequest) (*services.OptimizeImageResponse, error) {
	// Implementation for image optimization
	return nil, errors.NewBusinessError("Image optimization not yet implemented")
}
