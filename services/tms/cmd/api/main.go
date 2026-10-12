package main

import (
	"github.com/emoss08/trenova/internal/bootstrap/apiapp"
	"github.com/emoss08/trenova/internal/bootstrap/core"
	"github.com/emoss08/trenova/internal/bootstrap/edition"
)

func main() {
	apiapp.Run(core.OptionsFor(edition.Current()))
}
