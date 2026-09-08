/**
 * An employment cycle: everything between a hire (or rehire) and the
 * termination that ends it. Onboarding belongs to the event that opened the
 * cycle, offboarding to the one that closed it, and a checklist started by
 * hand belongs to whichever cycle was running when it began.
 *
 * Derived on read from the events and the checklists, because the server
 * owns both and a stored answer would drift from them.
 */

export type CycleEvent = { id: string; kind: string; effectiveAt: number };

export type CycleChecklist = {
  id: string;
  kind: string;
  status: string;
  startedAt: number;
  sourceEventId: string | null;
};

export type EmploymentCycle<T extends CycleChecklist = CycleChecklist> = {
  openedBy: CycleEvent | null;
  closedBy: CycleEvent | null;
  checklists: T[];
};

export type CycleStage = "onboarding" | "active" | "offboarding" | "left";

const OPENS_CYCLE = new Set(["Hired", "Rehired"]);
const CLOSES_CYCLE = new Set(["Terminated"]);

export function buildEmploymentCycles<T extends CycleChecklist>(
  events: readonly CycleEvent[],
  checklists: readonly T[],
): EmploymentCycle<T>[] {
  const ordered = [...events]
    .filter((event) => OPENS_CYCLE.has(event.kind) || CLOSES_CYCLE.has(event.kind))
    .sort((a, b) => a.effectiveAt - b.effectiveAt);

  const cycles: EmploymentCycle<T>[] = [];
  for (const event of ordered) {
    const current = cycles.at(-1);
    if (OPENS_CYCLE.has(event.kind)) {
      cycles.push({ openedBy: event, closedBy: null, checklists: [] });
    } else if (current && current.closedBy === null) {
      current.closedBy = event;
    } else {
      cycles.push({ openedBy: null, closedBy: event, checklists: [] });
    }
  }

  const byEvent = new Map<string, EmploymentCycle<T>>();
  for (const cycle of cycles) {
    if (cycle.openedBy) byEvent.set(cycle.openedBy.id, cycle);
    if (cycle.closedBy) byEvent.set(cycle.closedBy.id, cycle);
  }

  let undated: EmploymentCycle<T> | null = null;
  const sortedChecklists = [...checklists].sort((a, b) => a.startedAt - b.startedAt);
  for (const checklist of sortedChecklists) {
    const home =
      (checklist.sourceEventId ? byEvent.get(checklist.sourceEventId) : undefined) ??
      cycleRunningAt(cycles, checklist.startedAt);
    if (home) {
      home.checklists.push(checklist);
      continue;
    }
    undated ??= { openedBy: null, closedBy: null, checklists: [] };
    undated.checklists.push(checklist);
  }

  const result = cycles.reverse();
  if (undated) result.push(undated);
  return result;
}

function cycleRunningAt<T extends CycleChecklist>(
  cycles: readonly EmploymentCycle<T>[],
  at: number,
): EmploymentCycle<T> | undefined {
  let match: EmploymentCycle<T> | undefined;
  for (const cycle of cycles) {
    const opened = cycle.openedBy?.effectiveAt ?? Number.NEGATIVE_INFINITY;
    if (opened <= at) match = cycle;
  }
  return match;
}

/**
 * Where the employment stands. An open onboarding means they are still
 * arriving; an open offboarding means they are still leaving; a closed cycle
 * with nothing open means they have gone.
 */
export function cycleStage(cycle: EmploymentCycle): CycleStage {
  const open = (kind: string) =>
    cycle.checklists.some((row) => row.kind === kind && row.status === "Open");
  if (open("Offboarding")) return "offboarding";
  if (cycle.closedBy) return "left";
  if (open("Onboarding")) return "onboarding";
  return "active";
}

export const CYCLE_STAGE_LABELS: Record<CycleStage, string> = {
  onboarding: "Onboarding",
  active: "Active",
  offboarding: "Offboarding",
  left: "Left",
};

/** The steps of an employment, in the order they happen, for a stepper. */
export const CYCLE_STEPS: CycleStage[] = ["onboarding", "active", "offboarding", "left"];
