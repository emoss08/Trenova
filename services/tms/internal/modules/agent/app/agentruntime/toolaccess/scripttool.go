package toolaccess

import (
	"context"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"go.uber.org/zap"
)

// ScriptsOffered reports whether a turn opening now may be shown the script
// tool: some provider that may answer it has scripts on. Only a turn left to
// the providers' settings asks; a failed read offers nothing.
func (s *Access) ScriptsOffered(ctx context.Context, req *agentruntimeports.RunRequest) bool {
	if req.CodeMode != codemode.OverrideDefault || s.scripts == nil || req.Actor == nil {
		return false
	}
	offered, err := s.scripts.ScriptsOffered(ctx, req.Actor.TenantInfo())
	if err != nil {
		s.logger.Warn("could not read the providers' script settings; scripts are not offered",
			zap.Error(err))
		return false
	}

	return offered
}

// WithoutScripts takes the tool that runs a script out of a turn the code-mode
// gate keeps it from, so the model is never shown a tool it may not call.
func (s *Access) WithoutScripts(
	definition *agentdefinition.Definition,
	held []string,
	beyond map[string]agent.AutonomyTier,
) (kept []string, reached map[string]agent.AutonomyTier) {
	if held == nil {
		held = s.HeldTools(definition)
	}
	kept = slices.DeleteFunc(slices.Clone(held), isScriptRunner)
	reached = beyond
	if _, found := beyond[codemode.ToolRunScript]; found {
		reached = maps.Clone(beyond)
		delete(reached, codemode.ToolRunScript)
	}

	return kept, reached
}

func isScriptRunner(name string) bool {
	return name == codemode.ToolRunScript
}
