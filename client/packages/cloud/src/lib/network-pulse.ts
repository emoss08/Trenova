/**
 * Shipment status → chip wording. "Assigned" reads as loading to match the in-app
 * active-shipment breakdown, which already labels that bucket that way. An unmapped
 * status falls back to the raw value rather than being dropped, so a status added
 * server-side shows up as itself instead of silently vanishing from the band.
 */
const LANE_STATUS_LABELS: Record<string, string> = {
  New: "planned",
  PartiallyAssigned: "assigning",
  Assigned: "loading",
  InTransit: "in transit",
  Delayed: "delayed",
};

export function laneStatusLabel(status: string): string {
  return LANE_STATUS_LABELS[status] ?? status.toLowerCase();
}

export const NETWORK_PULSE_REFETCH_MS = 60_000;

// The band drifts continuously, so it needs enough chips to cover the track before the
// -50% translate wraps. A real instance running three lanes would otherwise show three
// chips and a long gap.
export const MIN_LANE_CHIPS = 8;
