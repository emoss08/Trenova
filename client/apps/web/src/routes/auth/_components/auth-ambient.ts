/**
 * Copy for the sign-in panel.
 *
 * Everything the panel renders is real: the metrics and the lane chips both come from
 * the public /system/network-pulse endpoint, aggregated across every organization and
 * business unit on the instance, and the panel renders nothing where the endpoint is
 * disabled or has nothing to report. The only fixed strings left are the headline and
 * the status wording below.
 */

export const AUTH_PITCH = "Sign in once. The network never stopped moving.";

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
