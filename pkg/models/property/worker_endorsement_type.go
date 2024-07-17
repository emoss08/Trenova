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

package property

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type WorkerEndorsement string

const (
	WorkerEndorsementNone         = WorkerEndorsement("None")
	WorkerEndorsementTanker       = WorkerEndorsement("Tanker")
	WorkerEndorsementHazmat       = WorkerEndorsement("Hazmat")
	WorkerEndorsementTankerHazmat = WorkerEndorsement("TankerHazmat")
)

func (o WorkerEndorsement) String() string {
	return string(o)
}

func (WorkerEndorsement) Values() []string {
	return []string{"None", "Tanker", "Hazmat", "TankerHazmat"}
}

func (o WorkerEndorsement) Value() (driver.Value, error) {
	return string(o), nil
}

func (o *WorkerEndorsement) Scan(value any) error {
	if value == nil {
		return errors.New("WorkerEndorsement: expected a value, got nil")
	}
	switch v := value.(type) {
	case string:
		*o = WorkerEndorsement(v)
	case []byte:
		*o = WorkerEndorsement(string(v))
	default:
		return fmt.Errorf("WorkerEndorsement: cannot can type %T into WorkerEndorsement", value)
	}
	return nil
}

func GetWorkerEndorsementList() []any {
	values := WorkerEndorsement("").Values()
	interfaces := make([]any, len(values))
	for i, v := range values {
		interfaces[i] = v
	}

	return interfaces
}
