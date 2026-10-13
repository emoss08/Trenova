import { stopTypeLabel } from "@trenova/shared/lib/stop-type";
import type { Stop } from "@trenova/shared/types/shipment";

// A stop as a person reads it on the route: its position, what it is, and where.
export function stopRouteLabel(
  stop: Pick<Stop, "type" | "addressLine" | "location">,
  index: number,
): string {
  const place = stop.location?.name || stop.addressLine;
  const position = `${index + 1}. ${stopTypeLabel(stop.type)}`;
  return place ? `${position} · ${place}` : position;
}
