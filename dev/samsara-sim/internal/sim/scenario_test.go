package sim

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestScenarioEngineShouldFailDeterministic(t *testing.T) {
	t.Parallel()

	engine, err := NewScenarioEngine("seed-1", "degraded")
	if err != nil {
		t.Fatalf("new scenario engine: %v", err)
	}

	signature := "GET|/assets|limit=100"
	first := engine.ShouldFail("degraded", signature)
	second := engine.ShouldFail("degraded", signature)
	if first != second {
		t.Fatal("expected deterministic failure decision")
	}
}

func TestScenarioEngineHeaderOverride(t *testing.T) {
	t.Parallel()

	engine, err := NewScenarioEngine("seed-1", "default")
	if err != nil {
		t.Fatalf("new scenario engine: %v", err)
	}

	request := httptest.NewRequest("GET", "/assets", nil)
	request.Header.Set(HeaderProfileOverride, "sparse")

	profile := engine.ResolveProfile(request)
	if profile != "sparse" {
		t.Fatalf("expected sparse profile, got %s", profile)
	}
}

func TestScenarioEngineShouldOmitEventDropsMany(t *testing.T) {
	t.Parallel()

	engine, err := NewScenarioEngine("seed-1", "partial")
	if err != nil {
		t.Fatalf("new scenario engine: %v", err)
	}

	omitted := 0
	total := 50
	for idx := 0; idx < total; idx++ {
		payload := map[string]any{
			"id": fmt.Sprintf("evt-%03d", idx),
		}
		if engine.ShouldOmitEvent("partial", "RouteStopArrival", payload) {
			omitted++
		}
	}

	if omitted < 15 {
		t.Fatalf("expected many omitted events in partial profile, got %d of %d", omitted, total)
	}
}
