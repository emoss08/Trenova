package task

import (
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

type TaskEnqueuer struct {
	Client *asynq.Client
	Logger *zerolog.Logger
}

func NewTaskEnqueuer(client *asynq.Client, logger *zerolog.Logger) *TaskEnqueuer {
	return &TaskEnqueuer{Client: client, Logger: logger}
}

func (e *TaskEnqueuer) EnqueueSendReportTask(reportID int) error {
	payload, err := json.Marshal(ReportPayload{ReportID: reportID})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeSendReport, payload)
	if _, err := e.Client.Enqueue(task); err != nil {
		return err
	}

	log.Printf("Enqueued send report task with ID: %d", reportID)
	return nil
}

func (e *TaskEnqueuer) EnqueueNormalTask(taskID int) error {
	payload, err := json.Marshal(NormalTaskPayload{TaskID: taskID})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeNormalTask, payload)
	if _, err := e.Client.Enqueue(task); err != nil {
		return err
	}

	log.Printf("Enqueued normal task with ID: %d", taskID)
	return nil
}

func (e *TaskEnqueuer) EnqueueCleanupTask(taskName string) error {
	payload, err := json.Marshal(CleanupPayload{TaskName: taskName})
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeCleanup, payload)
	if _, err := e.Client.Enqueue(task); err != nil {
		return err
	}

	log.Printf("Enqueued cleanup task: %s", taskName)
	return nil
}
