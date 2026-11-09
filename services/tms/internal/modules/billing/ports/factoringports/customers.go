package factoringports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CustomerNoticeStage string

const (
	CustomerNoticeAll          = CustomerNoticeStage("")
	CustomerNoticeNeeded       = CustomerNoticeStage("NeedsNotice")
	CustomerNoticeAwaiting     = CustomerNoticeStage("Awaiting")
	CustomerNoticeAcknowledged = CustomerNoticeStage("Acknowledged")
)

type ListFactorCustomersRequest struct {
	Filter             *pagination.QueryOptions `json:"filter"`
	Cursor             pagination.CursorInfo    `json:"-"`
	FactoringCompanyID pulid.ID                 `json:"factoringCompanyId"`
	Stage              CustomerNoticeStage      `json:"stage"`
	At                 int64                    `json:"at"`
}

type FactorCustomer struct {
	CustomerID     pulid.ID           `json:"customerId"`
	Code           string             `json:"code"`
	Name           string             `json:"name"`
	Status         domaintypes.Status `json:"status"`
	Notice         *factoring.Noa     `json:"notice"`
	LastReleasedAt *int64             `json:"lastReleasedAt"`
	FundedCount    int                `json:"fundedCount"`
	FundedMinor    int64              `json:"fundedMinor"`
}

type FactorCustomerPage struct {
	Items           []*FactorCustomer      `json:"items"`
	HasNextPage     bool                   `json:"hasNextPage"`
	TotalCount      *int                   `json:"totalCount"`
	TotalCountExact bool                   `json:"totalCountExact"`
	EdgeCursors     pagination.EdgeCursors `json:"-"`
}

type FactorCustomerRepository interface {
	List(ctx context.Context, req *ListFactorCustomersRequest) (*FactorCustomerPage, error)
}
