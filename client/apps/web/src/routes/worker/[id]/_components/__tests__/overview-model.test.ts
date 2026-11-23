import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  payRateText,
  workerStripItems,
  workerStripState,
  workerTabMarks,
  workerTenure,
} from "../overview-model";

const t = translate;

// Date-only values as the shared date field stores them: the start of that day on the reader's
// calendar (the test process's zone here), not UTC midnight.
const dayStart = (year: number, month: number, day: number) =>
  new Date(year, month - 1, day).getTime() / 1000;
const NOV_22_2026 = dayStart(2026, 11, 22);
const SEP_12_2022 = dayStart(2022, 9, 12);
const JAN_15_2027 = dayStart(2027, 1, 15);
const DEC_01_2026 = dayStart(2026, 12, 1);
const MAR_03_2026 = dayStart(2026, 3, 3);

type Concern = WorkerOverview["concerns"][number];

function concern(overrides: Partial<Concern>): Concern {
  return {
    severity: "Warning",
    code: "CredentialExpiring",
    subject: null,
    count: 1,
    occurredAt: null,
    dueAt: null,
    daysUntil: null,
    stopsAssignment: false,
    destination: { tab: "Qualification", view: "Credentials", record: null, recordId: null },
    ...overrides,
  };
}

/** Every section withheld: what a reader with only worker read, on a base plan, receives. */
function withheld(): WorkerOverview {
  return {
    asOf: NOV_22_2026,
    standing: "Good",
    concernCount: 0,
    concerns: [],
    clearance: null,
    worker: {
      id: "wrk_1",
      firstName: "Maria",
      lastName: "Alvarez",
      status: "Active",
      canBeAssigned: true,
      availableForDispatch: true,
      type: "Employee",
      driverType: "OTR",
      lastHomeAt: null,
      fleetCode: null,
      profile: null,
    },
    credentials: null,
    qualificationFile: null,
    drugAlcohol: null,
    training: null,
    safety: null,
    employment: null,
    checklist: null,
    pto: null,
    leave: null,
    openReview: null,
    lastReview: null,
    nextReviewAt: null,
    schedule: null,
    hos: null,
    pay: null,
    benefits: null,
    portal: null,
    nextTimeOff: null,
    documentCount: null,
    agentMemoryCount: null,
  };
}

/** The approved sample worker: every section present. */
function maria(): WorkerOverview {
  return {
    ...withheld(),
    standing: "Watch",
    credentials: {
      complianceStatus: "Pending",
      requiredCount: 8,
      validCount: 8,
      expiringCount: 1,
      expiredCount: 0,
      missingCount: 0,
    },
    qualificationFile: {
      complete: false,
      missingRequired: 0,
      expiringSoon: 0,
      expired: 0,
      outstanding: 1,
      safetyHistoryLate: true,
    },
    drugAlcohol: {
      nextQueryDaysUntil: 17,
      standing: {
        status: "Clear",
        returnToDuty: "NotRequired",
        lastClearinghouseQueryAt: null,
        nextClearinghouseQueryDue: null,
        openTestCount: 0,
      },
    },
    training: {
      compliant: true,
      requiredCount: 6,
      currentCount: 5,
      dueCount: 1,
      overdueCount: 0,
      expiringCount: 0,
      expiredCount: 0,
      missingCount: 0,
    },
    employment: { eventCount: 9, hiredAt: SEP_12_2022, openChecklistCount: 1 },
    checklist: {
      id: "wcl_1",
      name: "Onboarding",
      kind: "Onboarding",
      status: "Open",
      dueAt: null,
      progress: { requiredTotal: 8, requiredDone: 5, overdue: 0, percent: 62 },
    } as WorkerOverview["checklist"],
    pto: [
      {
        ptoType: "Sick",
        tracked: true,
        balanceDays: "3",
        pendingDays: "0",
        availableDays: "3",
      },
      {
        ptoType: "Vacation",
        tracked: true,
        balanceDays: "10",
        pendingDays: "0.5",
        availableDays: "9.5",
      },
    ],
    leave: { openCaseCount: 0, remainingHours: "480", totalHours: "480", exhausted: false },
    nextReviewAt: JAN_15_2027,
    schedule: { currentShiftName: null },
    hos: {
      link: "Reporting",
      dutyStatus: "driving",
      dutyStatusSince: null,
      driveRemainingMs: 22_320_000,
      driveLimitMs: null,
      shiftRemainingMs: null,
      shiftLimitMs: null,
      cycleRemainingMs: null,
      cycleLimitMs: null,
      breakRemainingMs: null,
      breakLimitMs: null,
      tractorCode: null,
      recordedAt: NOV_22_2026,
    },
    pay: {
      profileName: "OTR mileage",
      classification: "CompanyDriver",
      effectiveFrom: SEP_12_2022,
      headline: { kind: "Linehaul", method: "PerLoadedMile", rate: "0.62", overridden: true },
    } as WorkerOverview["pay"],
    benefits: { coveredCount: 2, notStartedCount: 0, employeeCostPerPeriodMinor: 4210 },
    portal: { access: "Linked", pendingProfileChangeCount: 1, lastSignInAt: null },
    nextTimeOff: null,
    documentCount: 14,
    agentMemoryCount: 4,
  };
}

describe("the section strips' state lines", () => {
  it("says how every view of the sample worker stands, in the approved words and tones", () => {
    const o = maria();
    const line = (view: Parameters<typeof workerStripState>[0]) => workerStripState(view, o, t);

    expect(line("history")).toEqual({ state: "9 events · hired Sep 12, 2022", tone: "none" });
    expect(line("checklists")).toEqual({ state: "1 open · 5 of 8 settled", tone: "none" });
    expect(line("credentials")).toEqual({
      state: "8 of 8 required on file · 1 expiring",
      tone: "warning",
    });
    expect(line("dq-file")).toEqual({ state: "Incomplete · 1 outstanding", tone: "danger" });
    expect(line("drug-and-alcohol")).toEqual({
      state: "Clear · query due in 17 days",
      tone: "warning",
    });
    expect(line("training")).toEqual({ state: "5 of 6 current · 1 due soon", tone: "warning" });
    expect(line("reviews")).toEqual({ state: "Next due Jan 15, 2027", tone: "none" });
    expect(line("hours-of-service").state).toMatch(/^Driving · 6 hr, 12 min drive left$/);
    expect(line("schedule")).toEqual({ state: "Not on a shift", tone: "none" });
    expect(line("time-off")).toEqual({ state: "9.5 vacation days available", tone: "none" });
    expect(line("leave")).toEqual({ state: "No open case", tone: "none" });
    expect(line("pay-profile")).toEqual({
      state: "OTR mileage · $0.62 a loaded mile",
      tone: "none",
    });
    expect(line("benefits")).toEqual({ state: "2 plans covered", tone: "none" });
    expect(line("documents")).toEqual({ state: "14 on file", tone: "none" });
    expect(line("driver-portal")).toEqual({ state: "Linked · 1 change waiting", tone: "warning" });
    expect(line("agent-memory")).toEqual({ state: "4 notes", tone: "none" });
  });

  it("says nothing for a section the plan or the reader's permissions withhold", () => {
    const o = withheld();
    const views = [
      "history",
      "checklists",
      "credentials",
      "dq-file",
      "drug-and-alcohol",
      "training",
      "hours-of-service",
      "schedule",
      "time-off",
      "leave",
      "pay-profile",
      "benefits",
      "documents",
      "driver-portal",
      "agent-memory",
    ] as const;

    for (const view of views) {
      expect(workerStripState(view, o, t), view).toEqual({ state: null, tone: "none" });
    }
  });

  it("reads no pay as no pay profile where the plan and permissions draw the view", () => {
    const o = maria();
    o.pay = null;

    expect(workerStripState("pay-profile", o, t, { drawable: true })).toEqual({
      state: "No pay profile",
      tone: "none",
    });
    expect(workerStripState("pay-profile", o, t)).toEqual({ state: null, tone: "none" });
    expect(workerStripItems(["pay-profile"], o, t)[0]?.state).toBe("No pay profile");
  });

  it("says nothing before the overview read answers", () => {
    expect(workerStripState("credentials", undefined, t)).toEqual({ state: null, tone: "none" });
  });

  it("names lapsed credentials and training in the danger tone", () => {
    const o = maria();
    o.credentials = { ...o.credentials!, expiredCount: 2, missingCount: 1 };
    o.training = { ...o.training!, overdueCount: 1, missingCount: 1 };

    expect(workerStripState("credentials", o, t)).toEqual({
      state: "7 of 8 required on file · 2 expired",
      tone: "danger",
    });
    expect(workerStripState("training", o, t)).toEqual({
      state: "5 of 6 current · 2 behind",
      tone: "danger",
    });
  });

  it("tells an overdue clearinghouse query and a prohibited driver apart", () => {
    const o = maria();
    o.drugAlcohol = { ...o.drugAlcohol!, nextQueryDaysUntil: -3 };
    expect(workerStripState("drug-and-alcohol", o, t)).toEqual({
      state: "Clear · query overdue by 3 days",
      tone: "danger",
    });

    o.drugAlcohol = {
      nextQueryDaysUntil: null,
      standing: { ...o.drugAlcohol.standing, status: "Prohibited" },
    };
    expect(workerStripState("drug-and-alcohol", o, t)).toEqual({
      state: "Prohibited",
      tone: "danger",
    });
  });

  it("tells a driver without telematics from one not linked and one with no logs", () => {
    const o = maria();
    o.hos = { ...o.hos!, link: "NotConnected" };
    expect(workerStripState("hours-of-service", o, t).state).toBe("Telematics not connected");
    o.hos = { ...o.hos, link: "NotLinked" };
    expect(workerStripState("hours-of-service", o, t).state).toBe(
      "Not linked to a telematics driver",
    );
    o.hos = { ...o.hos, link: "NoData" };
    expect(workerStripState("hours-of-service", o, t).state).toBe("No logs yet");
  });

  it("reads a review under way before the next one due, and an overdue one in danger", () => {
    const o = maria();
    o.nextReviewAt = DEC_01_2026 - 30 * 86_400;
    expect(workerStripState("reviews", o, t).tone).toBe("danger");

    o.openReview = {
      id: "rev_1",
      title: "2026 annual",
      status: "Draft",
      periodEnd: DEC_01_2026,
      overallScore: null,
    } as WorkerOverview["openReview"];
    expect(workerStripState("reviews", o, t)).toEqual({
      state: "Under way · period ends Dec 1, 2026",
      tone: "none",
    });
  });

  it("marks a portal with nothing waiting, and benefits not started", () => {
    const o = maria();
    o.portal = { access: "Invited", pendingProfileChangeCount: null, lastSignInAt: null };
    o.benefits = { coveredCount: 0, notStartedCount: 3, employeeCostPerPeriodMinor: 0 };

    expect(workerStripState("driver-portal", o, t)).toEqual({ state: "Invited", tone: "none" });
    expect(workerStripState("benefits", o, t)).toEqual({
      state: "3 plans not started",
      tone: "warning",
    });
  });

  it("builds a strip cell for each drawable view, labelled and with its line", () => {
    const items = workerStripItems(["documents", "agent-memory"], maria(), t);

    expect(items.map(({ value, label, state, tone }) => ({ value, label, state, tone }))).toEqual([
      { value: "documents", label: "Documents", state: "14 on file", tone: "none" },
      { value: "agent-memory", label: "Agent memory", state: "4 notes", tone: "none" },
    ]);
  });
});

describe("the pay rate", () => {
  it("is said the way the trade says it", () => {
    expect(payRateText("PerHour", "28.5", t)).toBe("$28.50 an hour");
    expect(payRateText("PercentOfRevenue", "22", t)).toBe("22% of revenue");
    expect(payRateText("FlatPerShipment", "150", t)).toBe("$150.00 a load");
  });
});

describe("the tab marks", () => {
  it("puts a red dot on a tab with something that needs action and an amber one otherwise", () => {
    const o = maria();
    o.concerns = [
      concern({
        severity: "Critical",
        destination: { tab: "Qualification", view: "DqFile", record: null, recordId: null },
      }),
      concern({ severity: "Warning" }),
      concern({
        severity: "Warning",
        destination: { tab: "Development", view: "Training", record: null, recordId: null },
      }),
      concern({
        severity: "Info",
        destination: { tab: "Safety", view: null, record: null, recordId: null },
      }),
    ];

    const marks = workerTabMarks(o, t);

    expect(marks.qualification).toEqual({
      kind: "dot",
      tone: "danger",
      label: "1 thing needs action",
    });
    expect(marks.development).toEqual({
      kind: "dot",
      tone: "warning",
      label: "1 thing is worth a look",
    });
    expect(marks.safety).toBeUndefined();
    expect(marks.files).toEqual({ kind: "count", value: 14 });
  });

  it("draws no count on Files when documents are withheld or there are none", () => {
    expect(workerTabMarks(withheld(), t).files).toBeUndefined();
    expect(workerTabMarks({ ...maria(), documentCount: 0 }, t).files).toBeUndefined();
    expect(workerTabMarks(undefined, t)).toEqual({});
  });
});

describe("the tenure on the header", () => {
  it("counts years and months from the hire date", () => {
    expect(workerTenure({ hireDate: SEP_12_2022, terminationDate: null }, NOV_22_2026, t)).toBe(
      "4 years 2 months in",
    );
  });

  it("says when a hire starts, and when a terminated worker left", () => {
    expect(workerTenure({ hireDate: DEC_01_2026, terminationDate: null }, NOV_22_2026, t)).toBe(
      "Starts Dec 1, 2026",
    );
    expect(
      workerTenure({ hireDate: SEP_12_2022, terminationDate: MAR_03_2026 }, NOV_22_2026, t),
    ).toBe("Left Mar 3, 2026");
  });

  it("says nothing without a hire date", () => {
    expect(workerTenure({ hireDate: null, terminationDate: null }, NOV_22_2026, t)).toBeNull();
  });
});
