import type { WorkerHosDailyLog, WorkerHosLogEntry } from "@/lib/graphql/telematics";
import { fromUserWallClock, setUserDatePreferences } from "@trenova/shared/lib/date";
import { afterEach, describe, expect, it } from "vitest";
import {
  buildDaySlots,
  buildLogSegments,
  dayLaneTotals,
  formatLaneTotal,
  HOS_RECORDS_SECONDS,
  hosWindow,
  laneTotalsMs,
  logHourLabels,
  logPath,
  type LogGridGeometry,
} from "../hos-model";

const DAY = 1_800_000_000;
const END = DAY + 86_400;
const minutes = (value: number) => DAY + value * 60;

function entry(
  status: string,
  start: number,
  end: number | null,
  extra: Partial<WorkerHosLogEntry> = {},
): WorkerHosLogEntry {
  return {
    hosStatusType: status,
    logStartAt: minutes(start),
    logEndAt: end === null ? null : minutes(end),
    remark: null,
    vehicleId: null,
    vehicleName: null,
    latitude: null,
    longitude: null,
    codrivers: null,
    ...extra,
  };
}

const geometry: LogGridGeometry = { left: 40, width: 1440, top: 22, laneHeight: 30 };

describe("buildLogSegments", () => {
  it("sorts the day's entries, clips them to the day and runs the open one to now", () => {
    const segments = buildLogSegments(
      [
        entry("driving", 430, 572, { vehicleName: "Tractor 4417" }),
        entry("offDuty", -120, 390, { remark: "Rest, Gary, IN" }),
        entry("onDuty", 390, 430),
        entry("driving", 822, null),
      ],
      DAY,
      END,
      minutes(968),
    );

    expect(
      segments.map((segment) => [segment.lane, segment.startSec, segment.endSec, segment.ongoing]),
    ).toEqual([
      [0, DAY, minutes(390), false],
      [3, minutes(390), minutes(430), false],
      [2, minutes(430), minutes(572), false],
      [2, minutes(822), minutes(968), true],
    ]);
    expect(segments[0].remark).toBe("Rest, Gary, IN");
    expect(segments[2].vehicleName).toBe("Tractor 4417");
  });

  it("puts personal conveyance in the off-duty lane and yard moves in the on-duty lane", () => {
    const segments = buildLogSegments(
      [entry("personalConveyance", 0, 60), entry("yardMove", 60, 90), entry("sleeperBed", 90, 120)],
      DAY,
      END,
      END,
    );
    expect(segments.map((segment) => segment.lane)).toEqual([0, 3, 1]);
  });

  it("drops an entry that ends before the day or is empty", () => {
    expect(
      buildLogSegments([entry("driving", -60, -10), entry("onDuty", 30, 30)], DAY, END, END),
    ).toEqual([]);
  });
});

describe("lane totals", () => {
  it("adds each lane's milliseconds from the segments the line draws", () => {
    const segments = buildLogSegments(
      [
        entry("offDuty", 0, 390),
        entry("onDuty", 390, 430),
        entry("driving", 430, 572),
        entry("onDuty", 572, 612),
        entry("offDuty", 612, 810),
        entry("onDuty", 810, 822),
        entry("driving", 822, 968),
      ],
      DAY,
      END,
      minutes(968),
    );
    const totals = laneTotalsMs(segments);
    expect(totals.map(formatLaneTotal)).toEqual(["9:48", "0:00", "4:48", "1:32"]);
  });

  it("falls back to the provider's summary for a day with no segments", () => {
    const dailyLog = {
      offDutyDurationMs: 36_000_000,
      personalConveyanceDurationMs: 600_000,
      sleeperBerthDurationMs: 0,
      driveDurationMs: 28_800_000,
      onDutyDurationMs: 3_600_000,
      yardMoveDurationMs: 900_000,
    } as WorkerHosDailyLog;
    expect(dayLaneTotals([], dailyLog).map(formatLaneTotal)).toEqual([
      "10:10",
      "0:00",
      "8:00",
      "1:15",
    ]);
    expect(dayLaneTotals([], null)).toEqual([0, 0, 0, 0]);
  });
});

describe("logPath", () => {
  it("steps straight between lanes with square changes and restarts after a gap", () => {
    const segments = buildLogSegments(
      [
        entry("offDuty", 0, 360),
        entry("driving", 360, 600),
        entry("driving", 600, 660),
        entry("onDuty", 720, 780),
      ],
      DAY,
      END,
      END,
    );
    expect(logPath(segments, DAY, END, geometry)).toBe("M40 37 H400 V97 H640 H700 M760 127 H820");
  });

  it("draws nothing for a day with no segments", () => {
    expect(logPath([], DAY, END, geometry)).toBe("");
  });
});

describe("logHourLabels", () => {
  it("writes midnight and noon as the paper log does, the other hours one to eleven", () => {
    const labels = logHourLabels("M", "N");
    expect(labels).toHaveLength(25);
    expect(labels.slice(0, 3)).toEqual(["M", "1", "2"]);
    expect(labels.slice(11, 14)).toEqual(["11", "N", "1"]);
    expect(labels[24]).toBe("M");
  });
});

describe("hosWindow", () => {
  const at = (hour: number, minute: number, day = 23) =>
    fromUserWallClock(new Date(2026, 10, day, hour, minute)) ?? 0;

  // The reads are keyed by the window, so it must not move between two readings of one day.
  it("stays the same all day and moves at midnight", () => {
    const morning = hosWindow(at(0, 1));
    const night = hosWindow(at(23, 59));
    expect(night).toEqual(morning);
    expect(morning.startDate).toBe("2026-11-17");
    expect(morning.endDate).toBe("2026-11-23");

    const next = hosWindow(at(0, 0, 24));
    expect(next.endDate).toBe("2026-11-24");
    expect(next.until).toBe(at(0, 0, 25));
  });

  it("covers thirty days of seconds up to the end of today", () => {
    const span = hosWindow(at(9, 30));
    expect(span.until).toBe(at(0, 0, 24));
    expect(span.until - span.since).toBe(HOS_RECORDS_SECONDS);
  });
});

/**
 * The server refuses a provider read longer than 31 days:
 * `services/tms/internal/modules/workforce/domain/telematics/window.go`, `MaxProviderWindowDays`.
 * `RequireProviderWindow` refuses `endAt - startAt > 31 * 86_400` seconds (forms and logs);
 * `RequireProviderDays` refuses day keys whose difference is 31 days or more (daily logs).
 */
const SERVER_MAX_WINDOW_DAYS = 31;
const SERVER_MAX_WINDOW_SECONDS = SERVER_MAX_WINDOW_DAYS * 86_400;

describe("every hours-of-service read stays inside the server's window", () => {
  afterEach(() => setUserDatePreferences({ timezone: undefined }));

  // Days chosen on and after the turns of daylight saving time, where a window built from wall
  // clock days is an hour longer or shorter than its count of days.
  const zones = ["America/Chicago", "Europe/London", "Australia/Lord_Howe", "Asia/Tokyo", "UTC"];
  const instants = [
    "2026-11-23T15:00:00Z",
    "2026-11-01T12:00:00Z",
    "2026-11-24T05:59:00Z",
    "2026-03-09T12:00:00Z",
    "2026-04-05T16:00:00Z",
    "2026-10-04T15:30:00Z",
    "2026-10-26T00:30:00Z",
  ];

  for (const zone of zones) {
    it(`in ${zone}`, () => {
      setUserDatePreferences({ timezone: zone });
      for (const iso of instants) {
        const now = Math.floor(Date.parse(iso) / 1000);
        const span = hosWindow(now);

        // WorkerFormSubmissions (startTime, endTime) and WorkerHosViolations (since).
        expect(span.until - span.since, `${zone} ${iso} records`).toBeLessThanOrEqual(
          SERVER_MAX_WINDOW_SECONDS,
        );
        expect(span.until).toBeGreaterThan(span.since);
        expect(span.until).toBeGreaterThan(now);

        // WorkerHosDailyLogs (startDate, endDate): fewer than 31 days apart.
        const days = (Date.parse(span.endDate) - Date.parse(span.startDate)) / 86_400_000;
        expect(days, `${zone} ${iso} daily logs`).toBeLessThan(SERVER_MAX_WINDOW_DAYS);
        expect(days).toBeGreaterThanOrEqual(0);

        // WorkerHosLogs (startTime, endTime): one day's slot at a time.
        for (const slot of buildDaySlots([], span.today)) {
          expect(slot.endAt - slot.startAt, `${zone} ${iso} ${slot.key}`).toBeLessThanOrEqual(
            SERVER_MAX_WINDOW_SECONDS,
          );
          expect(slot.endAt).toBeGreaterThan(slot.startAt);
        }
      }
    });
  }
});
