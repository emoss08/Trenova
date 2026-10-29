package importdraftquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
)

var (
	importRequiredFields = agenttoolschema.Source(
		"importDraft.requiredField",
		pagedraft.AllRequiredFields(),
	)
)
