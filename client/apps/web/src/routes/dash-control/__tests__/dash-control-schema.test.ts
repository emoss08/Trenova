import { describe, expect, it } from "vitest";
import { dashControlSchema, type DashControl } from "@/types/dash-control";
import { savedDashControl } from "./dash-control-fixture";

const issues = (changes: Partial<DashControl>) => {
  const result = dashControlSchema.safeParse({ ...savedDashControl, ...changes });
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
};

describe("dashControlSchema", () => {
  it("accepts the record the server sends", () => {
    expect(issues({})).toEqual([]);
  });

  it("needs a threshold while detention alerts are on", () => {
    expect(issues({ enableDetentionAlerts: true, detentionAlertThresholdMinutes: null })).toEqual([
      "detentionAlertThresholdMinutes",
    ]);
  });

  it("holds the threshold between 15 minutes and a day while alerts are on", () => {
    expect(issues({ enableDetentionAlerts: true, detentionAlertThresholdMinutes: 14 })).toEqual([
      "detentionAlertThresholdMinutes",
    ]);
    expect(issues({ enableDetentionAlerts: true, detentionAlertThresholdMinutes: 1441 })).toEqual([
      "detentionAlertThresholdMinutes",
    ]);
    expect(issues({ enableDetentionAlerts: true, detentionAlertThresholdMinutes: 15 })).toEqual([]);
    expect(issues({ enableDetentionAlerts: true, detentionAlertThresholdMinutes: 1440 })).toEqual(
      [],
    );
  });

  // The page hides these rows while what they depend on is off, and the save sends them
  // as the server needs (dash-control-submit.ts); refusing them here would fail on a
  // setting nobody can see.
  it("lets a hidden setting through for the save to correct", () => {
    expect(
      issues({
        requireLoadAcknowledgment: false,
        allowLoadRefusals: true,
        sendCredentialReminders: false,
        driverDigestCadence: "Weekly",
        enableDetentionAlerts: false,
        detentionAlertThresholdMinutes: null,
      }),
    ).toEqual([]);
  });

  it("refuses a round-up day that is not a day of the week", () => {
    expect(issues({ driverDigestWeekday: 7 })).toEqual(["driverDigestWeekday"]);
  });

  it("refuses a cadence the server does not know", () => {
    expect(issues({ driverDigestCadence: "Hourly" as DashControl["driverDigestCadence"] })).toEqual(
      ["driverDigestCadence"],
    );
  });
});
