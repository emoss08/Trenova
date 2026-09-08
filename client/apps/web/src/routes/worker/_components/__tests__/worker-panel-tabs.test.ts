import { describe, expect, it } from "vitest";
import { resolveWorkerPanelTab } from "../worker-panel-tabs";

describe("resolveWorkerPanelTab", () => {
  // The employment history used to be its own tab. Concerns and saved links
  // still say "timeline", and they must land on the history rather than on a
  // tab that no longer exists.
  it("sends the old timeline tab to the employment history view", () => {
    expect(resolveWorkerPanelTab("timeline", "details")).toEqual({
      tab: "employment",
      employmentView: "history",
    });
  });

  it("keeps the reader's own choice of view on the employment tab", () => {
    expect(resolveWorkerPanelTab("employment", "history")).toEqual({
      tab: "employment",
      employmentView: "history",
    });
    expect(resolveWorkerPanelTab("employment", "details")).toEqual({
      tab: "employment",
      employmentView: "details",
    });
  });

  it("leaves every other tab alone", () => {
    expect(resolveWorkerPanelTab("credentials", "history")).toEqual({
      tab: "credentials",
      employmentView: "history",
    });
  });
});
