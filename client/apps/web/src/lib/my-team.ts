import type { ApprovalDelegationRow, TeamMemberRow } from "@/lib/graphql/org-structure";
import { delegationState } from "@trenova/shared/lib/org-structure";
import { tenureParts } from "@trenova/shared/lib/tenure";
import {
  complianceStatusMeta,
  safetyRatingMeta,
  trainingHealthMetaOf,
} from "@trenova/shared/lib/worker-health";

export type TeamPath = "direct" | "terminal" | "covering";

export const TEAM_PATH_LABELS: Record<TeamPath, string> = {
  direct: "Direct reports",
  terminal: "Through a terminal you run",
  covering: "Covering for",
};

export type CoverSource = {
  id: string;
  name: string;
  scope: string;
  startsAt: number;
  endsAt: number | null;
  reason: string | null;
};

export type ClassifiedMember = {
  member: TeamMemberRow;
  path: TeamPath;
  /** Set when the member is only on the team while covering for their manager. */
  coveringFor: CoverSource | null;
};

/**
 * The managers whose approvals are in the signed-in user's hands right now,
 * keyed by their user id. A delegation that starts later or has been called
 * back does not put anybody on the team, so it is left out here too.
 */
export function coverSources(
  delegations: readonly ApprovalDelegationRow[],
  now: number,
): Map<string, CoverSource> {
  const sources = new Map<string, CoverSource>();
  for (const delegation of delegations) {
    if (delegationState(delegation, now) !== "active") continue;
    if (sources.has(delegation.delegatorId)) continue;
    sources.set(delegation.delegatorId, {
      id: delegation.delegatorId,
      name: delegation.delegator?.name ?? "A manager",
      scope: delegation.scope,
      startsAt: delegation.startsAt,
      endsAt: delegation.endsAt ?? null,
      reason: delegation.reason ?? null,
    });
  }
  return sources;
}

/**
 * Which of the three doors somebody came through. The server's `direct` flag
 * covers anybody whose manager is the user *or* a delegator, so the manager
 * id is what separates the user's own reports from the ones they are minding.
 */
export function classifyMembers(
  members: readonly TeamMemberRow[],
  userId: string | null | undefined,
  covers: ReadonlyMap<string, CoverSource>,
): ClassifiedMember[] {
  return members.map((member) => {
    if (!member.direct) {
      return { member, path: "terminal", coveringFor: null };
    }
    const managerId = member.managerId ?? null;
    if (managerId && managerId !== userId) {
      const source = covers.get(managerId);
      if (source) return { member, path: "covering", coveringFor: source };
    }
    return { member, path: "direct", coveringFor: null };
  });
}

export type AttentionSeverity = "critical" | "watch";

export type AttentionReason = {
  key: "compliance" | "training" | "safety";
  label: string;
  severity: AttentionSeverity;
};

/**
 * Everything about a person a manager might have to act on, worst first. A
 * critical reason stops them working or should; a watch reason is the thing
 * that becomes critical if nobody looks at it.
 */
export function attentionReasons(member: TeamMemberRow): AttentionReason[] {
  const reasons: AttentionReason[] = [];

  if (member.complianceStatus === "NonCompliant") {
    reasons.push({ key: "compliance", label: "Non-compliant", severity: "critical" });
  } else if (member.complianceStatus === "Pending") {
    reasons.push({ key: "compliance", label: "Compliance pending", severity: "watch" });
  }

  const training = trainingHealthMetaOf(member.trainingHealth);
  if (training.blocks) {
    reasons.push({
      key: "training",
      label: `Training ${training.label.toLowerCase()}`,
      severity: "critical",
    });
  } else if (member.trainingHealth === "DueSoon" || member.trainingHealth === "ExpiringSoon") {
    reasons.push({
      key: "training",
      label: `Training ${training.label.toLowerCase()}`,
      severity: "watch",
    });
  }

  if (member.safetyRating === "AtRisk") {
    reasons.push({ key: "safety", label: "Safety at risk", severity: "critical" });
  } else if (member.safetyRating === "Watch") {
    reasons.push({ key: "safety", label: "Safety watch", severity: "watch" });
  }

  return reasons.sort((a, b) => severityRank(a.severity) - severityRank(b.severity));
}

function severityRank(severity: AttentionSeverity): number {
  return severity === "critical" ? 0 : 1;
}

export function needsAttention(member: TeamMemberRow): boolean {
  return attentionReasons(member).some((reason) => reason.severity === "critical");
}

export function isWatched(member: TeamMemberRow): boolean {
  const reasons = attentionReasons(member);
  return reasons.length > 0 && reasons.every((reason) => reason.severity === "watch");
}

export type TeamSummary = {
  total: number;
  direct: number;
  terminal: number;
  covering: number;
  goodStanding: number;
  attention: number;
  watching: number;
  byReason: Record<AttentionReason["key"], number>;
  averageTenureDays: number | null;
  longestServing: { member: TeamMemberRow; days: number } | null;
};

export function summarizeTeam(rows: readonly ClassifiedMember[], now: number): TeamSummary {
  const summary: TeamSummary = {
    total: rows.length,
    direct: 0,
    terminal: 0,
    covering: 0,
    goodStanding: 0,
    attention: 0,
    watching: 0,
    byReason: { compliance: 0, training: 0, safety: 0 },
    averageTenureDays: null,
    longestServing: null,
  };

  let tenureTotal = 0;
  let tenureCount = 0;

  for (const row of rows) {
    summary[row.path] += 1;

    const reasons = attentionReasons(row.member);
    const critical = reasons.filter((reason) => reason.severity === "critical");
    if (critical.length > 0) {
      summary.attention += 1;
      for (const reason of critical) summary.byReason[reason.key] += 1;
    } else if (reasons.length > 0) {
      summary.watching += 1;
    } else {
      summary.goodStanding += 1;
    }

    if (row.member.hireDate > 0) {
      const days = tenureParts(row.member.hireDate, row.member.terminationDate, now).totalDays;
      tenureTotal += days;
      tenureCount += 1;
      if (!summary.longestServing || days > summary.longestServing.days) {
        summary.longestServing = { member: row.member, days };
      }
    }
  }

  summary.averageTenureDays = tenureCount > 0 ? Math.round(tenureTotal / tenureCount) : null;
  return summary;
}

export type TerminalGroup = {
  key: string;
  code: string;
  color: string | null;
  count: number;
  attention: number;
};

export const NO_TERMINAL_KEY = "none";

/** The team by the terminal each person sits in, biggest first. */
export function groupByTerminal(rows: readonly ClassifiedMember[]): TerminalGroup[] {
  const groups = new Map<string, TerminalGroup>();
  for (const { member } of rows) {
    const key = member.fleetCodeId ?? NO_TERMINAL_KEY;
    let group = groups.get(key);
    if (!group) {
      group = {
        key,
        code: member.fleetCode || "No terminal",
        color: member.fleetColor || null,
        count: 0,
        attention: 0,
      };
      groups.set(key, group);
    }
    group.count += 1;
    if (needsAttention(member)) group.attention += 1;
  }
  return Array.from(groups.values()).sort(
    (a, b) => b.count - a.count || a.code.localeCompare(b.code),
  );
}

export type Anniversary = {
  member: TeamMemberRow;
  years: number;
  /** Midnight UTC of the day it falls on, as Unix seconds. */
  onDate: number;
  inDays: number;
};

const DAY_SECONDS = 86_400;

function utcMidnight(unixSeconds: number): number {
  const date = new Date(unixSeconds * 1000);
  return Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()) / 1000;
}

/**
 * Work anniversaries inside the window, soonest first. Only whole years count
 * and only for people still here: nobody congratulates a leaver, and a start
 * date this year is a starter, not an anniversary.
 */
export function upcomingAnniversaries(
  rows: readonly ClassifiedMember[],
  now: number,
  windowDays = 30,
): Anniversary[] {
  const today = utcMidnight(now);
  const todayDate = new Date(today * 1000);
  const result: Anniversary[] = [];

  for (const { member } of rows) {
    if (member.status !== "Active" || member.hireDate <= 0) continue;
    const hired = new Date(member.hireDate * 1000);
    let year = todayDate.getUTCFullYear();
    let onDate = Date.UTC(year, hired.getUTCMonth(), hired.getUTCDate()) / 1000;
    if (onDate < today) {
      year += 1;
      onDate = Date.UTC(year, hired.getUTCMonth(), hired.getUTCDate()) / 1000;
    }
    const years = year - hired.getUTCFullYear();
    if (years < 1) continue;
    const inDays = Math.round((onDate - today) / DAY_SECONDS);
    if (inDays > windowDays) continue;
    result.push({ member, years, onDate, inDays });
  }

  return result.sort((a, b) => a.inDays - b.inDays || a.member.name.localeCompare(b.member.name));
}

export type RecentStarter = { member: TeamMemberRow; daysAgo: number };

/** People who joined inside the window, newest first. */
export function recentStarters(
  rows: readonly ClassifiedMember[],
  now: number,
  windowDays = 90,
): RecentStarter[] {
  const today = utcMidnight(now);
  const result: RecentStarter[] = [];
  for (const { member } of rows) {
    if (member.status !== "Active" || member.hireDate <= 0) continue;
    const daysAgo = Math.round((today - utcMidnight(member.hireDate)) / DAY_SECONDS);
    if (daysAgo < 0 || daysAgo > windowDays) continue;
    result.push({ member, daysAgo });
  }
  return result.sort((a, b) => a.daysAgo - b.daysAgo || a.member.name.localeCompare(b.member.name));
}

/** Name, title or terminal code, matched loosely enough for a half-typed name. */
export function matchesTeamSearch(member: TeamMemberRow, query: string): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) return true;
  return [member.name, member.positionTitle, member.fleetCode].some((field) =>
    field.toLowerCase().includes(needle),
  );
}

export type MemberHealth = {
  compliance: ReturnType<typeof complianceStatusMeta>;
  training: ReturnType<typeof trainingHealthMetaOf>;
  safety: ReturnType<typeof safetyRatingMeta>;
};

export function memberHealth(member: TeamMemberRow): MemberHealth {
  return {
    compliance: complianceStatusMeta(member.complianceStatus),
    training: trainingHealthMetaOf(member.trainingHealth),
    safety: safetyRatingMeta(member.safetyRating),
  };
}
