package conversationschedulejobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/jobs/conversationschedulejobs/conversationschedulecontract"

	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
)

var _ conversationscheduleports.ConversationScheduleSyncer = (*conversationschedulecontract.Schedules)(
	nil,
)
