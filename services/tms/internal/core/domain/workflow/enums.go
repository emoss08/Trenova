package workflow

import (
	"errors"
)

type TriggerType string

const (
	TriggerTypeManual    = TriggerType("Manual")
	TriggerTypeScheduled = TriggerType("Scheduled")
	TriggerTypeEvent     = TriggerType("Event")
)

func TriggerTypeFromString(s string) (TriggerType, error) {
	switch s {
	case "Manual":
		return TriggerTypeManual, nil
	case "Scheduled":
		return TriggerTypeScheduled, nil
	case "Event":
		return TriggerTypeEvent, nil
	default:
		return "", errors.New("invalid trigger type")
	}
}

func (t TriggerType) String() string {
	return string(t)
}

func (t TriggerType) IsValid() bool {
	switch t {
	case TriggerTypeManual, TriggerTypeScheduled, TriggerTypeEvent:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusActive   = Status("Active")
	StatusInactive = Status("Inactive")
	StatusDraft    = Status("Draft")
)

func StatusFromString(s string) (Status, error) {
	switch s {
	case "Active":
		return StatusActive, nil
	case "Inactive":
		return StatusInactive, nil
	case "Draft":
		return StatusDraft, nil
	default:
		return "", errors.New("invalid workflow status")
	}
}

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusDraft:
		return true
	default:
		return false
	}
}

type InstanceStatus string

const (
	InstanceStatusRunning   = InstanceStatus("Running")
	InstanceStatusCompleted = InstanceStatus("Completed")
	InstanceStatusFailed    = InstanceStatus("Failed")
	InstanceStatusCancelled = InstanceStatus("Cancelled")
	InstanceStatusPaused    = InstanceStatus("Paused")
)

func InstanceStatusFromString(s string) (InstanceStatus, error) {
	switch s {
	case "Running":
		return InstanceStatusRunning, nil
	case "Completed":
		return InstanceStatusCompleted, nil
	case "Failed":
		return InstanceStatusFailed, nil
	case "Cancelled":
		return InstanceStatusCancelled, nil
	case "Paused":
		return InstanceStatusPaused, nil
	default:
		return "", errors.New("invalid instance status")
	}
}

func (i InstanceStatus) String() string {
	return string(i)
}

func (i InstanceStatus) IsValid() bool {
	switch i {
	case InstanceStatusRunning,
		InstanceStatusCompleted,
		InstanceStatusFailed,
		InstanceStatusCancelled,
		InstanceStatusPaused:
		return true
	default:
		return false
	}
}

func (i InstanceStatus) IsTerminal() bool {
	switch i {
	case InstanceStatusCompleted, InstanceStatusFailed, InstanceStatusCancelled:
		return true
	default:
		return false
	}
}

type NodeType string

const (
	NodeTypeTrigger      = NodeType("Trigger")
	NodeTypeEntityUpdate = NodeType("EntityUpdate")
	NodeTypeCondition    = NodeType("Condition")
)

func NodeTypeFromString(s string) (NodeType, error) {
	switch s {
	case "Trigger":
		return NodeTypeTrigger, nil
	case "EntityUpdate":
		return NodeTypeEntityUpdate, nil
	case "Condition":
		return NodeTypeCondition, nil
	default:
		return "", errors.New("invalid node type")
	}
}

func (n NodeType) String() string {
	return string(n)
}

func (n NodeType) IsValid() bool {
	switch n {
	case NodeTypeTrigger, NodeTypeEntityUpdate, NodeTypeCondition:
		return true
	default:
		return false
	}
}

type NodeExecutionStatus string

const (
	NodeExecutionStatusPending   = NodeExecutionStatus("Pending")
	NodeExecutionStatusRunning   = NodeExecutionStatus("Running")
	NodeExecutionStatusCompleted = NodeExecutionStatus("Completed")
	NodeExecutionStatusFailed    = NodeExecutionStatus("Failed")
	NodeExecutionStatusSkipped   = NodeExecutionStatus("Skipped")
)

func NodeExecutionStatusFromString(s string) (NodeExecutionStatus, error) {
	switch s {
	case "Pending":
		return NodeExecutionStatusPending, nil
	case "Running":
		return NodeExecutionStatusRunning, nil
	case "Completed":
		return NodeExecutionStatusCompleted, nil
	case "Failed":
		return NodeExecutionStatusFailed, nil
	case "Skipped":
		return NodeExecutionStatusSkipped, nil
	default:
		return "", errors.New("invalid node execution status")
	}
}

func (n NodeExecutionStatus) String() string {
	return string(n)
}

func (n NodeExecutionStatus) IsValid() bool {
	switch n {
	case NodeExecutionStatusPending,
		NodeExecutionStatusRunning,
		NodeExecutionStatusCompleted,
		NodeExecutionStatusFailed,
		NodeExecutionStatusSkipped:
		return true
	default:
		return false
	}
}

func (n NodeExecutionStatus) IsTerminal() bool {
	switch n {
	case NodeExecutionStatusCompleted, NodeExecutionStatusFailed, NodeExecutionStatusSkipped:
		return true
	default:
		return false
	}
}

type ExecutionMode string

const (
	ExecutionModeManual    = ExecutionMode("Manual")
	ExecutionModeScheduled = ExecutionMode("Scheduled")
	ExecutionModeEvent     = ExecutionMode("Event")
)

func ExecutionModeFromString(s string) (ExecutionMode, error) {
	switch s {
	case "Manual":
		return ExecutionModeManual, nil
	case "Scheduled":
		return ExecutionModeScheduled, nil
	case "Event":
		return ExecutionModeEvent, nil
	default:
		return "", errors.New("invalid execution mode")
	}
}

func (e ExecutionMode) String() string {
	return string(e)
}

func (e ExecutionMode) IsValid() bool {
	switch e {
	case ExecutionModeManual, ExecutionModeScheduled, ExecutionModeEvent:
		return true
	default:
		return false
	}
}

type VersionStatus string

const (
	VersionStatusDraft     = VersionStatus("Draft")
	VersionStatusPublished = VersionStatus("Published")
	VersionStatusArchived  = VersionStatus("Archived")
)

func VersionStatusFromString(s string) (VersionStatus, error) {
	switch s {
	case "Draft":
		return VersionStatusDraft, nil
	case "Published":
		return VersionStatusPublished, nil
	case "Archived":
		return VersionStatusArchived, nil
	default:
		return "", errors.New("invalid version status")
	}
}

func (v VersionStatus) String() string {
	return string(v)
}

func (v VersionStatus) IsValid() bool {
	switch v {
	case VersionStatusDraft, VersionStatusPublished, VersionStatusArchived:
		return true
	default:
		return false
	}
}

func (v VersionStatus) IsDraft() bool {
	return v == VersionStatusDraft
}

func (v VersionStatus) IsPublished() bool {
	return v == VersionStatusPublished
}

func (v VersionStatus) IsArchived() bool {
	return v == VersionStatusArchived
}
