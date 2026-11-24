package bankreceiptbatchservice

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type countingWaker struct{ woken []pagination.TenantInfo }

func (w *countingWaker) Wake(_ context.Context, tenant pagination.TenantInfo) {
	w.woken = append(w.woken, tenant)
}
