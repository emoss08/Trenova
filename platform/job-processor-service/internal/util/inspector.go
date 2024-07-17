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

package util

import (
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

func InspectQueue(redisAddr string) error {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr})
	tasks, err := inspector.ListPendingTasks("default")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to list pending tasks")
		return err
	}
	for _, task := range tasks {
		log.Info().Msgf("Pending task: ID=%s, Type=%s, Payload=%s, Queue=%s", task.ID, task.Type, string(task.Payload), task.Queue)
	}

	return nil
}
