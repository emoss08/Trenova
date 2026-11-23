import type { WorkerActivityEntry } from "@/lib/graphql/worker-activity";
import { dayKey } from "@/lib/day-grouping";
import type { WorkerArea, WorkerTarget } from "@/routes/worker/use-worker-state";
import type { FieldFilterInput } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { Resource } from "@trenova/shared/types/permission";

/** The kinds of change the filter offers, each the record kinds it covers. */
export const ACTIVITY_KINDS = {
  profile: [Resource.Worker, Resource.ProfileChangeRequest],
  employment: [Resource.WorkerEmploymentEvent, Resource.WorkerChecklist],
  qualification: [Resource.WorkerCredential, Resource.WorkerDOTTest, Resource.Qualification],
  development: [Resource.WorkerTraining, Resource.PerformanceReview],
  safety: [
    Resource.WorkerSafetyEvent,
    Resource.WorkerDisciplinaryAction,
    Resource.WorkerRecognition,
    Resource.WorkerInjury,
  ],
  time: [
    Resource.WorkerPTO,
    Resource.WorkerLeave,
    Resource.WorkerSchedule,
    Resource.ShiftSwap,
    Resource.Timesheet,
    Resource.WorkerDispatchHold,
  ],
  pay: [Resource.DriverPayProfile, Resource.BenefitPlan],
  files: [Resource.Document, Resource.AgentMemory, Resource.DriverPortal, Resource.WorkerPolicy],
} as const satisfies Record<string, readonly string[]>;

export type ActivityKind = keyof typeof ACTIVITY_KINDS;

export const ACTIVITY_KIND_LABELS: Record<ActivityKind, string> = defineLabels({
  profile: "Profile",
  employment: "Employment",
  qualification: "Qualification",
  development: "Development",
  safety: "Safety",
  time: "Time",
  pay: "Pay",
  files: "Files",
});

/** Who made a change: anyone, the reader, the driver from Dash, any agent, or one person. */
export type ActorFilter = "all" | "me" | "dash" | "agents" | `user:${string}`;

const AGENT_CHANNELS = ["agent_conversation", "agent_run"];

/** The connection input's field filters for the chosen actor and kind. */
export function activityFieldFilters(
  actor: ActorFilter,
  kind: ActivityKind | "all",
  currentUserId: string | null,
): FieldFilterInput[] {
  const filters: FieldFilterInput[] = [];
  if (actor === "me" && currentUserId) {
    filters.push({ field: "principalId", operator: "eq", value: currentUserId });
  } else if (actor === "dash") {
    filters.push({ field: "channel", operator: "eq", value: "portal" });
  } else if (actor === "agents") {
    filters.push({ field: "channel", operator: "in", value: AGENT_CHANNELS });
  } else if (actor.startsWith("user:")) {
    filters.push({ field: "principalId", operator: "eq", value: actor.slice("user:".length) });
  }
  if (kind !== "all") {
    filters.push({ field: "resource", operator: "in", value: [...ACTIVITY_KINDS[kind]] });
  }
  return filters;
}

export type ActivitySource = "web" | "dash" | "agent" | "integration" | "system" | "support";

export const ACTIVITY_SOURCE_LABELS: Record<ActivitySource, string> = defineLabels({
  web: "Web",
  dash: "Dash",
  agent: "Agent",
  integration: "Integration",
  system: "System",
  support: "Support",
});

/** Where an entry came from, by the channel it arrived through. */
export function activitySource(channel: string): ActivitySource {
  if (channel === "portal") return "dash";
  if (AGENT_CHANNELS.includes(channel)) return "agent";
  if (channel === "api" || channel === "scim") return "integration";
  if (channel === "support") return "support";
  if (channel === "job" || channel === "cli" || channel === "system") return "system";
  return "web";
}

/**
 * Where on this page an entry's record is: a sub-record opens in its tab with its record param,
 * a root change opens the tab it belongs to. A deleted record has nothing left to open.
 */
export function activityTarget(entry: WorkerActivityEntry): WorkerTarget | null {
  const id = entry.resourceId;
  const deleted = entry.operation.toLowerCase() === "delete";
  const record = deleted || !id ? undefined : id;
  switch (entry.resource) {
    case Resource.Worker:
      return { tab: "profile" };
    case Resource.ProfileChangeRequest:
      return { tab: "files", view: "driver-portal", request: record };
    case Resource.WorkerEmploymentEvent:
      return { tab: "employment", view: "history", event: record };
    case Resource.WorkerChecklist:
      return { tab: "employment", view: "checklists", checklist: record };
    case Resource.WorkerCredential:
      return { tab: "qualification", view: "credentials", credential: record };
    case Resource.WorkerDOTTest:
      return { tab: "qualification", view: "drug-and-alcohol", test: record };
    case Resource.WorkerTraining:
      return { tab: "development", view: "training", training: record };
    case Resource.PerformanceReview:
      return { tab: "development", view: "reviews", review: record };
    case Resource.WorkerSafetyEvent:
      return { tab: "safety", safetyEvent: record };
    case Resource.WorkerDisciplinaryAction:
      return { tab: "safety", action: record };
    case Resource.WorkerInjury:
      return { tab: "safety", injury: record };
    case Resource.WorkerRecognition:
      return { tab: "safety" };
    case Resource.WorkerPTO:
      return { tab: "time", view: "time-off", timeOff: record };
    case Resource.WorkerLeave:
      return { tab: "time", view: "leave", leave: record };
    case Resource.WorkerSchedule:
    case Resource.ShiftSwap:
      return { tab: "time", view: "schedule" };
    case Resource.DriverPayProfile:
      return { tab: "pay", view: "pay-profile" };
    case Resource.BenefitPlan:
      return { tab: "pay", view: "benefits" };
    case Resource.Document:
      return { tab: "files", view: "documents" };
    case Resource.AgentMemory:
      return { tab: "files", view: "agent-memory" };
    case Resource.DriverPortal:
    case Resource.WorkerPolicy:
      return { tab: "files", view: "driver-portal" };
    default:
      return null;
  }
}

/** The target only when its tab and view are drawn for this reader on this plan. */
export function drawableTarget(
  entry: WorkerActivityEntry,
  isDrawable: (area: WorkerArea) => boolean,
): WorkerTarget | null {
  const target = activityTarget(entry);
  if (!target) return null;
  return isDrawable({ tab: target.tab, view: target.view ?? null }) ? target : null;
}

export function actorName(entry: WorkerActivityEntry): string | null {
  return entry.actorName?.trim() || entry.user?.name?.trim() || null;
}

export type ActivityDay = { key: string; entries: WorkerActivityEntry[] };

/** Entries in order, cut into the local days they fall on. */
export function activityDays(
  entries: readonly WorkerActivityEntry[],
  timezone: string,
): ActivityDay[] {
  const days: ActivityDay[] = [];
  for (const entry of entries) {
    const key = dayKey(entry.timestamp, timezone);
    const last = days[days.length - 1];
    if (last && last.key === key) last.entries.push(entry);
    else days.push({ key, entries: [entry] });
  }
  return days;
}

/** The people seen in the loaded entries, for the actor filter, by name. */
export function activityPeople(
  entries: readonly WorkerActivityEntry[],
): { id: string; name: string }[] {
  const people = new Map<string, string>();
  for (const entry of entries) {
    if (entry.principalType !== "user" || !entry.principalId) continue;
    if (AGENT_CHANNELS.includes(entry.channel)) continue;
    const name = actorName(entry);
    if (name && !people.has(entry.principalId)) people.set(entry.principalId, name);
  }
  return [...people.entries()]
    .map(([id, name]) => ({ id, name }))
    .sort((a, b) => a.name.localeCompare(b.name));
}
