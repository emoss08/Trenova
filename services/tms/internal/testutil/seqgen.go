package testutil

import (
	"context"

	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
)

type TestSequenceGenerator struct {
	SingleValue string
	BatchValues []string
}

func (g TestSequenceGenerator) Generate(
	_ context.Context,
	_ *seqgen.GenerateRequest,
) (string, error) {
	return g.SingleValue, nil
}

func (g TestSequenceGenerator) GenerateBatch(
	_ context.Context,
	req *seqgen.GenerateRequest,
) ([]string, error) {
	if req == nil || req.Count <= 0 {
		return []string{}, nil
	}

	if len(g.BatchValues) == 0 {
		return make([]string, req.Count), nil
	}

	if req.Count > len(g.BatchValues) {
		values := make([]string, req.Count)
		copy(values, g.BatchValues)
		return values, nil
	}

	return append([]string(nil), g.BatchValues[:req.Count]...), nil
}

func (g TestSequenceGenerator) GenerateShipmentProNumber(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
	_ string,
	_ string,
) (string, error) {
	return g.SingleValue, nil
}

func (g TestSequenceGenerator) GenerateConsolidationNumber(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
	_ string,
	_ string,
) (string, error) {
	return g.SingleValue, nil
}

func (g TestSequenceGenerator) GenerateInvoiceNumber(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
	_ string,
	_ string,
) (string, error) {
	return g.SingleValue, nil
}

func (g TestSequenceGenerator) GenerateWorkOrderNumber(
	_ context.Context,
	_ pulid.ID,
	_ pulid.ID,
	_ string,
	_ string,
) (string, error) {
	return g.SingleValue, nil
}
