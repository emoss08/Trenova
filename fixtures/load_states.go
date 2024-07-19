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
