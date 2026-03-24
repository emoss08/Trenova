package sim

import "testing"

func TestFaultEngineEndpointSpecificity(t *testing.T) {
	t.Parallel()

	engine := NewFaultEngine("fault-seed")
	if _, err := engine.Add(&FaultRule{
		ID:      "rule-generic",
		Enabled: true,
		Target: FaultTarget{
			Kind:        "endpoint",
			Method:      "*",
			PathPattern: "/fleet/*",
		},
		Match:  FaultMatch{Profile: "*"},
		Effect: FaultEffect{StatusCode: 503},
		Rate:   1,
	}); err != nil {
		t.Fatalf("add generic rule: %v", err)
	}
	if _, err := engine.Add(&FaultRule{
		ID:      "rule-specific",
		Enabled: true,
		Target: FaultTarget{
			Kind:        "endpoint",
			Method:      "GET",
			PathPattern: "/fleet/vehicles/stats",
		},
		Match:  FaultMatch{Profile: "default"},
		Effect: FaultEffect{StatusCode: 429},
		Rate:   1,
	}); err != nil {
		t.Fatalf("add specific rule: %v", err)
	}

	decision, ok := engine.EvaluateEndpoint(
		"default",
		"GET",
		"/fleet/vehicles/stats",
		"GET|/fleet/vehicles/stats|",
	)
	if !ok {
		t.Fatal("expected endpoint fault decision")
	}
	if decision.Rule.ID != "rule-specific" {
		t.Fatalf("expected specific rule, got %q", decision.Rule.ID)
	}
}

func TestFaultEngineRateDeterministic(t *testing.T) {
	t.Parallel()

	engine := NewFaultEngine("fault-seed")
	if _, err := engine.Add(&FaultRule{
		ID:      "rule-rate",
		Enabled: true,
		Target: FaultTarget{
			Kind:        "endpoint",
			Method:      "GET",
			PathPattern: "/assets",
		},
		Effect: FaultEffect{StatusCode: 429},
		Rate:   0.4,
	}); err != nil {
		t.Fatalf("add rate rule: %v", err)
	}

	signature := "GET|/assets|ids=a1"
	first, firstOK := engine.EvaluateEndpoint("default", "GET", "/assets", signature)
	second, secondOK := engine.EvaluateEndpoint("default", "GET", "/assets", signature)
	if firstOK != secondOK {
		t.Fatal("fault rate decision must be deterministic")
	}
	if firstOK && first.Rule.ID != second.Rule.ID {
		t.Fatalf(
			"expected deterministic selected rule, got %q and %q",
			first.Rule.ID,
			second.Rule.ID,
		)
	}
}

func TestFaultEngineWebhookProfileMatch(t *testing.T) {
	t.Parallel()

	engine := NewFaultEngine("fault-seed")
	if _, err := engine.Add(&FaultRule{
		ID:      "webhook-speeding",
		Enabled: true,
		Target: FaultTarget{
			Kind:             "webhook",
			WebhookEventType: "VehicleSpeeding",
		},
		Match:  FaultMatch{Profile: "default"},
		Effect: FaultEffect{Drop: true},
		Rate:   1,
	}); err != nil {
		t.Fatalf("add webhook rule: %v", err)
	}

	_, ok := engine.EvaluateWebhook(
		"default",
		"VehicleSpeeding",
		"VehicleSpeeding|evt-1|wh-1",
	)
	if !ok {
		t.Fatal("expected webhook rule match for default profile")
	}

	_, mismatch := engine.EvaluateWebhook(
		"partial",
		"VehicleSpeeding",
		"VehicleSpeeding|evt-1|wh-1",
	)
	if mismatch {
		t.Fatal("did not expect webhook rule match for non-default profile")
	}
}
