package main

import (
	"github.com/emoss08/keel/analytics/analyticscli"
	"github.com/emoss08/trenova/internal/keeltables"
)

func main() {
	analyticscli.Main(analyticscli.Config{Tables: keeltables.All()})
}
