import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  failureCause,
  failureFacts,
  failurePlace,
  failureStanding,
  failureTimes,
  isUnsettled,
  reviewSteps,
  stopLine,
} from "../service-failure-card";
import { HOUR, NOW, serviceFailure } from "./fixtures";

/** Fills a message's numbered places the way the catalog would. */
const t = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const mia = { id: "usr_1", name: "Mia Chen" };
const omar = { id: "usr_2", name: "Omar Haddad" };

describe("failureStanding", () => {
  it("asks for a look while open, follows a review, and settles once resolved or voided", () => {
    expect(failureStanding("Open")).toEqual({ label: "Open", tone: "warn" });
    expect(failureStanding("Reviewed")).toEqual({ label: "Reviewed", tone: "progress" });
    expect(failureStanding("Resolved")).toEqual({ label: "Resolved", tone: "success" });
    expect(failureStanding("Voided")).toEqual({ label: "Voided", tone: "muted" });
    expect(isUnsettled("Open")).toBe(true);
    expect(isUnsettled("Reviewed")).toBe(true);
    expect(isUnsettled("Resolved")).toBe(false);
    expect(isUnsettled("Voided")).toBe(false);
  });
});

describe("failurePlace and stopLine", () => {
  it("names the stop by its location, and which stop of the load it was", () => {
    const failure = serviceFailure();
    expect(failurePlace(failure, t)).toBe("Tyson Foods DC");
    expect(stopLine(failure, t)).toBe("Delivery, stop 3");
  });

  it("falls back to the town, then says the stop is unknown, and reads the recorded kind with no stop", () => {
    const unnamed = serviceFailure({
      stop: {
        ...serviceFailure().stop!,
        location: { ...serviceFailure().stop!.location!, name: " " },
      },
    });
    expect(failurePlace(unnamed, t)).toBe("Springdale, AR 72762");
    const gone = serviceFailure({ stop: null, stopType: "Pickup" });
    expect(failurePlace(gone, t)).toBe("Unknown stop");
    expect(stopLine(gone, t)).toBe("Pickup");
  });
});

describe("failureTimes", () => {
  it("reads when it was due, the grace, when the truck came and by how much it missed", () => {
    const stats = failureTimes(serviceFailure(), t);
    expect(stats.map((stat) => [stat.key, stat.label, stat.state])).toEqual([
      ["due", "Due by", undefined],
      ["grace", "Grace", undefined],
      ["arrived", "Arrived", "bad"],
      ["late", "Late by", "bad"],
    ]);
    expect(stats[1]?.value).toBe("30m");
    expect(stats[2]?.value).not.toBe(stats[0]?.value);
    expect(stats[3]?.value).toBe("1h 45m");
  });
});

describe("reviewSteps", () => {
  it("waits on a reviewer while open, with the system as the one that recorded it", () => {
    const steps = reviewSteps(serviceFailure(), t);
    expect(steps.map((step) => [step.key, step.label, step.state])).toEqual([
      ["recorded", "Detected", "done"],
      ["reviewed", "Reviewed", "now"],
      ["resolved", "Resolved", "todo"],
    ]);
    expect(steps[0]?.line).not.toContain("·");
    expect(steps[1]?.line).toBe("Waiting on a reviewer");
    expect(steps[2]?.line).toBe("Not yet");
  });

  it("says who logged, reviewed and resolved it, and when", () => {
    const steps = reviewSteps(
      serviceFailure({
        source: "Manual",
        status: "Resolved",
        createdBy: mia,
        reviewedAt: NOW - HOUR,
        reviewedBy: omar,
        resolvedAt: NOW,
        resolvedBy: mia,
      }),
      t,
    );
    expect(steps.map((step) => [step.label, step.state])).toEqual([
      ["Logged by hand", "done"],
      ["Reviewed", "done"],
      ["Resolved", "done"],
    ]);
    expect(steps[0]?.line).toMatch(/ · Mia Chen$/u);
    expect(steps[1]?.line).toMatch(/ · Omar Haddad$/u);
    expect(steps[2]?.line).toMatch(/ · Mia Chen$/u);
  });

  it("waits to be resolved once reviewed", () => {
    const steps = reviewSteps(
      serviceFailure({ source: "EDI", status: "Reviewed", reviewedAt: NOW, reviewedBy: omar }),
      t,
    );
    expect(steps[0]?.label).toBe("Received over EDI");
    expect(steps[2]).toMatchObject({ state: "now", line: "Waiting to be resolved" });
  });

  it("closes a voided failure with why, and marks a review it skipped", () => {
    const steps = reviewSteps(
      serviceFailure({
        source: "Integration",
        status: "Voided",
        voidedAt: NOW,
        voidedBy: omar,
        voidReason: "Appointment was moved by the consignee",
      }),
      t,
    );
    expect(steps.map((step) => [step.key, step.state, step.tone ?? null])).toEqual([
      ["recorded", "done", null],
      ["reviewed", "todo", null],
      ["voided", "now", "closed"],
    ]);
    expect(steps[0]?.label).toBe("Received from an integration");
    expect(steps[1]?.line).toBe("Not reviewed");
    expect(steps[2]?.note).toBe("Appointment was moved by the consignee");
    expect(steps[2]?.line).toMatch(/ · Omar Haddad$/u);
  });
});

describe("failureCause", () => {
  it("reads the reason code by its label, category and code", () => {
    expect(
      failureCause(
        serviceFailure({
          reasonCode: {
            id: "rc_1",
            code: "CNS-DLY",
            label: "Consignee delayed unloading",
            description: " Dock was full when the truck arrived. ",
            category: "Consignee",
          },
        }),
      ),
    ).toEqual({
      name: "Consignee delayed unloading",
      code: "CNS-DLY",
      category: "Consignee",
      description: "Dock was full when the truck arrived.",
    });
  });

  it("names a reason with no label by its code, and has none when none was given", () => {
    expect(
      failureCause(
        serviceFailure({
          reasonCode: { id: "rc_2", code: "WX", label: " ", description: "", category: "Weather" },
        }),
      )?.name,
    ).toBe("WX");
    expect(failureCause(serviceFailure())).toBeNull();
  });
});

describe("failureFacts", () => {
  it("lists what the office reads, saying when a 214 code comes from the failure's type", () => {
    const facts = Object.fromEntries(
      failureFacts(serviceFailure({ x12ReasonCodeOverride: "AG" }), t).map((fact) => [
        fact.key,
        fact.value,
      ]),
    );
    expect(facts).toMatchObject({
      number: "SF-2610-0042",
      type: "Late delivery",
      source: "Detected",
      stop: "Delivery, stop 3",
      status: "From the failure type",
      reason: "AG",
      exception: "None",
    });
  });
});
