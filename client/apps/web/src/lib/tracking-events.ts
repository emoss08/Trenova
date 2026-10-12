import type {
  ShipmentTrackingEventsQuery,
  TrackingEventOutcome,
} from "@trenova/graphql/generated/graphql";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";

export type TrackingEventRow = ShipmentTrackingEventsQuery["shipmentTrackingEvents"][number];

// A report that was used is complete; one waiting on an earlier stop is
// awaiting; a repeat or a report another one outranked is closed without having
// failed; one the load refused (a canceled load, an impossible time) failed.
export const TRACKING_OUTCOME_PHASES: Record<TrackingEventOutcome, StatusPhase> = {
  Applied: "complete",
  Pending: "awaiting",
  Duplicate: "closed",
  Superseded: "closed",
  Refused: "failed",
};

export function trackingOutcomePhase(outcome: TrackingEventOutcome): StatusPhase {
  return TRACKING_OUTCOME_PHASES[outcome];
}

export type TrackedStop = {
  id: string;
  moveSequence: number;
  sequence: number;
};

export type StopReportGroup = {
  stop: TrackedStop | null;
  stopId: string;
  events: TrackingEventRow[];
};

// Groups a shipment's reports under the stops they were tied to, in route order,
// each stop's reports oldest first. A report for a stop no longer on the
// shipment keeps its own group after the route so it is never silently dropped.
export function groupTrackingEventsByStop(
  events: readonly TrackingEventRow[],
  stops: readonly TrackedStop[],
): StopReportGroup[] {
  const byId = new Map(stops.map((stop) => [stop.id, stop]));
  const groups = new Map<string, StopReportGroup>();
  for (const event of events) {
    const group = groups.get(event.stopId) ?? {
      stop: byId.get(event.stopId) ?? null,
      stopId: event.stopId,
      events: [],
    };
    group.events.push(event);
    groups.set(event.stopId, group);
  }

  for (const group of groups.values()) {
    group.events.sort((a, b) => a.eventAt - b.eventAt || a.id.localeCompare(b.id));
  }

  return [...groups.values()].sort(compareGroups);
}

function compareGroups(a: StopReportGroup, b: StopReportGroup): number {
  if (a.stop && b.stop) {
    return a.stop.moveSequence - b.stop.moveSequence || a.stop.sequence - b.stop.sequence;
  }
  if (a.stop) {
    return -1;
  }
  if (b.stop) {
    return 1;
  }
  return a.stopId.localeCompare(b.stopId);
}
