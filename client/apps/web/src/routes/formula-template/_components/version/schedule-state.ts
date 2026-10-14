import type { FormulaTemplateVersion } from "@trenova/shared/types/formula-template";

/**
 * Where an approved snapshot's billing window stands. Pending: approved, but its
 * start has not come, so the date can move or be cleared to start now. Started:
 * it has billed shipments, so a new date approves the same content again as a
 * new version instead of rewriting history. Unapproved snapshots have no window.
 */
export type ScheduleState = "pending" | "started" | "unapproved";

export function scheduleState(
  version: Pick<FormulaTemplateVersion, "status" | "effectiveFrom">,
  nowUnix: number,
): ScheduleState {
  if (version.status !== "Active") return "unapproved";
  if (version.effectiveFrom != null && version.effectiveFrom > nowUnix) return "pending";
  return "started";
}
