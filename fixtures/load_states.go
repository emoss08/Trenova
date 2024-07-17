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

package fixtures

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

func loadUSStates(ctx context.Context, db *bun.DB) error {
	type stateData struct {
		Data struct {
			Name   string `json:"name"`
			Iso3   string `json:"iso3"`
			States []struct {
				Name      string `json:"name"`
				StateCode string `json:"state_code"`
			} `json:"states"`
		} `json:"data"`
	}

	url := "https://countriesnow.space/api/v0.1/countries/states"
	jsonData := map[string]string{"country": "United States"}
	jsonValue, _ := json.Marshal(jsonData)

	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonValue))
	request.Header.Set("Content-Type", "application/json")

	clientHTTP := &http.Client{}
	response, err := clientHTTP.Do(request)
	if err != nil {
		log.Printf("The HTTP request failed with error %s\n", err)
	}
	defer response.Body.Close()

	var data stateData
	if decodeErr := json.NewDecoder(response.Body).Decode(&data); decodeErr != nil {
		log.Printf("Error parsing the response data: %s\n", decodeErr)
	}

	for _, state := range data.Data.States {
		exists, stateErr := db.NewSelect().Model((*models.UsState)(nil)).Where("name = ?", state.Name).Exists(ctx)
		if stateErr != nil {
			log.Printf("Error checking if state exists: %s\n", stateErr)
			continue
		}

		if !exists {
			newState := &models.UsState{
				Name:         state.Name,
				Abbreviation: state.StateCode,
				CountryName:  data.Data.Name,
				CountryIso3:  data.Data.Iso3,
			}

			_, insertErr := db.NewInsert().Model(newState).Exec(ctx)

			if insertErr != nil {
				log.Printf("Error inserting state: %s\n", insertErr)
			}
		}
	}

	return nil
}
