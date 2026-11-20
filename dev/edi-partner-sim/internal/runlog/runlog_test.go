package runlog

import (
	"errors"
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
}

func TestScenariosAttachToTheirKeysAndTally(t *testing.T) {
	t.Parallel()

	var notices []string
	s := New(2, fixedNow, func(n Notice) { notices = append(notices, n.Kind) })
	run, err := s.Start(Start{Kind: KindRun, Number: 7})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if err = s.PutScenario(run.ID, &Scenario{Partner: "acme", Slug: "held",
		Status: StatusRunning, Steps: []Step{{Index: 0, Status: StatusPending}}}); err != nil {
		t.Fatalf("put scenario: %v", err)
	}
	if err = s.PutStep(run.ID, &StepPost{Partner: "acme", Scenario: "held", Step: Step{
		Index:   0,
		Status:  StatusPassed,
		Matches: []Match{{MessageID: "acme-000001", Key: "K1"}},
		Events:  []Event{{Key: "K1", Label: "Flow held", Tone: ToneHold}},
	}}); err != nil {
		t.Fatalf("put step: %v", err)
	}
	attached, ok := s.For("acme", "K1")
	if !ok || attached.Scenario.Slug != "held" || !attached.Scenario.Waiting {
		t.Fatalf("attached %+v %v", attached, ok)
	}
	if current, _ := s.Current(); current.Tally.Waiting != 1 ||
		current.Partners["acme"].Waiting != 1 {
		t.Fatalf("tally %+v", current)
	}
	if err = s.PutStep(run.ID, &StepPost{Partner: "acme", Scenario: "held", Step: Step{
		Index:  1,
		Status: StatusPassed,
		Events: []Event{{Key: "K1", Label: "Planner released"}},
	}}); err != nil {
		t.Fatalf("put step: %v", err)
	}
	if attached, _ = s.For("acme", "K1"); attached.Scenario.Waiting {
		t.Fatal("a release did not end the wait")
	}
	if err = s.PutScenario(run.ID, &Scenario{Partner: "acme", Slug: "held",
		Status: StatusFailed}); err != nil {
		t.Fatalf("finish the scenario: %v", err)
	}
	if attached, _ = s.For("acme", "K1"); len(attached.Scenario.Steps) != 2 ||
		attached.Scenario.Status != StatusFailed {
		t.Fatalf("a header update lost the steps: %+v", attached.Scenario)
	}
	if _, err = s.Finish(run.ID, Finish{Status: StatusFailed}); err != nil {
		t.Fatalf("finish: %v", err)
	}
	want := []string{"run.started", "run.scenario", "run.step", "run.step", "run.scenario",
		"run.finished"}
	if len(notices) != len(want) {
		t.Fatalf("notices %v", notices)
	}
	for index := range want {
		if notices[index] != want[index] {
			t.Fatalf("notices %v, want %v", notices, want)
		}
	}
}

func TestTheStoreKeepsTheNewestRunsAndRestores(t *testing.T) {
	t.Parallel()

	s := New(2, fixedNow, nil)
	var ids []string
	for number := range uint64(3) {
		run, _ := s.Start(Start{Kind: KindSoak, Number: number + 1})
		ids = append(ids, run.ID)
		_ = s.PutScenario(run.ID, &Scenario{Partner: "acme", Slug: "soak-1",
			Status: StatusPassed, Keys: []string{"K" + run.ID}})
	}
	if _, ok := s.Get(ids[0]); ok {
		t.Fatal("the oldest run was kept past the capacity")
	}
	if _, ok := s.For("acme", "K"+ids[0]); ok {
		t.Fatal("an evicted run still answers for its keys")
	}
	restored := New(2, fixedNow, nil)
	restored.Restore(s.Snapshot())
	if attached, ok := restored.For("acme", "K"+ids[2]); !ok || attached.Run.ID != ids[2] {
		t.Fatalf("restored %+v %v", attached, ok)
	}
	next, _ := restored.Start(Start{Kind: KindRun})
	if next.ID == ids[2] {
		t.Fatal("the restored store reuses run ids")
	}
	restored.Reset("acme")
	if _, ok := restored.For("acme", "K"+ids[2]); ok {
		t.Fatal("the reset kept the partner's keys")
	}
}

func TestBadProgressIsRefused(t *testing.T) {
	t.Parallel()

	s := New(0, fixedNow, nil)
	if _, err := s.Start(Start{Kind: "walk"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("an unknown kind: %v", err)
	}
	run, _ := s.Start(Start{Kind: KindRun})
	for _, c := range []struct {
		err  error
		want error
	}{
		{s.PutScenario(run.ID, &Scenario{Slug: "x", Status: StatusRunning}), ErrInvalid},
		{s.PutScenario("r99", &Scenario{Partner: "a", Slug: "x", Status: StatusRunning}),
			ErrNoRun},
		{s.PutStep(run.ID, &StepPost{Partner: "a", Scenario: "x",
			Step: Step{Status: StatusPassed}}), ErrNoScenario},
		{s.PutStep(run.ID, &StepPost{Partner: "a", Scenario: "x",
			Step: Step{Index: MaxSteps, Status: StatusPassed}}), ErrInvalid},
	} {
		if !errors.Is(c.err, c.want) {
			t.Errorf("got %v, want %v", c.err, c.want)
		}
	}
}
