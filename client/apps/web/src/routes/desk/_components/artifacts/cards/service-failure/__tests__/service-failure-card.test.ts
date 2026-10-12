import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  ediCodes,
  failureCause,
  failurePlace,
  failureStanding,
  isUnsettled,
  missClock,
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

describe("missClock", () => {
  const MINUTE = 60;
  const shape = (clock: ReturnType<typeof missClock>) =>
    clock.map((entry) =>
      entry.kind === "event"
        ? [entry.key, entry.label, entry.state]
        : [entry.window, entry.text, Number(entry.share.toFixed(3))],
    );

  it("reads the cutoff, the end of its grace and the arrival, with the grace and the miss between", () => {
    const clock = missClock(serviceFailure(), t);
    expect(shape(clock)).toEqual([
      ["due", "Due by", null],
      ["grace", "30m of grace", 0.222],
      ["grace", "Grace ends", "warn"],
      ["late", "Late by 1h 45m", 0.778],
      ["arrived", "Arrived", "bad"],
    ]);
    const days = clock.flatMap((entry) => (entry.kind === "event" ? [entry.day] : []));
    expect(days[0]).not.toBeNull();
    expect(days.slice(1)).toEqual([null, null]);
  });

  it("leaves the cutoff alone with no grace, and names the day again once the clock crosses midnight", () => {
    const cutoff = new Date(2026, 9, 9, 23, 0).getTime() / 1000;
    const clock = missClock(
      { scheduledCutoff: cutoff, actualArrival: cutoff + 3 * HOUR, gracePeriodMinutes: 0 },
      t,
    );
    expect(shape(clock)).toEqual([
      ["due", "Due by", null],
      ["late", "Late by 3h", 1],
      ["arrived", "Arrived", "bad"],
    ]);
    const [due, , arrived] = clock;
    expect(due?.kind === "event" && due.day).toBeTruthy();
    expect(arrived?.kind === "event" && arrived.day).not.toBe(due?.kind === "event" && due.day);
  });

  it("lands an arrival inside the grace or before the cutoff where it fell, with nothing late", () => {
    const cutoff = NOW - 4 * HOUR;
    expect(
      shape(
        missClock(
          { scheduledCutoff: cutoff, actualArrival: cutoff + 10 * MINUTE, gracePeriodMinutes: 30 },
          t,
        ),
      ),
    ).toEqual([
      ["due", "Due by", null],
      ["grace", "10m of grace", 0.333],
      ["arrived", "Arrived", null],
      ["grace", "20m of grace", 0.667],
      ["grace", "Grace ends", null],
    ]);
    expect(
      shape(
        missClock(
          { scheduledCutoff: cutoff, actualArrival: cutoff - 15 * MINUTE, gracePeriodMinutes: 15 },
          t,
        ),
      ),
    ).toEqual([
      ["arrived", "Arrived", null],
      ["early", "15m before the cutoff", 0.5],
      ["due", "Due by", null],
      ["grace", "15m of grace", 0.5],
      ["grace", "Grace ends", null],
    ]);
  });

  it("draws an arrival on the cutoff itself as one moment after another, with no stretch between", () => {
    const cutoff = NOW - HOUR;
    expect(
      shape(
        missClock({ scheduledCutoff: cutoff, actualArrival: cutoff, gracePeriodMinutes: 0 }, t),
      ),
    ).toEqual([
      ["due", "Due by", null],
      ["arrived", "Arrived", null],
    ]);
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

describe("ediCodes", () => {
  it("says which 214 code is set on the failure and which comes from its type", () => {
    expect(ediCodes(serviceFailure({ x12ReasonCodeOverride: " AG " }), t)).toEqual([
      { key: "status", label: "Status code", code: null, note: "From the failure type" },
      { key: "reason", label: "Reason code", code: "AG", note: "Set on this failure" },
      { key: "exception", label: "Exception code", code: null, note: "None" },
    ]);
    expect(
      ediCodes(serviceFailure({ x12StatusCodeOverride: "X6", x12ExceptionCode: "A1" }), t).map(
        (code) => [code.key, code.code],
      ),
    ).toEqual([
      ["status", "X6"],
      ["reason", null],
      ["exception", "A1"],
    ]);
  });
});
