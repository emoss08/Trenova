package volumegen

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol/documentcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
)

func (g *Generator) DispatchControl(
	ctx context.Context,
) (*dispatchcontrol.DispatchControl, error) {
	control, err := FirstOrNil(ctx, g.DB, dispatchcontroldb.DispatchControls.Query())
	if err != nil {
		return nil, fmt.Errorf("load dispatch control: %w", err)
	}

	return control, nil
}

func (g *Generator) DocumentControl(
	ctx context.Context,
) (*documentcontrol.DocumentControl, error) {
	control, err := FirstOrNil(ctx, g.DB, documentcontroldb.DocumentControls.Query())
	if err != nil {
		return nil, fmt.Errorf("load document control: %w", err)
	}

	return control, nil
}

func ServiceFailureGraceMinutes(control *dispatchcontrol.DispatchControl) int {
	if control == nil || control.ServiceFailureGracePeriod == nil ||
		*control.ServiceFailureGracePeriod <= 0 {
		return dispatchcontrol.DefaultServiceFailureGracePeriod
	}

	return *control.ServiceFailureGracePeriod
}
