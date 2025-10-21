package seqgen

import (
	"context"

	"github.com/emoss08/trenova/pkg/pulid"
)

type formatProvider struct{}

func NewFormatProvider() FormatProvider {
	return &formatProvider{}
}

func (p *formatProvider) GetFormat(
	ctx context.Context,
	sequenceType SequenceType,
	orgID, buID pulid.ID,
) (*Format, error) {
	return DefaultShipmentProNumberFormat(), nil
}
