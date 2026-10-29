package runtimetest

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
)

func KeptShipments(rows int) agentruntimeports.ShownArtifact {
	labels := make([]string, 0, rows)
	for idx := range rows {
		labels = append(labels, fmt.Sprintf("PRO-%d", idx+1))
	}

	return agentruntimeports.ShownArtifact{
		ID:     pulid.MustNew("art_"),
		Kind:   "table_view",
		Title:  "Late shipments",
		Rows:   rows,
		Labels: labels,
	}
}

func MarkdownReprint(rows int) string {
	var b strings.Builder
	b.WriteString("Here they are:\n| Pro | Status |\n| --- | --- |\n")
	for idx := range rows {
		fmt.Fprintf(&b, "| **PRO-%d** | Late |\n", idx+1)
	}
	b.WriteString("\nCall the carriers first.")

	return b.String()
}
