package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

var TexasJurisdictionID = pulid.MustNew("ifj_")

type FakeJurisdictions struct{}

func (FakeJurisdictions) GetJurisdiction(
	_ context.Context,
	id pulid.ID,
) (*ifta.Jurisdiction, error) {
	if id != TexasJurisdictionID {
		return nil, errortypes.NewNotFoundError("Jurisdiction not found")
	}

	return &ifta.Jurisdiction{ID: TexasJurisdictionID, Code: "TX", Name: "Texas"}, nil
}
