//revive:disable-next-line:var-naming
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
)

const managedDataUsagePath = "/managed/v1/data-usage"

type ManagedDataUsage interface {
	Usage(ctx context.Context) *datausageports.Usage
}

type managedDataUsage struct {
	MeasuredAt     time.Time `json:"measuredAt"`
	DatabaseBytes  *int64    `json:"databaseBytes,omitempty"`
	DocumentsBytes *int64    `json:"documentsBytes,omitempty"`
	SearchBytes    *int64    `json:"searchBytes,omitempty"`
}

func (h *managedHandler) routeDataUsage(mux *http.ServeMux) {
	mux.HandleFunc("GET "+managedDataUsagePath, h.serveDataUsage)
}

func (h *managedHandler) serveDataUsage(w http.ResponseWriter, r *http.Request) {
	usage := h.dataUsage.Usage(r.Context())
	h.json(w, http.StatusOK, managedDataUsage{
		MeasuredAt:     time.Unix(usage.MeasuredAt, 0).UTC(),
		DatabaseBytes:  usage.DatabaseBytes,
		DocumentsBytes: usage.DocumentsBytes,
		SearchBytes:    usage.SearchBytes,
	})
}
