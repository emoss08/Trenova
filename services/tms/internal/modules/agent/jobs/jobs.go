package jobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/runstepledger"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentqualityjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/aiauditjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/aicorrectionjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/aifeedbackjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/aituneupjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/briefingjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/completionjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/conversationschedulejobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/decisioncommitjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/dispatchjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/importassistantjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/insightjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/reflectionjobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/retrievaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/watchtowerjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		aiauditjobs.Module,
		agentwaitjobs.Module,
		agentjobs.Module,
		reflectionjobs.Module,
		agentqualityjobs.Module,
		agentflow.Module,
		assistantjobs.Module,
		completionjobs.Module,
		importassistantjobs.Module,
		agentquerytoolservice.Module,
		agenttoolcatalog.Module,
		agenttoolpolicy.Module,
		agentruntime.Module,
		runstepledger.Module,
		conversationschedulejobs.Module,
		insightjobs.Module,
		watchtowerjobs.Module,
		briefingjobs.Module,
		aicorrectionjobs.Module,
		aifeedbackjobs.Module,
		aituneupjobs.Module,
		decisioncommitjobs.Module,
		retrievaljobs.Module,
		dispatchjobs.Module,
		agentjobs.WorkerModule,
		conversationschedulejobs.WorkerModule,
		agentqualityjobs.WorkerModule,
	)
}
