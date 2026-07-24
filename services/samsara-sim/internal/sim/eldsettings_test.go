package sim

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

const defaultFixturePath = "../../config/fixtures/default.json"

func loadDefaultFixtureStore(t *testing.T) *Store {
	t.Helper()

	store, err := NewStoreFromFixtureFile(defaultFixturePath)
	if err != nil {
		t.Fatalf("load default fixture: %v", err)
	}
	return store
}

func driverRulesets(t *testing.T, driver Record) []map[string]any {
	t.Helper()

	rawRulesets, ok := nestedAny(driver, "eldSettings", "rulesets").([]any)
	if !ok || len(rawRulesets) == 0 {
		t.Fatalf("expected eldSettings.rulesets on driver %q", recordID(driver))
	}
	out := make([]map[string]any, 0, len(rawRulesets))
	for _, raw := range rawRulesets {
		ruleset, isMap := anyAsMap(raw)
		if !isMap {
			t.Fatalf("expected ruleset object on driver %q", recordID(driver))
		}
		out = append(out, ruleset)
	}
	return out
}

func TestDefaultFixtureDriverEldSettingsRoster(t *testing.T) {
	t.Parallel()

	store := loadDefaultFixtureStore(t)
	drivers, err := store.List(ResourceDrivers)
	if err != nil {
		t.Fatalf("list drivers: %v", err)
	}
	if len(drivers) == 0 {
		t.Fatal("expected drivers in default fixture")
	}

	sixtyHourCycle := 0
	texasIntrastateShift := 0
	defaultRulesets := 0
	for _, driver := range drivers {
		rulesets := driverRulesets(t, driver)
		ruleset := Record(rulesets[0])

		cycle := stringValue(ruleset, "cycle")
		shift := stringValue(ruleset, "shift")
		if stringValue(ruleset, "jurisdiction") != "TX" {
			t.Fatalf(
				"expected TX jurisdiction on driver %q, got %q",
				recordID(driver),
				stringValue(ruleset, "jurisdiction"),
			)
		}
		if stringValue(ruleset, "restart") != "34-hour Restart" {
			t.Fatalf("unexpected restart %q on driver %q", stringValue(ruleset, "restart"), recordID(driver))
		}
		if stringValue(ruleset, "break") != "Property (off-duty/sleeper)" {
			t.Fatalf("unexpected break %q on driver %q", stringValue(ruleset, "break"), recordID(driver))
		}

		switch cycle {
		case "USA 70 hour / 8 day":
		case "USA 60 hour / 7 day":
			sixtyHourCycle++
		default:
			t.Fatalf("unexpected cycle %q on driver %q", cycle, recordID(driver))
		}
		switch shift {
		case "US Interstate Property":
		case "Texas Intrastate":
			texasIntrastateShift++
		default:
			t.Fatalf("unexpected shift %q on driver %q", shift, recordID(driver))
		}
		if cycle == "USA 70 hour / 8 day" && shift == "US Interstate Property" {
			defaultRulesets++
		}
	}

	if sixtyHourCycle != 2 {
		t.Fatalf("expected 2 drivers on USA 60 hour / 7 day, got %d", sixtyHourCycle)
	}
	if texasIntrastateShift != 1 {
		t.Fatalf("expected 1 driver on Texas Intrastate shift, got %d", texasIntrastateShift)
	}
	if defaultRulesets != len(drivers)-3 {
		t.Fatalf(
			"expected %d drivers on the default ruleset, got %d",
			len(drivers)-3,
			defaultRulesets,
		)
	}
}

func TestServerFleetDriversExposeEldSettings(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Auth.Tokens = []string{"dev-samsara-token"}
	cfg.Webhooks.Enabled = false

	scenarios, err := NewScenarioEngine("eld-settings-seed", "default")
	if err != nil {
		t.Fatalf("failed to initialize scenario engine: %v", err)
	}
	srv := NewServer(&cfg, loadDefaultFixtureStore(t), scenarios, nil, nil)

	response := performAuthorizedRequest(srv, http.MethodGet, "/fleet/drivers")
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 from /fleet/drivers, got %d", response.Code)
	}
	records := mustReadDataRecords(t, response.Body.Bytes())
	if len(records) == 0 {
		t.Fatal("expected driver records")
	}
	for _, record := range records {
		rulesets := driverRulesets(t, record)
		for _, ruleset := range rulesets {
			for _, key := range []string{"cycle", "shift", "restart", "break", "jurisdiction"} {
				if stringValue(Record(ruleset), key) == "" {
					t.Fatalf("expected %q on ruleset for driver %q", key, recordID(record))
				}
			}
		}
	}
}
