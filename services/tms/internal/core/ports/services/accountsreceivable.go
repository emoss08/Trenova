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
	CustomerID          pulid.ID                     `json:"customerId"`
	CustomerName        string                       `json:"customerName"`
	StatementDate       int64                        `json:"statementDate"`
	StartDate           int64                        `json:"startDate"`
	OpeningBalanceMinor int64                        `json:"openingBalanceMinor"`
	TotalChargesMinor   int64                        `json:"totalChargesMinor"`
	TotalPaymentsMinor  int64                        `json:"totalPaymentsMinor"`
	EndingBalanceMinor  int64                        `json:"endingBalanceMinor"`
	Aging               repositories.ARAgingBucketTotals `json:"aging"`
	Transactions        []*ARStatementTransaction    `json:"transactions"`
	OpenItems           []*repositories.AROpenItem   `json:"openItems"`
}

type ARAgingSummary struct {
	AsOfDate int64                         `json:"asOfDate"`
	Totals   repositories.ARAgingBucketTotals `json:"totals"`
	Rows     []*repositories.ARCustomerAgingRow `json:"rows"`
}
