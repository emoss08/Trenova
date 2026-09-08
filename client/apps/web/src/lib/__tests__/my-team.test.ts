import { describe, expect, it } from "vitest";
import type { ApprovalDelegationRow, TeamMemberRow } from "../graphql/org-structure";
import {
  attentionReasons,
  classifyMembers,
  coverSources,
  groupByTerminal,
  isWatched,
  matchesTeamSearch,
  needsAttention,
  recentStarters,
  summarizeTeam,
  upcomingAnniversaries,
  type ClassifiedMember,
} from "../my-team";

const DAY = 86_400;
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;
const ME = "usr_me";
const BOSS = "usr_boss";

function member(over: Partial<TeamMemberRow> = {}): TeamMemberRow {
  return {
    workerId: "wrk_1",
    name: "Ada Byrne",
    status: "Active",
    fleetCodeId: "fc_1",
    fleetCode: "SOUTH",
    fleetColor: "#2563eb",
    positionId: "jpos_1",
    positionTitle: "Over-the-Road Driver",
    direct: true,
    managerId: ME,
    complianceStatus: "Compliant",
    trainingHealth: "Current",
    safetyRating: "Excellent",
    hireDate: Date.UTC(2023, 10, 14) / 1000,
    terminationDate: null,
    ...over,
  };
}

function delegation(over: Partial<ApprovalDelegationRow> = {}): ApprovalDelegationRow {
  return {
    id: "dlg_1",
    delegatorId: BOSS,
    delegateId: ME,
    scope: "All",
    startsAt: NOW - 5 * DAY,
    endsAt: NOW + 5 * DAY,
    reason: null,
    revokedAt: null,
    version: 1,
    delegator: { id: BOSS, name: "Bea Boss" },
    delegate: { id: ME, name: "Me" },
    ...over,
  };
}

function classified(rows: TeamMemberRow[], covers = coverSources([], NOW)): ClassifiedMember[] {
  return classifyMembers(rows, ME, covers);
}

describe("coverSources", () => {
  // A delegation that has not started, has ended, or was called back puts
  // nobody on the team, whatever the server's row still says.
  it("keeps only delegations in force now", () => {
    const covers = coverSources(
      [
        delegation(),
        delegation({ id: "dlg_2", delegatorId: "usr_later", startsAt: NOW + DAY }),
        delegation({ id: "dlg_3", delegatorId: "usr_done", endsAt: NOW - DAY }),
        delegation({ id: "dlg_4", delegatorId: "usr_gone", revokedAt: NOW - 60 }),
      ],
      NOW,
    );
    expect(Array.from(covers.keys())).toEqual([BOSS]);
    expect(covers.get(BOSS)).toMatchObject({ name: "Bea Boss", scope: "All" });
  });

  it("keeps the first delegation when the same manager delegated twice", () => {
    const covers = coverSources(
      [delegation({ scope: "TimeOff" }), delegation({ id: "dlg_2", scope: "Expenses" })],
      NOW,
    );
    expect(covers.get(BOSS)?.scope).toBe("TimeOff");
  });
});

describe("classifyMembers", () => {
  it("separates the three doors onto the team", () => {
    const covers = coverSources([delegation()], NOW);
    const rows = classifyMembers(
      [
        member(),
        member({ workerId: "wrk_2", direct: false, managerId: null }),
        member({ workerId: "wrk_3", managerId: BOSS }),
      ],
      ME,
      covers,
    );
    expect(rows.map((row) => row.path)).toEqual(["direct", "terminal", "covering"]);
    expect(rows[2].coveringFor?.name).toBe("Bea Boss");
  });

  // The server flags a delegator's reports as direct. Without the delegation
  // list to explain that, the flag is the best answer there is.
  it("falls back to direct when the manager is unknown to the delegation list", () => {
    const rows = classified([member({ managerId: BOSS })]);
    expect(rows[0].path).toBe("direct");
    expect(rows[0].coveringFor).toBeNull();
  });

  it("never turns a terminal member into cover, even under a delegator", () => {
    const covers = coverSources([delegation()], NOW);
    const rows = classifyMembers([member({ direct: false, managerId: BOSS })], ME, covers);
    expect(rows[0].path).toBe("terminal");
  });
});

describe("attentionReasons", () => {
  it("lists critical reasons before watch reasons", () => {
    const reasons = attentionReasons(
      member({ complianceStatus: "Pending", trainingHealth: "Overdue", safetyRating: "Watch" }),
    );
    expect(reasons.map((reason) => `${reason.severity}:${reason.key}`)).toEqual([
      "critical:training",
      "watch:compliance",
      "watch:safety",
    ]);
  });

  it("treats every blocking training state as critical", () => {
    for (const health of ["Missing", "Failed", "Expired", "Overdue"]) {
      expect(needsAttention(member({ trainingHealth: health }))).toBe(true);
    }
    for (const health of ["DueSoon", "ExpiringSoon", "Scheduled", "Current"]) {
      expect(needsAttention(member({ trainingHealth: health }))).toBe(false);
    }
  });

  it("names the training state in the reason", () => {
    expect(attentionReasons(member({ trainingHealth: "Expired" }))[0].label).toBe(
      "Training expired",
    );
    expect(attentionReasons(member({ trainingHealth: "DueSoon" }))[0].label).toBe(
      "Training due soon",
    );
  });

  it("is watched only when every reason is a watch reason", () => {
    expect(isWatched(member({ safetyRating: "Watch" }))).toBe(true);
    expect(isWatched(member({ safetyRating: "Watch", complianceStatus: "NonCompliant" }))).toBe(
      false,
    );
    expect(isWatched(member())).toBe(false);
  });
});

describe("summarizeTeam", () => {
  it("counts paths, standing and reasons", () => {
    const covers = coverSources([delegation()], NOW);
    const rows = classifyMembers(
      [
        member(),
        member({ workerId: "wrk_2", direct: false, complianceStatus: "NonCompliant" }),
        member({ workerId: "wrk_3", managerId: BOSS, safetyRating: "Watch" }),
        member({
          workerId: "wrk_4",
          trainingHealth: "Overdue",
          safetyRating: "AtRisk",
        }),
      ],
      ME,
      covers,
    );
    const summary = summarizeTeam(rows, NOW);
    expect(summary).toMatchObject({
      total: 4,
      direct: 2,
      terminal: 1,
      covering: 1,
      goodStanding: 1,
      attention: 2,
      watching: 1,
      byReason: { compliance: 1, training: 1, safety: 1 },
    });
  });

  it("averages tenure over people with a hire date and names the longest serving", () => {
    const rows = classified([
      member({ hireDate: NOW - 100 * DAY }),
      member({ workerId: "wrk_2", name: "Old Hand", hireDate: NOW - 300 * DAY }),
      member({ workerId: "wrk_3", hireDate: 0 }),
    ]);
    const summary = summarizeTeam(rows, NOW);
    expect(summary.averageTenureDays).toBe(200);
    expect(summary.longestServing?.member.name).toBe("Old Hand");
    expect(summary.longestServing?.days).toBe(300);
  });

  it("stops a leaver's tenure at their termination date", () => {
    const rows = classified([
      member({
        status: "Terminated",
        hireDate: NOW - 400 * DAY,
        terminationDate: NOW - 100 * DAY,
      }),
    ]);
    expect(summarizeTeam(rows, NOW).averageTenureDays).toBe(300);
  });

  it("has no tenure when nobody has a hire date", () => {
    expect(summarizeTeam(classified([member({ hireDate: 0 })]), NOW).averageTenureDays).toBeNull();
  });
});

describe("groupByTerminal", () => {
  it("groups by terminal, biggest first, with the flagged count", () => {
    const groups = groupByTerminal(
      classified([
        member(),
        member({ workerId: "wrk_2", fleetCodeId: "fc_2", fleetCode: "NORTH", fleetColor: "#000" }),
        member({
          workerId: "wrk_3",
          fleetCodeId: "fc_2",
          fleetCode: "NORTH",
          fleetColor: "#000",
          complianceStatus: "NonCompliant",
        }),
      ]),
    );
    expect(groups.map((group) => [group.code, group.count, group.attention])).toEqual([
      ["NORTH", 2, 1],
      ["SOUTH", 1, 0],
    ]);
  });

  it("puts people without a terminal under one heading", () => {
    const groups = groupByTerminal(
      classified([
        member({ fleetCodeId: null, fleetCode: "", fleetColor: "" }),
        member({ workerId: "wrk_2", fleetCodeId: null, fleetCode: "", fleetColor: "" }),
      ]),
    );
    expect(groups).toEqual([
      { key: "none", code: "No terminal", color: null, count: 2, attention: 0 },
    ]);
  });
});

describe("upcomingAnniversaries", () => {
  it("finds anniversaries inside the window, soonest first", () => {
    const rows = classified([
      member({ name: "In five", hireDate: Date.UTC(2024, 8, 12) / 1000 }),
      member({ workerId: "wrk_2", name: "Today", hireDate: Date.UTC(2021, 8, 7) / 1000 }),
      member({ workerId: "wrk_3", name: "Too far", hireDate: Date.UTC(2020, 9, 20) / 1000 }),
    ]);
    const result = upcomingAnniversaries(rows, NOW);
    expect(result.map((item) => [item.member.name, item.years, item.inDays])).toEqual([
      ["Today", 5, 0],
      ["In five", 2, 5],
    ]);
  });

  it("wraps into next year when this year's date has passed", () => {
    const rows = classified([member({ hireDate: Date.UTC(2022, 0, 3) / 1000 })]);
    const result = upcomingAnniversaries(rows, Date.UTC(2026, 11, 20) / 1000);
    expect(result).toHaveLength(1);
    expect(result[0].years).toBe(5);
    expect(result[0].inDays).toBe(14);
  });

  it("ignores first-year starters and people who have left", () => {
    const rows = classified([
      member({ hireDate: Date.UTC(2026, 8, 10) / 1000 }),
      member({
        workerId: "wrk_2",
        status: "Terminated",
        hireDate: Date.UTC(2020, 8, 10) / 1000,
        terminationDate: NOW - DAY,
      }),
    ]);
    expect(upcomingAnniversaries(rows, NOW)).toEqual([]);
  });

  it("includes the last day of the window and not the one after", () => {
    const rows = classified([
      member({ hireDate: Date.UTC(2020, 9, 7) / 1000 }),
      member({ workerId: "wrk_2", hireDate: Date.UTC(2020, 9, 8) / 1000 }),
    ]);
    expect(upcomingAnniversaries(rows, NOW).map((item) => item.inDays)).toEqual([30]);
  });
});

describe("recentStarters", () => {
  it("lists active people who started inside the window, newest first", () => {
    const rows = classified([
      member({ name: "Last month", hireDate: NOW - 30 * DAY }),
      member({ workerId: "wrk_2", name: "Yesterday", hireDate: NOW - DAY }),
      member({ workerId: "wrk_3", name: "Long ago", hireDate: NOW - 91 * DAY }),
      member({
        workerId: "wrk_4",
        name: "Left",
        status: "Terminated",
        hireDate: NOW - 10 * DAY,
        terminationDate: NOW - DAY,
      }),
    ]);
    expect(recentStarters(rows, NOW).map((item) => [item.member.name, item.daysAgo])).toEqual([
      ["Yesterday", 1],
      ["Last month", 30],
    ]);
  });
});

describe("matchesTeamSearch", () => {
  it("matches name, title and terminal, case-insensitively", () => {
    const row = member();
    expect(matchesTeamSearch(row, "ada")).toBe(true);
    expect(matchesTeamSearch(row, "ROAD")).toBe(true);
    expect(matchesTeamSearch(row, "south")).toBe(true);
    expect(matchesTeamSearch(row, "north")).toBe(false);
  });

  it("matches everyone on a blank query", () => {
    expect(matchesTeamSearch(member(), "   ")).toBe(true);
  });
});
