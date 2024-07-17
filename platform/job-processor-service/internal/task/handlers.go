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
