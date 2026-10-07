import type { AgentIconName } from "@/components/agent-identity/agent-identity";
import { cronRunsBetween, parseCronSchedule, wallClockAt } from "@/lib/cron";
import type { AutonomyTier, ToolCatalogEntry, TriggerMode } from "@/types/assistant";
import { agentFormDefaults, type AgentFormValues } from "../agent-form-schema";
import { effectiveTier, splitCoreTools } from "../tool-catalog";

export const BUILDER_STARTS = ["chat", "scheduled", "event", "blank"] as const;
export type BuilderStart = (typeof BUILDER_STARTS)[number];

const START_TRIGGER: Record<BuilderStart, TriggerMode> = {
  chat: "Chat",
  scheduled: "Scheduled",
  event: "Event",
  blank: "Chat",
};

const START_ICON: Record<BuilderStart, AgentIconName> = {
  chat: "bot",
  scheduled: "bell",
  event: "radar",
  blank: "bot",
};

/** A tool each start begins with, when the catalog offers it. */
const START_TOOLS: Record<BuilderStart, { name: string; tier: AutonomyTier }[]> = {
  chat: [],
  scheduled: [{ name: "schedule_report_email", tier: "Propose" }],
  event: [{ name: "flag_for_manual_review", tier: "AutoExecute" }],
  blank: [],
};

export const DEFAULT_SCHEDULE = "0 6 * * 1-5";
export const DEFAULT_INTERVAL_SECONDS = 300;

/**
 * What a new agent starts as: in shadow, asking a person before any change, with the
 * trigger and the one tool its start implies. Nothing is chosen that the catalog lacks.
 */
export function startValues(
  start: BuilderStart,
  context: { timezone: string; catalog: readonly ToolCatalogEntry[] },
): AgentFormValues {
  const trigger = START_TRIGGER[start];
  const offered = new Set(context.catalog.map((tool) => tool.name));
  const tools = START_TOOLS[start].filter((tool) => offered.has(tool.name));
  return {
    ...agentFormDefaults,
    icon: START_ICON[start],
    accent: "indigo",
    triggerMode: trigger,
    cronExpression: trigger === "Scheduled" ? DEFAULT_SCHEDULE : "",
    cronTimezone: trigger === "Scheduled" ? context.timezone : "",
    autonomyCeiling: "ActWithApproval",
    shadowMode: true,
    simulationMode: false,
    dailyRunLimit: 100,
    maxToolCalls: 10,
    toolNames: tools.map((tool) => tool.name),
    toolTiers: Object.fromEntries(tools.map((tool) => [tool.name, tool.tier])),
  };
}

export type TriggerProblem = "events" | "schedule" | "badSchedule" | "interval" | null;

/** Why the agent's trigger cannot be saved as it stands, or null. */
export function triggerProblem(
  values: Pick<AgentFormValues, "triggerMode" | "cronExpression" | "eventKinds" | "intervalSeconds">,
): TriggerProblem {
  switch (values.triggerMode) {
    case "Event":
      return values.eventKinds.length ? null : "events";
    case "Scheduled":
      if (!values.cronExpression.trim()) return "schedule";
      return parseCronSchedule(values.cronExpression) ? null : "badSchedule";
    case "Continuous":
      return values.intervalSeconds >= 60 ? null : "interval";
    default:
      return null;
  }
}

/** The tools a person chose, leaving out those every agent holds anyway. */
export function chosenTools(
  values: Pick<AgentFormValues, "toolNames">,
  catalog: readonly ToolCatalogEntry[],
): ToolCatalogEntry[] {
  const { selectable } = splitCoreTools(catalog);
  const held = new Set(values.toolNames);
  return selectable.filter((tool) => held.has(tool.name));
}

export type CheckStatus = "ok" | "warn" | "todo";
export type CheckKey = "who" | "instr" | "trig" | "tools" | "limits" | "team" | "test";

/** The parts that must be done before an agent is ready to go live. */
export const REQUIRED_CHECKS: readonly CheckKey[] = ["who", "instr", "trig", "tools", "test"];

/** Instructions shorter than this read as not yet written. */
const MIN_INSTRUCTIONS = 40;

export type CheckInput = {
  values: AgentFormValues;
  findings: number;
  openRisk: number;
  chosen: number;
  /** The draft as it was last tried, or null before it has been. */
  tried: string | null;
  draft: string;
};

export function checklist(input: CheckInput): Record<CheckKey, CheckStatus> {
  const { values } = input;
  return {
    who: values.name.trim() && values.description.trim() ? "ok" : "todo",
    instr:
      input.findings > 0
        ? "warn"
        : values.instructions.trim().length > MIN_INSTRUCTIONS
          ? "ok"
          : "todo",
    trig: triggerProblem(values) ? "todo" : "ok",
    tools: input.openRisk > 0 ? "warn" : input.chosen > 0 ? "ok" : "todo",
    limits: "ok",
    team: "ok",
    test: input.tried === null ? "todo" : input.tried === input.draft ? "ok" : "warn",
  };
}

export function readiness(status: Record<CheckKey, CheckStatus>): { done: number; of: number } {
  return {
    done: REQUIRED_CHECKS.filter((key) => status[key] === "ok").length,
    of: REQUIRED_CHECKS.length,
  };
}

/** Change tools by the tier they would run at, after the ceiling: propose, ask first, on its own. */
export function changeTiers(
  values: Pick<AgentFormValues, "toolNames" | "toolTiers" | "autonomyCeiling">,
  catalog: readonly ToolCatalogEntry[],
): { reads: number; byTier: Record<AutonomyTier, number> } {
  const byTier: Record<AutonomyTier, number> = { Propose: 0, ActWithApproval: 0, AutoExecute: 0 };
  let reads = 0;
  for (const tool of chosenTools(values, catalog)) {
    if (tool.kind === "query") {
      reads += 1;
      continue;
    }
    byTier[effectiveTier(tool.name, values.toolTiers, values.autonomyCeiling)] += 1;
  }
  return { reads, byTier };
}

export type WeekDay = {
  /** Unix second the day starts, in the agent's time zone. */
  start: number;
  weekday: number;
  today: boolean;
  runs: { hour: number; minute: number; past: boolean }[];
};

export type WeekStrip =
  | { kind: "unreadable" }
  | { kind: "quiet" }
  | { kind: "week"; days: WeekDay[]; upcoming: number; nowFraction: number };

const DAY = 86_400;

/**
 * The next seven days as the schedule fills them, today first, in the agent's time zone:
 * each run on its day, those already gone today marked, and how many are still to come.
 */
export function weekStrip(expression: string, timezone: string, now: number): WeekStrip {
  const schedule = parseCronSchedule(expression);
  if (!schedule) {
    return { kind: "unreadable" };
  }
  const wall = wallClockAt(now, timezone);
  const sinceMidnight = wall.hour * 3600 + wall.minute * 60 + (now % 60);
  const todayStart = now - sinceMidnight;
  const runs = cronRunsBetween(schedule, timezone, todayStart, todayStart + 7 * DAY);
  const upcoming = runs.filter((run) => run >= now).length;
  if (upcoming === 0) {
    return { kind: "quiet" };
  }

  const days: WeekDay[] = Array.from({ length: 7 }, (_, index) => {
    const start = todayStart + index * DAY;
    return { start, weekday: (wall.weekday + index) % 7, today: index === 0, runs: [] };
  });
  for (const run of runs) {
    const index = Math.min(6, Math.floor((run - todayStart) / DAY));
    const at = wallClockAt(run, timezone);
    days[index]?.runs.push({ hour: at.hour, minute: at.minute, past: run < now });
  }
  return { kind: "week", days, upcoming, nowFraction: sinceMidnight / DAY };
}

/** The first run at or after a moment, within the next eight days, or null. */
export function nextRun(expression: string, timezone: string, now: number): number | null {
  const schedule = parseCronSchedule(expression);
  if (!schedule) return null;
  return cronRunsBetween(schedule, timezone, now, now + 8 * DAY)[0] ?? null;
}

/** The last second of a calendar date ("2026-10-31") in a time zone. */
export function endOfDay(date: string, timezone: string): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!match) return null;
  const [, year, month, day] = match.map(Number) as [number, number, number, number];
  const guess = Date.UTC(year, month - 1, day, 23, 59, 59) / 1000;
  const wall = wallClockAt(guess, timezone);
  const shown = Date.UTC(wall.year, wall.month - 1, wall.day, wall.hour, wall.minute, 59) / 1000;
  return guess - (shown - guess);
}

/** The calendar date ("2026-10-31") a moment falls on in a time zone. */
export function dateIn(unix: number, timezone: string): string {
  const wall = wallClockAt(unix, timezone);
  return `${wall.year}-${String(wall.month).padStart(2, "0")}-${String(wall.day).padStart(2, "0")}`;
}
