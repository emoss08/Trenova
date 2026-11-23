import { describe, expect, it } from "vitest";
import { buildEmploymentCycles, cycleStage } from "../checklist-cycles";

type Event = { id: string; kind: string; effectiveAt: number };
type Checklist = {
  id: string;
  kind: string;
  status: string;
  startedAt: number;
  sourceEventId: string | null;
};

const hired: Event = { id: "wee_hire", kind: "Hired", effectiveAt: 1_600_000_000 };
const terminated: Event = { id: "wee_term", kind: "Terminated", effectiveAt: 1_650_000_000 };
const rehired: Event = { id: "wee_rehire", kind: "Rehired", effectiveAt: 1_700_000_000 };

const onboarding1: Checklist = {
  id: "wcl_on1",
  kind: "Onboarding",
  status: "Completed",
  startedAt: 1_600_000_000,
  sourceEventId: "wee_hire",
};
const offboarding1: Checklist = {
  id: "wcl_off1",
  kind: "Offboarding",
  status: "Completed",
  startedAt: 1_650_000_000,
  sourceEventId: "wee_term",
};
const onboarding2: Checklist = {
  id: "wcl_on2",
  kind: "Onboarding",
  status: "Open",
  startedAt: 1_700_000_000,
  sourceEventId: "wee_rehire",
};
const audit: Checklist = {
  id: "wcl_audit",
  kind: "Custom",
  status: "Open",
  startedAt: 1_620_000_000,
  sourceEventId: null,
};

describe("buildEmploymentCycles", () => {
  // A worker who was hired, left and came back has had two employments, and
  // each one owns its own onboarding and offboarding. The newest is first
  // because it is the one anybody opening the tab is working on.
  it("splits the record into employments at every hire and termination, newest first", () => {
    const cycles = buildEmploymentCycles(
      [hired, terminated, rehired],
      [onboarding1, offboarding1, onboarding2, audit],
    );

    expect(cycles.map((cycle) => cycle.openedBy?.id)).toEqual(["wee_rehire", "wee_hire"]);
    expect(cycles[0].closedBy).toBeNull();
    expect(cycles[0].checklists.map((row) => row.id)).toEqual(["wcl_on2"]);
    expect(cycles[1].closedBy?.id).toBe("wee_term");
    expect(cycles[1].checklists.map((row) => row.id)).toEqual(["wcl_on1", "wcl_audit", "wcl_off1"]);
  });

  // A checklist started by hand carries no event. It belongs to whichever
  // employment was running when it started, not to a bucket of its own.
  it("files a hand-started checklist under the employment it started in", () => {
    const cycles = buildEmploymentCycles([hired, terminated, rehired], [audit]);

    expect(cycles[1].checklists.map((row) => row.id)).toEqual(["wcl_audit"]);
    expect(cycles[0].checklists).toEqual([]);
  });

  // The server records a hire for every worker, but a record imported from
  // elsewhere may not have one. Its checklists still need somewhere to live.
  it("keeps checklists with no employment event in a single undated cycle", () => {
    const cycles = buildEmploymentCycles([], [audit]);

    expect(cycles).toHaveLength(1);
    expect(cycles[0].openedBy).toBeNull();
    expect(cycles[0].checklists.map((row) => row.id)).toEqual(["wcl_audit"]);
  });

  it("returns nothing for a worker with no events and no checklists", () => {
    expect(buildEmploymentCycles([], [])).toEqual([]);
  });
});

describe("cycleStage", () => {
  it("reads the stage from what the employment has done so far", () => {
    expect(cycleStage({ openedBy: hired, closedBy: null, checklists: [onboarding2] })).toBe(
      "onboarding",
    );
    expect(cycleStage({ openedBy: hired, closedBy: null, checklists: [onboarding1] })).toBe(
      "active",
    );
    expect(
      cycleStage({
        openedBy: hired,
        closedBy: terminated,
        checklists: [onboarding1, { ...offboarding1, status: "Open" }],
      }),
    ).toBe("offboarding");
    expect(
      cycleStage({
        openedBy: hired,
        closedBy: terminated,
        checklists: [onboarding1, offboarding1],
      }),
    ).toBe("left");
  });

  // An onboarding that was cancelled is not an onboarding in progress, and an
  // employment with no onboarding at all is simply active.
  it("treats a cancelled or absent onboarding as settled", () => {
    expect(
      cycleStage({
        openedBy: hired,
        closedBy: null,
        checklists: [{ ...onboarding1, status: "Cancelled" }],
      }),
    ).toBe("active");
    expect(cycleStage({ openedBy: hired, closedBy: null, checklists: [] })).toBe("active");
  });
});
