package main

import (
	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/cloud"
)

func init() {
	edition.Register(cloud.Edition())
}
