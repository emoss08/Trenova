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
	// _ Override flags for custom message formatting
	UseCustomTitle   bool
	UseCustomMessage bool
}

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
		EventType:        notification.EventJobShipmentDelay,
		Priority:         notification.PriorityMedium,
		FailurePriority:  notification.PriorityHigh,
		TitleTemplate:    "Shipment Delay Notice!",
		MessageTemplate:  "%s",
		Tags:             []string{"job", "shipment", "delay"},
		UseCustomTitle:   true,
		UseCustomMessage: true,
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

func GetJobNotificationConfig(jobType string) (*JobNotificationConfig, bool) {
	config, exists := JobNotificationRegistry[jobType]
	return config, exists
}

func RegisterJobType(jobType string, config *JobNotificationConfig) {
	JobNotificationRegistry[jobType] = config
}

func GetEventType(jobType string) notification.EventType {
	if config, exists := JobNotificationRegistry[jobType]; exists {
		return config.EventType
	}

	// TODO(wolfred): We should throw an error if the job type is unknown.

	return notification.EventJobUnknown
}

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

func GetTitle(jobType string, success bool) string {
	config, exists := JobNotificationRegistry[jobType]
	if !exists {
		if success {
			return "Job Completed"
		}
		return "Job Failed"
	}

	// _ Check if custom title is enabled
	if config.UseCustomTitle {
		return config.TitleTemplate
	}

	status := "Completed"
	if !success {
		status = "Failed"
	}

	return fmt.Sprintf(config.TitleTemplate, status)
}

func GetMessage(success bool, jobType, jobID, result string) string {
	config, exists := JobNotificationRegistry[jobType]
	if !exists {
		status := "completed successfully"
		if !success {
			status = "failed"
		}
		return fmt.Sprintf("Job %s (%s) has %s: %s", jobID, jobType, status, result)
	}

	// _ Check if custom message is enabled
	if config.UseCustomMessage {
		return fmt.Sprintf(config.MessageTemplate, result)
	}

	status := "completed successfully"
	if !success {
		status = "failed"
	}

	return fmt.Sprintf(config.MessageTemplate, jobID, status, result)
}

func GetTags(jobType string) []string {
	config, exists := JobNotificationRegistry[jobType]
	if !exists {
		return []string{"job", jobType}
	}

	return config.Tags
}
