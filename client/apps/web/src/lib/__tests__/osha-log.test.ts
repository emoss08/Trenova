import {
  caseFilterCounts,
  certificationTrack,
  certifyBlocker,
  certifyBlockerMessage,
  filterCases,
  illnessTypeNumber,
  logColumn,
  matchesQuery,
  postingState,
  summaryCaption,
  yearsOffered,
} from "@/lib/osha-log";
import { describe, expect, it } from "vitest";

const DAY = 86_400;
// The 2026 window: February 1 to April 30, 2027.
const POST_FROM = Date.UTC(2027, 1, 1) / 1000;
const POST_THROUGH = Date.UTC(2027, 3, 30, 23, 59, 59) / 1000;
const formatDate = (unix: number) => new Date(unix * 1000).toISOString().slice(0, 10);

function totals(over: Record<string, number> = {}) {
  return {
    deaths: 0,
    daysAwayCases: 2,
    jobTransferCases: 1,
    otherRecordableCases: 4,
    totalRecordableCases: 7,
    totalDaysAway: 40,
    totalDaysRestricted: 12,
    injuryCount: 6,
    skinDisorderCount: 1,
    respiratoryCount: 0,
    poisoningCount: 0,
    hearingLossCount: 0,
    otherIllnessCount: 0,
    openCases: 0,
    ...over,
  };
}

function summary(over: Record<string, unknown> = {}) {
  return {
    status: "Draft",
    averageEmployees: 120,
    totalHoursWorked: 240_000,
    executiveName: "Dana Reyes",
    certifiedAt: null,
    submittedAt: null,
    ...over,
  };
}

function entry(over: Record<string, unknown> = {}) {
  return {
    id: "inj_1",
    caseNumber: 1,
    classification: "DaysAway",
    illnessType: "Injury",
    status: "Closed",
    recordable: true,
    logName: "Ada Byrne",
    description: "Slipped on the trailer step",
    location: "Yard, dock 4",
    bodyPart: "Left ankle",
    occurredAt: Date.UTC(2026, 2, 3) / 1000,
    daysAway: 12,
    daysRestricted: 0,
    ...over,
  };
}

describe("yearsOffered", () => {
  it("offers the retention period, this year first", () => {
    expect(yearsOffered(2026)).toEqual([2026, 2025, 2024, 2023, 2022]);
  });
});

describe("the form's own letters and numbers", () => {
  it("maps a classification to its 300 log column", () => {
    expect(logColumn("Death")).toBe("G");
    expect(logColumn("DaysAway")).toBe("H");
    expect(logColumn("JobTransferOrRestriction")).toBe("I");
    expect(logColumn("OtherRecordable")).toBe("J");
    expect(logColumn("FirstAidOnly")).toBeNull();
    expect(logColumn("NotRecordable")).toBeNull();
  });

  it("numbers illness types as the 300A does", () => {
    expect(illnessTypeNumber("Injury")).toBe(1);
    expect(illnessTypeNumber("HearingLoss")).toBe(5);
    expect(illnessTypeNumber("OtherIllness")).toBe(6);
    expect(illnessTypeNumber("Unknown")).toBeNull();
  });
});

describe("postingState", () => {
  it("counts the days until the window opens", () => {
    expect(postingState(POST_FROM - 10 * DAY, POST_FROM, POST_THROUGH)).toEqual({
      phase: "before",
      days: 10,
    });
  });

  it("is open on the first and last day, never reporting zero days left", () => {
    expect(postingState(POST_FROM, POST_FROM, POST_THROUGH).phase).toBe("open");
    const lastDay = postingState(POST_THROUGH - 3600, POST_FROM, POST_THROUGH);
    expect(lastDay).toEqual({ phase: "open", days: 1 });
  });

  it("closes the second after the window ends", () => {
    expect(postingState(POST_THROUGH + 1, POST_FROM, POST_THROUGH)).toEqual({
      phase: "closed",
      days: 0,
    });
  });
});

describe("certificationTrack", () => {
  const base = { postFrom: POST_FROM, postThrough: POST_THROUGH, formatDate };

  it("points at the open cases before anything else", () => {
    const steps = certificationTrack({
      ...base,
      totals: totals({ openCases: 2 }),
      summary: summary(),
      now: POST_FROM - 60 * DAY,
    });
    expect(steps.map((step) => [step.id, step.state])).toEqual([
      ["close", "active"],
      ["figures", "done"],
      ["certify", "pending"],
      ["post", "pending"],
      ["submit", "pending"],
    ]);
    expect(steps[0].detail).toBe("2 cases still accruing days");
  });

  it("asks for the figures before the signature, even with no summary at all", () => {
    const steps = certificationTrack({
      ...base,
      totals: totals(),
      summary: null,
      now: POST_FROM - 60 * DAY,
    });
    expect(steps[0].state).toBe("done");
    expect(steps[1]).toMatchObject({
      state: "active",
      detail: "Start the summary to record headcount and hours",
    });
    expect(steps[2].state).toBe("pending");
  });

  it("treats a summary with zero hours as figures not yet recorded", () => {
    const steps = certificationTrack({
      ...base,
      totals: totals(),
      summary: summary({ totalHoursWorked: 0 }),
      now: POST_FROM - 60 * DAY,
    });
    expect(steps[1]).toMatchObject({
      state: "active",
      detail: "Add the average headcount and the hours worked",
    });
  });

  it("waits on the posting window once certified, then counts the days left", () => {
    const certified = summary({
      status: "Certified",
      certifiedAt: Date.UTC(2027, 0, 20) / 1000,
    });
    const waiting = certificationTrack({
      ...base,
      totals: totals(),
      summary: certified,
      now: POST_FROM - 12 * DAY,
    });
    expect(waiting[2]).toMatchObject({ state: "done", detail: "Dana Reyes on 2027-01-20" });
    expect(waiting[3]).toMatchObject({
      state: "active",
      detail: "Window opens 2027-02-01, in 12 days",
    });

    const open = certificationTrack({
      ...base,
      totals: totals(),
      summary: certified,
      now: POST_THROUGH - 20 * DAY,
    });
    expect(open[3]).toMatchObject({
      state: "active",
      detail: "Window is open until 2027-04-30, 20 days left",
    });
  });

  it("closes the posting step only when the window has closed over a certified summary", () => {
    const now = POST_THROUGH + 5 * DAY;
    const uncertified = certificationTrack({ ...base, totals: totals(), summary: summary(), now });
    expect(uncertified[3].state).toBe("pending");

    const certified = certificationTrack({
      ...base,
      totals: totals(),
      summary: summary({ status: "Certified", submittedAt: Date.UTC(2027, 2, 1) / 1000 }),
      now,
    });
    expect(certified.map((step) => step.state)).toEqual(["done", "done", "done", "done", "done"]);
    expect(certified[4].detail).toBe("Sent 2027-03-01");
  });

  it("says a clean year still has to post its summary", () => {
    const steps = certificationTrack({
      ...base,
      totals: totals({ totalRecordableCases: 0 }),
      summary: null,
      now: POST_FROM - 60 * DAY,
    });
    expect(steps[0].detail).toBe(
      "Nothing recordable this year; the summary still has to be posted",
    );
  });
});

describe("certifyBlocker", () => {
  it("names the first reason the server would refuse", () => {
    expect(certifyBlocker(null, { openCases: 0 })).toBe("no-summary");
    expect(certifyBlocker(summary({ averageEmployees: 0 }), { openCases: 0 })).toBe("no-figures");
    expect(certifyBlocker(summary(), { openCases: 3 })).toBe("open-cases");
    expect(certifyBlocker(summary(), { openCases: 0 })).toBeNull();
    expect(certifyBlockerMessage("open-cases", 1)).toBe(
      "Close the 1 case still accruing days first",
    );
  });
});

describe("filterCases", () => {
  const rows = [
    entry({ id: "b", caseNumber: 2, status: "Open" }),
    entry({
      id: "c",
      caseNumber: 3,
      classification: "FirstAidOnly",
      recordable: false,
      logName: "Ben Cole",
      description: "Paper cut, bandaged",
    }),
    entry(),
  ];

  it("reads in case-number order and counts each filter", () => {
    expect(filterCases(rows, "all", "").map((row) => row.caseNumber)).toEqual([1, 2, 3]);
    expect(caseFilterCounts(rows)).toEqual({ all: 3, recordable: 2, open: 1, off: 1 });
  });

  it("keeps the log, the open cases, or the decisions not to record", () => {
    expect(filterCases(rows, "recordable", "").map((row) => row.id)).toEqual(["inj_1", "b"]);
    expect(filterCases(rows, "open", "").map((row) => row.id)).toEqual(["b"]);
    expect(filterCases(rows, "off", "").map((row) => row.id)).toEqual(["c"]);
  });

  it("searches the posted name, the description, where it happened and the number", () => {
    expect(filterCases(rows, "all", "ben").map((row) => row.id)).toEqual(["c"]);
    expect(filterCases(rows, "all", "trailer").map((row) => row.id)).toEqual(["inj_1", "b"]);
    expect(filterCases(rows, "all", "dock 4")).toHaveLength(3);
    expect(filterCases(rows, "all", "3").map((row) => row.id)).toEqual(["c"]);
  });

  it("does not find a privacy case by the name the log withholds", () => {
    const privacy = entry({ logName: "Privacy Case" });
    expect(matchesQuery(privacy, "ada")).toBe(false);
    expect(matchesQuery(privacy, "privacy")).toBe(true);
  });
});

describe("summaryCaption", () => {
  it("captions each year with what the summary has reached", () => {
    expect(summaryCaption(undefined)).toBe("No summary");
    expect(summaryCaption({ status: "Draft" })).toBe("Draft");
    expect(summaryCaption({ status: "Certified" })).toBe("Certified");
  });
});
