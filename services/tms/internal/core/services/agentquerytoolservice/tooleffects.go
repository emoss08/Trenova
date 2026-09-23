package agentquerytoolservice

import "github.com/emoss08/trenova/internal/core/domain/agent"

func (t *openPageTool) Effect() agent.ToolEffect { return agent.ToolEffectNavigate }

func (t *findInTrenovaTool) Effect() agent.ToolEffect { return agent.ToolEffectDiscover }

func (t *runReportTool) Effect() agent.ToolEffect { return agent.ToolEffectPresent }

func (t *compareReportRunsTool) Effect() agent.ToolEffect { return agent.ToolEffectPresent }

func (t *composeTableViewTool) Effect() agent.ToolEffect { return agent.ToolEffectPresent }

func (t *planDispatchTool) Effect() agent.ToolEffect { return agent.ToolEffectPresent }
