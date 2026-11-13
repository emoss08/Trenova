package watchtower

import "github.com/emoss08/trenova/shared/pulid"

// HasAgent reports whether the item was handed to an agent: it names the
// agent and the run it was handed to.
func (i *Item) HasAgent() bool {
	return i.AgentDefinitionID.IsNotNil() && i.AgentRunID.IsNotNil()
}

// HandTo notes on the item which agent it was handed to, the run that has
// it and who handed it over, and puts the item in progress.
func (i *Item) HandTo(definitionID, runID, handedBy pulid.ID, now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	i.AgentDefinitionID = definitionID
	i.AgentRunID = runID
	i.AgentHandedByID = handedBy
	i.AgentHandedAt = &now
	if i.Status != StatusInProgress {
		i.Status = StatusInProgress
		i.SnoozedUntil = nil
		if i.AcknowledgedAt == nil {
			i.AcknowledgedAt = &now
		}
	}
	return nil
}
