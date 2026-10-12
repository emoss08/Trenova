import type { TriageStopTimes } from "@/lib/graphql/inbox";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { lateMinutesOf, stopRows, stopsReading, type StopFormat } from "../stops-reading";

const MIN = 60;
const START = 1_000_000;
const END = START + 120 * MIN;

const format: StopFormat = { time: (seconds) => `t${(seconds - START) / MIN}` };

function stops(overrides: Partial<TriageStopTimes> = {}): TriageStopTimes {
  return {
    shipment: {
      id: "shp_1",
      proNumber: "SEED-PAY-002",
      bol: "30218",
      poNumber: "99-210",
      originCity: "Savannah",
      originState: "GA",
      destinationCity: "Atlanta",
      destinationState: "GA",
    },
    stopName: "Harborline DC",
    appointmentStart: START,
    appointmentEnd: END,
    arrivedAt: END + 40 * MIN,
    checkedInAt: END + 52 * MIN,
    departedAt: END + 202 * MIN,
    lateMinutes: 40,
    billedDetentionMinutes: 150,
    billedDetentionAmount: "185.00",
    detentionPaysWhenLate: false,
    arrivalSource: "ELD",
    ...overrides,
  };
}

describe("lateMinutesOf", () => {
  it("measures lateness from the arrival and the window's close, not the stored figure", () => {
    expect(lateMinutesOf(stops({ lateMinutes: 0 }))).toBe(40);
    expect(lateMinutesOf(stops({ arrivedAt: END + 30 }))).toBe(1);
  });

  it("is zero inside the window and unknown without an arrival", () => {
    expect(lateMinutesOf(stops({ arrivedAt: END - 5 * MIN, lateMinutes: 12 }))).toBe(0);
    expect(lateMinutesOf(stops({ arrivedAt: null }))).toBeNull();
  });

  it("takes the server's figure when there is no window to measure against", () => {
    expect(lateMinutesOf(stops({ appointmentEnd: null, lateMinutes: 25 }))).toBe(25);
  });
});

describe("stopRows", () => {
  it("lays out the window, the arrival with its source, check-in and departure", () => {
    expect(stopRows(translate, stops(), format)).toEqual([
      { key: "appointment", label: "Appointment window", value: "t0 – t120", late: false },
      { key: "arrived", label: "Arrived (ELD)", value: "t160", late: true },
      { key: "checkedIn", label: "Checked in at gate", value: "t172", late: false },
      { key: "departed", label: "Unloaded and out", value: "t322", late: false },
    ]);
  });

  it("says what is not on record instead of leaving a gap", () => {
    const rows = stopRows(
      translate,
      stops({
        appointmentStart: null,
        arrivedAt: null,
        checkedInAt: null,
        departedAt: null,
        arrivalSource: "",
      }),
      format,
    );
    expect(rows.map((row) => [row.label, row.value, row.late])).toEqual([
      ["Appointment window", "By t120", false],
      ["Arrived", "Not recorded", false],
      ["Checked in at gate", "Not recorded", false],
      ["Unloaded and out", "Not recorded", false],
    ]);
    expect(
      stopRows(translate, stops({ appointmentStart: null, appointmentEnd: null }), format)[0].value,
    ).toBe("Not recorded");
  });

  it("does not repeat a source the server says is not recorded", () => {
    expect(stopRows(translate, stops({ arrivalSource: "Not recorded" }), format)[1].label).toBe(
      "Arrived",
    );
  });
});

describe("stopsReading", () => {
  it("says a late arrival under terms that only pay on time will not hold up", () => {
    expect(stopsReading(translate, stops(), "Harborline Grocers", format)).toBe(
      "The truck arrived 40 minutes after the window. Harborline Grocers only pays detention on an on-time arrival, so the charge is unlikely to stand.",
    );
  });

  it("says a late arrival under terms that pay anyway can stand, in hours past an hour", () => {
    expect(
      stopsReading(
        translate,
        stops({ arrivedAt: END + 125 * MIN, detentionPaysWhenLate: true }),
        null,
        format,
      ),
    ).toBe(
      "The truck arrived 2 hours 5 minutes after the window, but the customer pays detention on a late arrival too, so the charge can stand.",
    );
    expect(
      stopsReading(translate, stops({ arrivedAt: END + 60 * MIN }), null, format),
    ).toContain("arrived 1 hour after the window");
  });

  it("says an on-time arrival that waited carries the charge, and one that did not owes none", () => {
    expect(
      stopsReading(translate, stops({ arrivedAt: END - 10 * MIN }), "Harborline Grocers", format),
    ).toBe(
      "The truck arrived inside the window and was held 2 hours 30 minutes, so the charge stands.",
    );
    expect(
      stopsReading(
        translate,
        stops({ arrivedAt: END - 10 * MIN, billedDetentionMinutes: 0 }),
        null,
        format,
      ),
    ).toBe("The truck arrived inside the window.");
  });

  it("says when the times cannot settle it", () => {
    expect(stopsReading(translate, stops({ arrivedAt: null }), null, format)).toBe(
      "There's no arrival time on record for Harborline DC, so the stop times can't settle it.",
    );
    expect(
      stopsReading(
        translate,
        stops({ appointmentStart: null, appointmentEnd: null, lateMinutes: 0 }),
        null,
        format,
      ),
    ).toBe(
      "The truck arrived at t160, but there's no appointment window on record to measure it against.",
    );
  });
});
