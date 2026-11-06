import type {
  DispatchBoardMove,
  DispatchCandidate,
  DispatchUncoveredReason,
} from "@/lib/graphql/dispatch-console";
import { defineLabels, type LabelMap } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { formatCurrencyWhole } from "@trenova/shared/lib/utils";

/** What each part of a pairing's cost is called, as the server's cost components name them. */
export const COST_COMPONENT_LABELS = defineLabels({
  emptyMiles: "Empty miles",
  lateness: "Lateness",
  idleWait: "Waiting",
  rest: "Rest before driving",
  homeTime: "Time away from home",
  fairness: "Miles above the fleet",
  customerExperience: "New to the customer",
  onTimeHistory: "On-time history",
  safety: "Safety history",
  acceptance: "Acceptance",
  trailerContinuity: "Trailer swap",
  haulFit: "Haul fit",
  timeOffBuffer: "Close to time off",
  endMarket: "Where they end up",
});

export type CostComponent = keyof typeof COST_COMPONENT_LABELS;

export function costComponentLabel(component: string): string {
  return component in COST_COMPONENT_LABELS
    ? COST_COMPONENT_LABELS[component as CostComponent]
    : component;
}

/**
 * Why a plan left a move uncovered, named so it reads as a heading over one move or over a
 * group of them. Typed by the schema's enum, so a reason the server adds fails to compile here.
 */
export const UNCOVERED_REASON_LABELS: LabelMap<Record<DispatchUncoveredReason, string>> =
  defineLabels({
    NoCandidate: "No eligible driver",
    Disqualified: "Nearest drivers disqualified",
    BeyondDeadhead: "Past the empty-mile limit",
    LocationUnknown: "No driver location known",
    NotWorthIt: "Not worth covering",
    Outbid: "Drivers worth more elsewhere",
  });

/** Cents as whole dollars, with a sign for a credit: "$142", "−$15". */
/**
 * What the move's last plan says the best driver for it pays in preferences. A candidate
 * costs the plan its own cost less this, which is the number the plan weighed against the
 * coverage price; zero for a move no plan has priced or that is already covered.
 */
export function preferenceFloorOf(move: DispatchBoardMove): number {
  return move.isCovered ? 0 : (move.planPrice?.preferenceFloorCents ?? 0);
}

export function formatCents(cents: number): string {
  const dollars = formatCurrencyWhole(Math.round(Math.abs(cents) / 100));
  return cents < 0 ? `−${dollars}` : dollars;
}

/** The fact behind a cost line, from the candidate it belongs to, when there is one to show. */
export function costLineFact(component: string, candidate: DispatchCandidate): string | null {
  switch (component) {
    case "emptyMiles":
      return candidate.deadheadMiles == null
        ? null
        : translate("{0} mi", Math.round(candidate.deadheadMiles).toLocaleString());
    case "lateness":
      return candidate.deliveryLateMinutes > 0
        ? translate(
            "{0} late to delivery",
            formatClockDurationMs(candidate.deliveryLateMinutes * 60_000),
          )
        : candidate.minutesOfSlack < 0
          ? translate(
              "{0} late to pickup",
              formatClockDurationMs(-candidate.minutesOfSlack * 60_000),
            )
          : null;
    case "rest":
      return candidate.restMs > 0 ? formatClockDurationMs(candidate.restMs) : null;
    default:
      return null;
  }
}
