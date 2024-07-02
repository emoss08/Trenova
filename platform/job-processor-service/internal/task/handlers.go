package task

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
)

func HandleSendReportTask(ctx context.Context, t *asynq.Task) error {
	var p ReportPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Add your report sending logic here
	log.Printf("Sending report with ID: %d", p.ReportID)
	return nil
}

func HandleNormalTask(ctx context.Context, t *asynq.Task) error {
	var p NormalTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Add your normal task logic here
	log.Printf("Processing normal task with ID: %d", p.TaskID)
	return nil
}

func HandleCleanupTask(ctx context.Context, t *asynq.Task) error {
	var p CleanupPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	// Add your cleanup task logic here
	log.Printf("Running cleanup task: %s", p.TaskName)
	return nil
}
