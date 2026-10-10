package agentflowcontract

import (
	"time"
)

// DrainWindow is how long a finished run waits for its reader to say it
// has the last event. A reader that is there says so within a poll or
// two; one that is not is not coming, and the run closes without it.
const DrainWindow = 15 * time.Second

const PriorityInteractive = 1

const PriorityOneShot = 2

const PriorityBackground = 3

// SteerSignal carries something the person said to a turn while it works.
// The turn reads it at its next step.
const SteerSignal = "assistant-turn-steer"
