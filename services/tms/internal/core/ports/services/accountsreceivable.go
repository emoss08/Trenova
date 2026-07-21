package services

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

type ARStatementTransaction struct {
	TransactionDate     int64  `json:"transactionDate"`
	EventType           string `json:"eventType"`
	DocumentNumber      string `json:"documentNumber"`
	SourceObjectID      string `json:"sourceObjectId"`
	AmountMinor         int64  `json:"amountMinor"`
	ChargeMinor         int64  `json:"chargeMinor"`
	PaymentMinor        int64  `json:"paymentMinor"`
	RunningBalanceMinor int64  `json:"runningBalanceMinor"`
}

type ARCustomerStatement struct {
	CustomerID          pulid.ID                         `json:"customerId"`
	CustomerName        string                           `json:"customerName"`
	StatementDate       int64                            `json:"statementDate"`
	StartDate           int64                            `json:"startDate"`
	OpeningBalanceMinor int64                            `json:"openingBalanceMinor"`
	TotalChargesMinor   int64                            `json:"totalChargesMinor"`
	TotalPaymentsMinor  int64                            `json:"totalPaymentsMinor"`
	EndingBalanceMinor  int64                            `json:"endingBalanceMinor"`
	Aging               repositories.ARAgingBucketTotals `json:"aging"`
	Transactions        []*ARStatementTransaction        `json:"transactions"`
	OpenItems           []*repositories.AROpenItem       `json:"openItems"`
}

type ARAgingSummary struct {
	AsOfDate int64                              `json:"asOfDate"`
	Totals   repositories.ARAgingBucketTotals   `json:"totals"`
	Rows     []*repositories.ARCustomerAgingRow `json:"rows"`
}

type ARDSOTrendPoint struct {
	PeriodEnd      int64   `json:"periodEnd"`
	DSODays        float64 `json:"dsoDays"`
	ARBalanceMinor int64   `json:"arBalanceMinor"`
	BilledMinor    int64   `json:"billedMinor"`
}

type ARDashboardKPIs struct {
	AsOfDate       int64                           `json:"asOfDate"`
	Overview       *repositories.ARBalanceOverview `json:"overview"`
	CurrentDSODays float64                         `json:"currentDsoDays"`
	DSODeltaDays   float64                         `json:"dsoDeltaDays"`
	CEI            float64                         `json:"cei"`
	AvgDaysToPay   float64                         `json:"avgDaysToPay"`
	OverduePercent float64                         `json:"overduePercent"`
	WriteOffRatio  float64                         `json:"writeOffRatio"`
	DisputeRate    float64                         `json:"disputeRate"`
	ShortPayRate   float64                         `json:"shortPayRate"`
}

type ARCollectionPerformance struct {
	Totals        *repositories.ARCollectionTotals `json:"totals"`
	CEI           float64                          `json:"cei"`
	WriteOffRatio float64                          `json:"writeOffRatio"`
	DisputeRate   float64                          `json:"disputeRate"`
	ShortPayRate  float64                          `json:"shortPayRate"`
}

type ARCustomerProfile struct {
	Snapshot          *repositories.ARCustomerSnapshot `json:"snapshot"`
	DSODays           float64                          `json:"dsoDays"`
	CreditUtilization float64                          `json:"creditUtilization"`
	DelinquencyScore  float64                          `json:"delinquencyScore"`
}
