package formulaquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
)

var (
	formulaKinds = agenttoolschema.Source("formula.kind", formulaassistantservice.AllKinds())
)
