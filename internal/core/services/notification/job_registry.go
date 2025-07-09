package notification

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
)

type JobNotificationConfig struct {
	EventType       notification.EventType
	Priority        notification.Priority
	FailurePriority notification.Priority
	TitleTemplate   string
	MessageTemplate string
	Tags            []string
}

// JobNotificationRegistry maps job types to their notification configurations
var JobNotificationRegistry = map[string]*JobNotificationConfig{
	"duplicate_shipment": {
		EventType:       notification.EventJobShipmentDuplicate,
		Priority:        notification.PriorityMedium,
		FailurePriority: notification.PriorityHigh,
		TitleTemplate:   "Shipment Duplication %s",
		MessageTemplate: "Shipment duplication job %s has %s: %s",
		Tags:            []string{"job", "shipment", "duplication"},
	},
	"pattern_analysis": {
		EventType:       notification.EventJobPatternAnalysis,
		Priority:        notification.PriorityMedium,
		FailurePriority: notification.PriorityHigh,
		TitleTemplate:   "Pattern Analysis %s",
		MessageTemplate: "Pattern analysis job %s has %s: %s",
		Tags:            []string{"job", "analysis", "patterns"},
	},
	"delay_shipment": {
		EventType:       notification.EventJobShipmentDelay,
		Priority:        notification.PriorityMedium,
		FailurePriority: notification.PriorityHigh,
		TitleTemplate:   "Shipment Delay %s",
		MessageTemplate: "Shipment delay job %s has %s: %s",
		Tags:            []string{"job", "shipment", "delay"},
	},
	"compliance_check": {
		EventType:       notification.EventJobComplianceCheck,
		Priority:        notification.PriorityHigh,
		FailurePriority: notification.PriorityCritical,
		TitleTemplate:   "Compliance Check %s",
		MessageTemplate: "Compliance check job %s has %s: %s",
		Tags:            []string{"job", "compliance", "safety"},
	},
	"billing_process": {
		EventType:       notification.EventJobBillingProcess,
		Priority:        notification.PriorityMedium,
		FailurePriority: notification.PriorityHigh,
		TitleTemplate:   "Billing Process %s",
		MessageTemplate: "Billing process job %s has %s: %s",
		Tags:            []string{"job", "billing", "finance"},
	},
}

// GetJobNotificationConfig returns the notification configuration for a job type
func GetJobNotificationConfig(jobType string) (*JobNotificationConfig, bool) {
	config, exists := JobNotificationRegistry[jobType]
	return config, exists
}

// RegisterJobType adds a new job type to the registry
func RegisterJobType(jobType string, config *JobNotificationConfig) {
	JobNotificationRegistry[jobType] = config
}

// GetEventType returns the event type for a job, with fallback for unknown types
func GetEventType(jobType string) notification.EventType {
	if config, exists := JobNotificationRegistry[jobType]; exists {
		return config.EventType
	}

	// TODO(wolfred): We should throw an error if the job type is unknown.

	return notification.EventJobUnknown
}

// GetPriority returns the priority for a job based on success status
func GetPriority(jobType string, success bool) notification.Priority {
	config, exists := JobNotificationRegistry[jobType]
	if !exists {
		if success {
			return notification.PriorityMedium
		}
		return notification.PriorityHigh
	}

	if success {
		return config.Priority
	}
	return config.FailurePriority
}

// GetTitle generates the title for a job notification
func GetTitle(jobType string, success bool) string {
	config, exists := JobNotificationRegistry[jobType]
	if !exists {
		if success {
			return "Job Completed"
		}
		return "Job Failed"
	}

	status := "Completed"
	if !success {
		status = "Failed"
	}

	return fmt.Sprintf(config.TitleTemplate, status)
}

// GetMessage generates the message for a job notification
func GetMessage(success bool, jobType, jobID, result string) string {
	config, exists := JobNotificationRegistry[jobType]
	if !exists {
		status := "completed successfully"
		if !success {
			status = "failed"
		}
		return fmt.Sprintf("Job %s (%s) has %s: %s", jobID, jobType, status, result)
	}

	status := "completed successfully"
	if !success {
		status = "failed"
	}

	return fmt.Sprintf(config.MessageTemplate, jobID, status, result)
}

// GetTags returns the tags for a job notification
func GetTags(jobType string) []string {
	config, exists := JobNotificationRegistry[jobType]
	if !exists {
		return []string{"job", jobType}
	}

	return config.Tags
}
