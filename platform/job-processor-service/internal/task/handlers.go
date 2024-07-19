// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
