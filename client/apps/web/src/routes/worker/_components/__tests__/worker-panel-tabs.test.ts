import { BASE_ONLY_UNAVAILABLE, unavailableFeatures } from "@/test/plan";
import { describe, expect, it } from "vitest";
import { isWorkerTabOnPlan, resolveWorkerPanelTab } from "../worker-panel-tabs";

describe("resolveWorkerPanelTab", () => {
  // The employment history used to be its own tab. Concerns and saved links
  // still say "timeline", and they must land on the history rather than on a
  // tab that no longer exists.
  it("sends the old timeline tab to the employment history view", () => {
    expect(resolveWorkerPanelTab("timeline", "details", [])).toEqual({
      tab: "employment",
      employmentView: "history",
    });
  });

  it("keeps the reader's own choice of view on the employment tab", () => {
    expect(resolveWorkerPanelTab("employment", "history", [])).toEqual({
      tab: "employment",
      employmentView: "history",
    });
    expect(resolveWorkerPanelTab("employment", "details", [])).toEqual({
      tab: "employment",
      employmentView: "details",
    });
  });

  it("leaves every other tab alone", () => {
    expect(resolveWorkerPanelTab("credentials", "history", [])).toEqual({
      tab: "credentials",
      employmentView: "history",
    });
  });

  it("lands a link to a tab the plan leaves out on the overview", () => {
    expect(resolveWorkerPanelTab("training", "details", BASE_ONLY_UNAVAILABLE)).toEqual({
      tab: "overview",
      employmentView: "details",
    });
    expect(
      resolveWorkerPanelTab("pto", "details", unavailableFeatures("workforce.time_off")),
    ).toEqual({ tab: "overview", employmentView: "details" });
  });

  it("keeps base tabs on a base-only plan", () => {
    expect(resolveWorkerPanelTab("checklist", "details", BASE_ONLY_UNAVAILABLE)).toEqual({
      tab: "checklist",
      employmentView: "details",
    });
  });
});

describe("isWorkerTabOnPlan", () => {
  it("drops every add-on tab on a base-only plan and keeps the base record", () => {
    const tabs = [
      "overview",
      "general",
      "employment",
      "compliance",
      "credentials",
      "checklist",
      "training",
      "safety",
      "testing",
      "dqf",
      "reviews",
      "hos",
      "pay",
      "pto",
      "schedule",
      "leave",
      "documents",
      "portal",
    ];

    expect(tabs.filter((tab) => isWorkerTabOnPlan(tab, BASE_ONLY_UNAVAILABLE))).toEqual([
      "overview",
      "general",
      "employment",
      "compliance",
      "checklist",
      "hos",
      "documents",
    ]);
  });

  it("drops only the tabs of the feature the plan leaves out", () => {
    const unavailable = unavailableFeatures("workforce.talent");

    expect(isWorkerTabOnPlan("training", unavailable)).toBe(false);
    expect(isWorkerTabOnPlan("reviews", unavailable)).toBe(false);
    expect(isWorkerTabOnPlan("safety", unavailable)).toBe(true);
    expect(isWorkerTabOnPlan("pto", unavailable)).toBe(true);
  });
});
