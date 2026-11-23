import type { WorkerHosDailyLog, WorkerHosLogEntry } from "@/lib/graphql/telematics";
import { describe, expect, it } from "vitest";
import {
  buildLogSegments,
  dayLaneTotals,
  formatLaneTotal,
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
