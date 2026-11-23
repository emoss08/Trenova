import type { EmploymentEffect } from "@/lib/graphql/worker-employment";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { describeEmploymentEffect, failedEffects } from "../employment-effects";

function effect(overrides: Partial<EmploymentEffect>): EmploymentEffect {
  return {
    kind: "DispatchRemoved",
    name: null,
    count: null,
    days: null,
    ptoType: null,
    startDate: null,
    endDate: null,
    failed: false,
    ...overrides,
  } as EmploymentEffect;
}

describe("describeEmploymentEffect", () => {
  it("words a payout with its plural and names the time off type below", () => {
    const one = describeEmploymentEffect(
      effect({ kind: "PtoPaidOut", days: "1.00", ptoType: "Sick" }),
      translate,
    );
    expect(one).toMatchObject({ text: "Pays out 1 day of time off.", sub: "Sick", danger: true });
    const many = describeEmploymentEffect(
      effect({ kind: "PtoPaidOut", days: "9.50", ptoType: "Vacation" }),
      translate,
    );
    expect(many.text).toBe("Pays out 9.5 days of time off.");
  });

  it("says the same effect in the past tense after the save", () => {
    const line = describeEmploymentEffect(
      effect({ kind: "PayAssignmentEnded", name: "OTR mileage" }),
      translate,
      "did",
    );
    expect(line).toMatchObject({ text: "Ended the pay assignment.", sub: "OTR mileage" });
  });

  it("names the cancelled time off request by its dates", () => {
    const line = describeEmploymentEffect(
      effect({
        kind: "TimeOffCancelled",
        ptoType: "Vacation",
        days: "2.00",
        startDate: Date.UTC(2026, 11, 23, 12) / 1000,
        endDate: Date.UTC(2026, 11, 24, 12) / 1000,
      }),
      translate,
    );
    expect(line.text).toBe("Cancels the time off request from Dec 23, 2026 to Dec 24, 2026.");
    expect(line.sub).toBe("Vacation · 2 days");
  });

  it("counts checklists closed and courses assigned", () => {
    expect(
      describeEmploymentEffect(effect({ kind: "ChecklistClosed", count: 2 }), translate).text,
    ).toBe("Closes 2 open checklists this makes moot.");
    expect(
      describeEmploymentEffect(
        effect({ kind: "ChecklistClosed", count: 1, name: "Onboarding" }),
        translate,
      ).text,
    ).toBe("Closes the Onboarding checklist, which this makes moot.");
    expect(
      describeEmploymentEffect(effect({ kind: "TrainingAssigned", count: 1 }), translate).text,
    ).toBe("Assigns 1 required course.");
  });

  it("tells the person a portal revocation that failed must be done by hand", () => {
    const line = describeEmploymentEffect(
      effect({ kind: "PortalAccessRevoked", failed: true }),
      translate,
      "did",
    );
    expect(line).toMatchObject({
      text: "Dash access could not be shut off. Revoke it by hand.",
      failed: true,
      danger: true,
    });
  });

  it("picks out the effects that failed", () => {
    const ok = effect({ kind: "ChecklistStarted", name: "Offboarding" });
    const bad = effect({ kind: "PortalAccessRevoked", failed: true });
    expect(failedEffects([ok, bad])).toEqual([bad]);
  });
});
