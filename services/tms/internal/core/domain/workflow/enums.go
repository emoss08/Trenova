package workflow

import (
	"database/sql/driver"
	"fmt"
)

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"    // Being edited, not published
	WorkflowStatusActive   WorkflowStatus = "active"   // Published and can be triggered
	WorkflowStatusInactive WorkflowStatus = "inactive" // Published but disabled
	WorkflowStatusArchived WorkflowStatus = "archived" // Archived, cannot be activated
)

func (s WorkflowStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *WorkflowStatus) Scan(value any) error {
	if value == nil {
		*s = WorkflowStatusDraft
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan WorkflowStatus: %v", value)
	}

	*s = WorkflowStatus(str)
	return nil
}

func (s WorkflowStatus) String() string {
	return string(s)
}

// TriggerType represents the type of trigger that starts a workflow
type TriggerType string

const (
	TriggerTypeManual           TriggerType = "manual"            // Manually triggered
	TriggerTypeScheduled        TriggerType = "scheduled"         // Time-based (cron)
	TriggerTypeShipmentStatus   TriggerType = "shipment_status"   // Shipment status change
	TriggerTypeDocumentUploaded TriggerType = "document_uploaded" // Document upload event
	TriggerTypeEntityCreated    TriggerType = "entity_created"    // Entity creation event
	TriggerTypeEntityUpdated    TriggerType = "entity_updated"    // Entity update event
	TriggerTypeWebhook          TriggerType = "webhook"           // External webhook
)

func (t TriggerType) Value() (driver.Value, error) {
	return string(t), nil
}

func (t *TriggerType) Scan(value any) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan TriggerType: %v", value)
	}

	*t = TriggerType(str)
	return nil
}

func (t TriggerType) String() string {
	return string(t)
}

// NodeType represents the type of a workflow node
type NodeType string

const (
	NodeTypeTrigger   NodeType = "trigger"   // Trigger node (start)
	NodeTypeAction    NodeType = "action"    // Action node
	NodeTypeCondition NodeType = "condition" // If/else condition
	NodeTypeLoop      NodeType = "loop"      // Loop/iteration
	NodeTypeDelay     NodeType = "delay"     // Delay/wait
	NodeTypeEnd       NodeType = "end"       // End node
)

func (n NodeType) Value() (driver.Value, error) {
	return string(n), nil
}

func (n *NodeType) Scan(value any) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan NodeType: %v", value)
	}

	*n = NodeType(str)
	return nil
}

func (n NodeType) String() string {
	return string(n)
}

// ActionType represents the type of action a workflow node can perform
type ActionType string

const (
	// Shipment actions
	ActionTypeShipmentUpdateStatus   ActionType = "shipment_update_status"
	ActionTypeShipmentAssignCarrier  ActionType = "shipment_assign_carrier"
	ActionTypeShipmentAssignDriver   ActionType = "shipment_assign_driver"
	ActionTypeShipmentUpdateField    ActionType = "shipment_update_field"

	// Billing actions
	ActionTypeBillingValidateRequirements ActionType = "billing_validate_requirements"
	ActionTypeBillingTransferToQueue      ActionType = "billing_transfer_to_queue"
	ActionTypeBillingGenerateInvoice      ActionType = "billing_generate_invoice"
	ActionTypeBillingSendInvoice          ActionType = "billing_send_invoice"

	// Document actions
	ActionTypeDocumentValidateCompleteness ActionType = "document_validate_completeness"
	ActionTypeDocumentRequestMissing       ActionType = "document_request_missing"
	ActionTypeDocumentGenerate             ActionType = "document_generate"

	// Notification actions
	ActionTypeNotificationSendEmail   ActionType = "notification_send_email"
	ActionTypeNotificationSendSMS     ActionType = "notification_send_sms"
	ActionTypeNotificationSendWebhook ActionType = "notification_send_webhook"
	ActionTypeNotificationSendPush    ActionType = "notification_send_push"

	// Data actions
	ActionTypeDataTransform     ActionType = "data_transform"
	ActionTypeDataAPICall       ActionType = "data_api_call"
	ActionTypeDataDatabaseQuery ActionType = "data_database_query"

	// Flow control actions
	ActionTypeFlowApprovalRequest    ActionType = "flow_approval_request"
	ActionTypeFlowWaitForEvent       ActionType = "flow_wait_for_event"
	ActionTypeFlowParallelExecution  ActionType = "flow_parallel_execution"
)

func (a ActionType) Value() (driver.Value, error) {
	return string(a), nil
}

func (a *ActionType) Scan(value any) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ActionType: %v", value)
	}

	*a = ActionType(str)
	return nil
}

func (a ActionType) String() string {
	return string(a)
}

// ExecutionStatus represents the status of a workflow execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"   // Queued for execution
	ExecutionStatusRunning   ExecutionStatus = "running"   // Currently executing
	ExecutionStatusPaused    ExecutionStatus = "paused"    // Paused by user
	ExecutionStatusCompleted ExecutionStatus = "completed" // Completed successfully
	ExecutionStatusFailed    ExecutionStatus = "failed"    // Failed with error
	ExecutionStatusCanceled  ExecutionStatus = "canceled"  // Canceled by user
	ExecutionStatusTimeout   ExecutionStatus = "timeout"   // Execution timeout
)

func (e ExecutionStatus) Value() (driver.Value, error) {
	return string(e), nil
}

func (e *ExecutionStatus) Scan(value any) error {
	if value == nil {
		*e = ExecutionStatusPending
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ExecutionStatus: %v", value)
	}

	*e = ExecutionStatus(str)
	return nil
}

func (e ExecutionStatus) String() string {
	return string(e)
}

// StepStatus represents the status of a workflow execution step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusRetrying  StepStatus = "retrying"
)

func (s StepStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *StepStatus) Scan(value any) error {
	if value == nil {
		*s = StepStatusPending
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan StepStatus: %v", value)
	}

	*s = StepStatus(str)
	return nil
}

func (s StepStatus) String() string {
	return string(s)
}
