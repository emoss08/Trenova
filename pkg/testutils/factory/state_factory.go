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

package factory

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

type StateFactory struct {
	db *bun.DB
}

func NewStateFactory(db *bun.DB) *StateFactory {
	return &StateFactory{db: db}
}

func (f *StateFactory) CreateUSState(ctx context.Context) (*models.UsState, error) {
	state := &models.UsState{
		Name:         "North Carolina",
		Abbreviation: "NC",
		CountryName:  "United States",
		CountryIso3:  "USA",
	}

	_, err := f.db.NewInsert().Model(state).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return state, nil
}
