package officedoc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/xuri/excelize/v2"
)

func readXLSX(ctx context.Context, data []byte) (pages []services.OfficePage, err error) {
	book, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit:    maxTotalBytes,
		UnzipXMLSizeLimit: maxPartBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	defer func() {
		err = errors.Join(err, book.Close())
	}()

	builder := newPageBuilder()
	for _, sheet := range book.GetSheetList() {
		if builder.done() {
			break
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		if err = readSheet(ctx, book, sheet, builder); err != nil {
			return nil, err
		}
		builder.finishPage()
	}

	return builder.pages, nil
}

func readSheet(ctx context.Context, book *excelize.File, sheet string, pages *pageBuilder) error {
	rows, err := book.Rows(sheet)
	if err != nil {
		return fmt.Errorf("read sheet %q: %w", sheet, err)
	}

	count := 0
	for rows.Next() {
		count++
		if count%ctxCheckEvery == 0 {
			if err = ctx.Err(); err != nil {
				return errors.Join(err, rows.Close())
			}
		}

		cells, colErr := rows.Columns()
		if colErr != nil {
			return errors.Join(fmt.Errorf("read sheet %q: %w", sheet, colErr), rows.Close())
		}
		if line := joinCells(cells); line != "" {
			pages.write(line)
			pages.write("\n")
		}
	}
	if err = rows.Close(); err != nil {
		return fmt.Errorf("read sheet %q: %w", sheet, err)
	}

	cells, err := book.GetPictureCells(sheet)
	if err != nil {
		return fmt.Errorf("read pictures of sheet %q: %w", sheet, err)
	}
	for _, cell := range cells {
		pictures, picErr := book.GetPictures(sheet, cell)
		if picErr != nil {
			return fmt.Errorf("read pictures at %s!%s: %w", sheet, cell, picErr)
		}
		for i, picture := range pictures {
			ext := strings.ToLower(picture.Extension)
			if _, ok := ocrImageExts[ext]; !ok {
				continue
			}
			pages.attach(fmt.Sprintf("%s-%s-%d%s", sheet, cell, i+1, ext), ext, picture.File)
		}
	}

	return nil
}

func joinCells(cells []string) string {
	end := len(cells)
	for end > 0 && strings.TrimSpace(cells[end-1]) == "" {
		end--
	}
	if end == 0 {
		return ""
	}
	return strings.Join(cells[:end], "\t")
}
