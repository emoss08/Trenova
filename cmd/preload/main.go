package main

// Send request to https://countriesnow.space/api/v0.1/countries/states
// With Json {"country":"United States"}
// Get the response and print it to the console

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/emoss08/trenova/database"
	_ "github.com/emoss08/trenova/ent/runtime"
	"github.com/emoss08/trenova/ent/usstate"
	tools "github.com/emoss08/trenova/util"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// StateData is the structure that holds the API response format.
type StateData struct {
	Data struct {
		Name   string `json:"name"`
		Iso3   string `json:"iso3"`
		States []struct {
			Name      string `json:"name"`
			StateCode string `json:"state_code"`
		} `json:"states"`
	} `json:"data"`
}

func main() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	// Initialize the database
	client := database.NewEntClient(tools.GetEnv("SERVER_DB_DSN", "host=localhost port=5432 user=postgres password=postgres dbname=trenova sslmode=disable"))

	defer client.Close()

	if os.Getenv("ENV") == "production" {
		log.Panic("Cannot run seeder in production environment")
	}

	ctx := context.Background()

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

	var data StateData
	if decodeErr := json.NewDecoder(response.Body).Decode(&data); decodeErr != nil {
		log.Printf("Error parsing the response data: %s\n", decodeErr)
	}

	for _, state := range data.Data.States {
		exists, stateErr := client.UsState.Query().Where(usstate.Abbreviation(state.StateCode)).Exist(ctx)
		if stateErr != nil {
			log.Printf("Error checking for state existence: %s\n", stateErr)
			continue
		}

		if !exists {
			_, saveErr := client.UsState.
				Create().
				SetName(state.Name).
				SetAbbreviation(state.StateCode).
				SetCountryName(data.Data.Name).
				SetCountryIso3(data.Data.Iso3).
				Save(ctx)
			if saveErr != nil {
				log.Printf("Failed to create state: %s\n", saveErr)
				continue
			}
			log.Printf("State %s added\n", state.Name)
		}
	}

	log.Println("States preloading completed.")
}
