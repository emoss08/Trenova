import { BASE_ONLY_UNAVAILABLE, unavailableFeatures } from "@/test/plan";
import { Resource } from "@trenova/shared/types/permission";
import type { UnavailableFeature } from "@trenova/shared/types/permission";
import { describe, expect, it } from "vitest";
import {
  drawableWorkerTabs,
  drawableWorkerViews,
  firstDrawableWorkerView,
  isWorkerTabDrawable,
  isWorkerViewDrawable,
  resolveWorkerArea,
  workerAreas,
  type WorkerAreaAccess,
} from "../worker-areas";

const EVERY_RESOURCE = new Set<string>(Object.values(Resource));

function access(
  unavailable: readonly UnavailableFeature[] = [],
  readable: ReadonlySet<string> = EVERY_RESOURCE,
): WorkerAreaAccess {
  return { unavailable, canRead: (resource) => readable.has(resource) };
}

function without(...resources: string[]): ReadonlySet<string> {
  return new Set([...EVERY_RESOURCE].filter((resource) => !resources.includes(resource)));
}

describe("drawable tabs", () => {
  it("draws every tab for a full plan and a reader who may read everything", () => {
    expect(drawableWorkerTabs(access())).toEqual([
      "overview",
      "profile",
      "employment",
      "qualification",
      "development",
      "safety",
      "time",
      "pay",
      "files",
      "activity",
    ]);
  });

  it("keeps the base record on a base-only plan and drops every add-on area", () => {
    const base = access(BASE_ONLY_UNAVAILABLE);

    expect(drawableWorkerTabs(base)).toEqual([
      "overview",
      "profile",
      "employment",
      "time",
      "files",
      "activity",
    ]);
    expect(drawableWorkerViews("time", base)).toEqual(["hours-of-service"]);
    expect(drawableWorkerViews("files", base)).toEqual(["documents"]);
  });

  it("drops only the views of the feature the plan leaves out", () => {
    const noTalent = access(unavailableFeatures("workforce.talent"));

    expect(isWorkerTabDrawable("development", noTalent)).toBe(false);
    expect(isWorkerTabDrawable("safety", noTalent)).toBe(true);
    expect(drawableWorkerViews("qualification", noTalent)).toEqual([
      "credentials",
      "dq-file",
      "drug-and-alcohol",
    ]);
  });

  it("gates benefits on the benefits feature and pay on driver pay, each on its own", () => {
    expect(drawableWorkerViews("pay", access(unavailableFeatures("tms.settlement")))).toEqual([
      "benefits",
    ]);
    expect(drawableWorkerViews("pay", access(unavailableFeatures("workforce.benefits")))).toEqual([
      "pay-profile",
    ]);
    expect(
      isWorkerTabDrawable(
        "pay",
        access(unavailableFeatures("tms.settlement", "workforce.benefits")),
      ),
    ).toBe(false);
  });

  it("withholds a view from a reader who may not read its records", () => {
    const reader = access([], without(Resource.DriverPayProfile, Resource.WorkerDOTTest));

    expect(isWorkerViewDrawable("pay-profile", reader)).toBe(false);
    expect(drawableWorkerViews("pay", reader)).toEqual(["benefits"]);
    expect(drawableWorkerViews("qualification", reader)).toEqual(["credentials", "dq-file"]);
  });

  it("does not draw a tab whose every view is withheld", () => {
    const reader = access([], without(Resource.WorkerTraining, Resource.PerformanceReview));

    expect(isWorkerTabDrawable("development", reader)).toBe(false);
    expect(firstDrawableWorkerView("development", reader)).toBeNull();
  });

  it("draws safety and activity only for readers of safety events and the audit log", () => {
    const reader = access([], without(Resource.WorkerSafetyEvent, Resource.AuditLog));

    expect(isWorkerTabDrawable("safety", reader)).toBe(false);
    expect(isWorkerTabDrawable("activity", reader)).toBe(false);
  });

  it("draws agent memory only where the assistant is on the plan and readable", () => {
    expect(drawableWorkerViews("files", access(unavailableFeatures("ai.desk")))).toEqual([
      "documents",
      "driver-portal",
    ]);
    expect(drawableWorkerViews("files", access([], without(Resource.Assistant)))).toEqual([
      "documents",
      "driver-portal",
    ]);
  });
});

describe("resolveWorkerArea", () => {
  it("keeps a drawable tab and view", () => {
    expect(resolveWorkerArea({ tab: "time", view: "leave" }, access())).toEqual({
      tab: "time",
      view: "leave",
    });
  });

  it("opens a tab named without a view on its first drawable view", () => {
    const reader = access([], without(Resource.WorkerCredential));

    expect(resolveWorkerArea({ tab: "qualification", view: null }, reader)).toEqual({
      tab: "qualification",
      view: "dq-file",
    });
  });

  it("moves a view that is withheld to the nearest drawable view of the same tab", () => {
    expect(
      resolveWorkerArea(
        { tab: "time", view: "time-off" },
        access(unavailableFeatures("workforce.time_off")),
      ),
    ).toEqual({ tab: "time", view: "hours-of-service" });
  });

  it("lands an unknown, off-plan or unreadable tab on the overview", () => {
    expect(resolveWorkerArea({ tab: "timeline", view: "history" }, access())).toEqual({
      tab: "overview",
      view: null,
    });
    expect(
      resolveWorkerArea({ tab: "development", view: "training" }, access(BASE_ONLY_UNAVAILABLE)),
    ).toEqual({ tab: "overview", view: null });
    expect(
      resolveWorkerArea({ tab: "activity", view: null }, access([], without(Resource.AuditLog))),
    ).toEqual({ tab: "overview", view: null });
  });

  it("gives a tab without views no view, whatever the address says", () => {
    expect(resolveWorkerArea({ tab: "safety", view: "credentials" }, access())).toEqual({
      tab: "safety",
      view: null,
    });
  });
});

describe("workerAreas", () => {
  it("answers which areas are drawn from one reading of plan and permissions", () => {
    const areas = workerAreas(access(BASE_ONLY_UNAVAILABLE));

    expect(areas.tabs).toContain("employment");
    expect(areas.tabs).not.toContain("pay");
    expect(areas.viewsOf("employment")).toEqual(["history", "checklists"]);
    expect(areas.isDrawable({ tab: "time", view: "hours-of-service" })).toBe(true);
    expect(areas.isDrawable({ tab: "time", view: "schedule" })).toBe(false);
    expect(areas.isDrawable({ tab: "profile", view: null })).toBe(true);
    expect(areas.resolve({ tab: "pay", view: "benefits" })).toEqual({
      tab: "overview",
      view: null,
    });
  });
});
