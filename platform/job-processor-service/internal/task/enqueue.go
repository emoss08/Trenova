// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
