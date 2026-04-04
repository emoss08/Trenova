package shipmentservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *service) checkDuplicateBOLsWithControl(
	ctx context.Context,
	control *tenant.ShipmentControl,
	req *repositories.DuplicateBOLCheckRequest,
) error {
	if !control.CheckForDuplicateBOLs {
		return nil
	}

	duplicates, err := s.repo.CheckForDuplicateBOLs(ctx, req)
	if err != nil {
		return err
	}

	if len(duplicates) == 0 {
		return nil
	}

	proNumbers := make([]string, 0, len(duplicates))
	for _, duplicate := range duplicates {
		proNumbers = append(proNumbers, duplicate.ProNumber)
	}

	me := errortypes.NewMultiError()
	me.Add(
		"bol",
		errortypes.ErrInvalid,
		fmt.Sprintf(
			"BOL is already in use by shipment(s) with Pro Number(s): %s",
			strings.Join(proNumbers, ", "),
		),
	)

	if me.HasErrors() {
		return me
	}

	return nil
}
