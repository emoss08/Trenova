package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/emailpreview"
	"github.com/emoss08/trenova/shared/i18n"
)

func main() {
	out := flag.String("out", "build/email-preview", "the directory to write the preview into")
	languages := flag.String("locales", "en,es", "the languages to render, comma-separated")
	flag.Parse()

	locales := make([]i18n.Locale, 0, 2)
	for _, tag := range strings.Split(*languages, ",") {
		locale, ok := i18n.Parse(strings.TrimSpace(tag))
		if !ok {
			fmt.Fprintf(os.Stderr, "email-preview: %q is not a supported language\n", tag)
			os.Exit(2)
		}
		locales = append(locales, locale)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := emailpreview.Write(ctx, *out, locales)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "email-preview:", err)
		os.Exit(1)
	}
	fmt.Println("email-preview: wrote", *out+"/index.html")
}
