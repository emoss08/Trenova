package render

import (
	"errors"
	"fmt"
	"os"

	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/spf13/cobra"
)

var contentType string

var RenderCmd = &cobra.Command{
	Use:           thumbnailservice.RenderCommandName,
	Short:         "Render a document thumbnail from stdin to stdout",
	Hidden:        true,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		generator := thumbnailservice.NewInProcessGenerator()

		thumb, err := generator.Generate(cmd.Context(), os.Stdin, contentType)
		if err != nil {
			if errors.Is(err, thumbnailservice.ErrPDFHasNoPages) {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(thumbnailservice.RenderExitNoPages)
			}
			return err
		}

		if _, err = os.Stdout.Write(thumb); err != nil {
			return fmt.Errorf("failed to write thumbnail: %w", err)
		}

		return nil
	},
}

func init() {
	RenderCmd.Flags().
		StringVar(&contentType, "content-type", "application/pdf", "MIME type of the input document")
}
