package main

import (
	"github.com/emoss08/trenova/internal/bootstrap/apiapp"
	"github.com/emoss08/trenova/internal/bootstrap/core"
)

func main() {
	apiapp.Run(core.Options())
}
