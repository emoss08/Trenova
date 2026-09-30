package invoiceservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ListEmailAttempts(
	ctx context.Context,
	req repositories.ListInvoiceEmailAttemptsRequest,
) (*pagination.ListResult[*invoice.EmailAttempt], error) {
	return s.repo.ListEmailAttempts(ctx, req)
}

func invoicePDFName(entity *invoice.Invoice) string {
	number := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(entity.Number)
	return "invoice-" + number + ".pdf"
}

func unixDate(value int64) string {
	if value == 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format("2006-01-02")
}

func unixDatePtr(value *int64) string {
	if value == nil {
		return ""
	}
	return unixDate(*value)
}

func moneyString(currency, amount string) string {
	if currency == "" {
		currency = "USD"
	}
	return currency + " " + amount
}

func actorUserID(actor *servicesports.RequestActor, tenantInfo pagination.TenantInfo) pulid.ID {
	if actor != nil && actor.UserID.IsNotNil() {
		return actor.UserID
	}
	return tenantInfo.UserID
}
