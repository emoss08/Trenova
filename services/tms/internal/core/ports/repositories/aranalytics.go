package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetARAnalyticsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type ListARSeriesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
	Weeks      int                   `json:"weeks"`
}

type ListARCashFlowRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate    int64                 `json:"asOfDate"`
	PastWeeks   int                   `json:"pastWeeks"`
	FutureWeeks int                   `json:"futureWeeks"`
}

type GetARCollectionMetricsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
	PeriodDays int                   `json:"periodDays"`
}

type ListARTopOverdueCustomersRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
	Limit      int                   `json:"limit"`
}

type ListARCollectionsWorklistRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
	Limit      int                   `json:"limit"`
}

type GetARCustomerSnapshotRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CustomerID pulid.ID              `json:"customerId"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type ARBalanceOverview struct {
	TotalOpenMinor       int64               `json:"totalOpenMinor"`
	OverdueMinor         int64               `json:"overdueMinor"`
	UnappliedCashMinor   int64               `json:"unappliedCashMinor"`
	DisputedOpenMinor    int64               `json:"disputedOpenMinor"`
	OpenInvoiceCount     int                 `json:"openInvoiceCount"`
	OverdueInvoiceCount  int                 `json:"overdueInvoiceCount"`
	DisputedInvoiceCount int                 `json:"disputedInvoiceCount"`
	AvgDaysPastDue       float64             `json:"avgDaysPastDue"`
	Buckets              ARAgingBucketTotals `json:"buckets"`
}

type ARBalancePoint struct {
	PeriodEnd      int64 `json:"periodEnd"`
	ARBalanceMinor int64 `json:"arBalanceMinor"`
	BilledMinor    int64 `json:"billedMinor"`
}

type ARAgingTrendPoint struct {
	PeriodEnd int64               `json:"periodEnd"`
	Buckets   ARAgingBucketTotals `json:"buckets"`
}

type ARCashFlowPoint struct {
	WeekStart     int64 `json:"weekStart"`
	ExpectedMinor int64 `json:"expectedMinor"`
	OpenDueMinor  int64 `json:"openDueMinor"`
	ActualMinor   int64 `json:"actualMinor"`
	IsForecast    bool  `json:"isForecast"`
}

type ARCollectionTotals struct {
	PeriodStart              int64   `json:"periodStart"`
	PeriodEnd                int64   `json:"periodEnd"`
	BeginningOpenMinor       int64   `json:"beginningOpenMinor"`
	EndingOpenMinor          int64   `json:"endingOpenMinor"`
	EndingCurrentMinor       int64   `json:"endingCurrentMinor"`
	CreditSalesMinor         int64   `json:"creditSalesMinor"`
	CollectedMinor           int64   `json:"collectedMinor"`
	AvgDaysToPay             float64 `json:"avgDaysToPay"`
	ShortPayMinor            int64   `json:"shortPayMinor"`
	ShortPayApplicationCount int     `json:"shortPayApplicationCount"`
	ApplicationCount         int     `json:"applicationCount"`
	DisputedInvoiceCount     int     `json:"disputedInvoiceCount"`
	PostedInvoiceCount       int     `json:"postedInvoiceCount"`
}

type ARTopOverdueCustomer struct {
	CustomerID        pulid.ID `json:"customerId"`
	CustomerName      string   `json:"customerName"`
	OverdueMinor      int64    `json:"overdueMinor"`
	TotalOpenMinor    int64    `json:"totalOpenMinor"`
	OldestDaysPastDue int      `json:"oldestDaysPastDue"`
	OpenInvoiceCount  int      `json:"openInvoiceCount"`
}

type ARCollectionsWorklistItem struct {
	InvoiceID       pulid.ID `json:"invoiceId"`
	CustomerID      pulid.ID `json:"customerId"`
	CustomerName    string   `json:"customerName"`
	InvoiceNumber   string   `json:"invoiceNumber"`
	DueDate         int64    `json:"dueDate"`
	OpenAmountMinor int64    `json:"openAmountMinor"`
	DaysPastDue     int      `json:"daysPastDue"`
	IsDisputed      bool     `json:"isDisputed"`
	HasShortPay     bool     `json:"hasShortPay"`
	Severity        string   `json:"severity"`
}

type ARMonthlyCollectionPoint struct {
	MonthStart  int64 `json:"monthStart"`
	AmountMinor int64 `json:"amountMinor"`
}

type ARCustomerSnapshot struct {
	CustomerID            pulid.ID                    `json:"customerId"`
	CustomerName          string                      `json:"customerName"`
	TotalOpenMinor        int64                       `json:"totalOpenMinor"`
	OverdueMinor          int64                       `json:"overdueMinor"`
	UnappliedCashMinor    int64                       `json:"unappliedCashMinor"`
	CreditLimitMinor      int64                       `json:"creditLimitMinor"`
	HasCreditLimit        bool                        `json:"hasCreditLimit"`
	OpenInvoiceCount      int                         `json:"openInvoiceCount"`
	OldestOpenInvoiceDate int64                       `json:"oldestOpenInvoiceDate"`
	OldestDaysPastDue     int                         `json:"oldestDaysPastDue"`
	LastPaymentDate       int64                       `json:"lastPaymentDate"`
	LastPaymentMinor      int64                       `json:"lastPaymentMinor"`
	AvgDaysToPay          float64                     `json:"avgDaysToPay"`
	BilledTrailing91Minor int64                       `json:"billedTrailing91Minor"`
	Buckets               ARAgingBucketTotals         `json:"buckets"`
	MonthlyCollections    []*ARMonthlyCollectionPoint `json:"monthlyCollections"`
}

type ARPaymentStats struct {
	PostedTodayMinor      int64 `json:"postedTodayMinor"`
	PostedTodayCount      int   `json:"postedTodayCount"`
	UnappliedCashMinor    int64 `json:"unappliedCashMinor"`
	UnappliedPaymentCount int   `json:"unappliedPaymentCount"`
	ReversedLast30Minor   int64 `json:"reversedLast30Minor"`
	ReversedLast30Count   int   `json:"reversedLast30Count"`
}

type ARAnalyticsRepository interface {
	GetPaymentStats(
		ctx context.Context,
		req GetARAnalyticsRequest,
	) (*ARPaymentStats, error)
	GetBalanceOverview(
		ctx context.Context,
		req GetARAnalyticsRequest,
	) (*ARBalanceOverview, error)
	ListBalanceSeries(
		ctx context.Context,
		req ListARSeriesRequest,
	) ([]*ARBalancePoint, error)
	ListAgingTrend(
		ctx context.Context,
		req ListARSeriesRequest,
	) ([]*ARAgingTrendPoint, error)
	ListCashFlow(
		ctx context.Context,
		req ListARCashFlowRequest,
	) ([]*ARCashFlowPoint, error)
	GetCollectionTotals(
		ctx context.Context,
		req GetARCollectionMetricsRequest,
	) (*ARCollectionTotals, error)
	ListTopOverdueCustomers(
		ctx context.Context,
		req ListARTopOverdueCustomersRequest,
	) ([]*ARTopOverdueCustomer, error)
	ListCollectionsWorklist(
		ctx context.Context,
		req ListARCollectionsWorklistRequest,
	) ([]*ARCollectionsWorklistItem, error)
	GetCustomerSnapshot(
		ctx context.Context,
		req GetARCustomerSnapshotRequest,
	) (*ARCustomerSnapshot, error)
}
