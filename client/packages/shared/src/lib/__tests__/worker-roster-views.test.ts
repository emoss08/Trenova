import { describe, expect, it } from "vitest";
import {
  activeRosterView,
  EXPIRING_SOON_DAYS,
  rosterViews,
  type RosterViewId,
} from "../worker-roster-views";

const NOW = 1_800_000_000;

function viewById(id: RosterViewId) {
  const view = rosterViews(NOW).find((candidate) => candidate.id === id);
  if (!view) throw new Error(`no view ${id}`);
  return view;
}

describe("rosterViews", () => {
  it("offers the questions HR opens the roster to answer", () => {
    expect(rosterViews(NOW).map((view) => view.id)).toEqual([
      "all",
      "non-compliant",
      "training-overdue",
      "prohibited",
      "at-risk",
      "expiring-soon",
    ]);
  });

  it("clears every filter for everyone", () => {
    expect(viewById("all").filters).toEqual([]);
  });

  // "Overdue" in the office's sense covers anything a worker cannot currently
  // show, not just the one enum value that happens to be spelled Overdue.
  it("treats expired, failed and never-assigned training as overdue too", () => {
    const filter = viewById("training-overdue").filters[0];
    expect(filter.field).toBe("profile.trainingHealth");
    expect(filter.operator).toBe("in");
    expect(filter.value).toEqual(["Overdue", "Expired", "Failed", "Missing"]);
  });

  it("filters non-compliance and at-risk safety on the roll-up columns", () => {
    expect(viewById("non-compliant").filters).toEqual([
      { field: "profile.complianceStatus", operator: "eq", value: "NonCompliant" },
    ]);
    expect(viewById("at-risk").filters).toEqual([
      { field: "profile.safetyRating", operator: "eq", value: "AtRisk" },
    ]);
  });

  // The cutoff is derived from the clock the caller passes, so the view is
  // deterministic in tests and moves with the day in the product.
  it("cuts the expiry view off a fixed window from the given time", () => {
    const filter = viewById("expiring-soon").filters[0];
    expect(filter.field).toBe("profile.nextCredentialExpiry");
    expect(filter.operator).toBe("lte");
    expect(filter.value).toBe(NOW + EXPIRING_SOON_DAYS * 86_400);
  });
});

describe("activeRosterView", () => {
  it("reports everyone when nothing is filtered", () => {
    expect(activeRosterView([], rosterViews(NOW))).toBe("all");
  });

  it("recognises a view the user clicked", () => {
    const views = rosterViews(NOW);
    expect(activeRosterView(viewById("at-risk").filters, views)).toBe("at-risk");
  });

  // The expiry cutoff drifts with the clock. Matching on the value would mean
  // the chip stopped looking selected a second after it was clicked.
  it("still recognises the expiry view after the cutoff has moved on", () => {
    const clicked = rosterViews(NOW).find((view) => view.id === "expiring-soon");
    const later = rosterViews(NOW + 3_600);
    expect(activeRosterView(clicked!.filters, later)).toBe("expiring-soon");
  });

  it("reports nothing when the filters are somebody's own", () => {
    const views = rosterViews(NOW);
    expect(
      activeRosterView([{ field: "status", operator: "eq", value: "Active" }], views),
    ).toBeNull();
  });

  // A view plus an extra filter of the reader's own is no longer that view,
  // and the chip must not claim otherwise.
  it("reports nothing when a view has been narrowed further", () => {
    const views = rosterViews(NOW);
    expect(
      activeRosterView(
        [
          { field: "profile.safetyRating", operator: "eq", value: "AtRisk" },
          { field: "driverType", operator: "eq", value: "OTR" },
        ],
        views,
      ),
    ).toBeNull();
  });
});

// A prohibited driver must not be dispatched, so the roster needs a one-click
// way to see exactly who that is.
it("has a view for drivers who must not be dispatched", () => {
  const view = rosterViews(NOW).find((candidate) => candidate.id === "prohibited");
  expect(view?.filters).toEqual([
    { field: "profile.drugAlcoholStatus", operator: "eq", value: "Prohibited" },
  ]);
});
