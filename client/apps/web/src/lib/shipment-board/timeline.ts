import type { Shipment } from "@trenova/shared/types/shipment";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";

export type TimelineRow = {
  shipment: Shipment;
  pickup: number;
  delivery: number;
  slipped: number | null;
};

/**
 * Rows keep the server's stage order (so grouping and the user's sort still
 * decide which loads come first) and run by pickup within a stage.
 */
export function buildTimelineRows(shipments: readonly Shipment[], dayStart: number): TimelineRow[] {
  const hourOf = (unix: number) => (unix - dayStart) / 3600;
  const stageOrder = new Map<string, number>();
  const rows: (TimelineRow & { order: number })[] = [];
  for (const shipment of shipments) {
    const stage = shipment.stage ?? "NeedsCoverage";
    if (!stageOrder.has(stage)) stageOrder.set(stage, stageOrder.size);
    const origin = getOriginStop(shipment);
    const destination = getDestinationStop(shipment);
    if (!origin?.scheduledWindowStart || !destination?.scheduledWindowStart) continue;
    const appointment = destination.scheduledWindowEnd ?? destination.scheduledWindowStart;
    const projected = shipment.eta?.estimatedArrival;
    rows.push({
      shipment,
      order: stageOrder.get(stage) ?? 0,
      pickup: hourOf(origin.scheduledWindowStart),
      delivery: hourOf(appointment),
      slipped:
        shipment.stage === "Late" && projected && projected > appointment
          ? hourOf(projected)
          : null,
    });
  }
  return rows.sort((a, b) => a.order - b.order || a.pickup - b.pickup);
}
