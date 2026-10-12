import { usePresenceViewers, type PresenceViewer } from "@/hooks/use-presence-viewers";
import { shipmentCommentsRealtime } from "@/lib/shipment-comment-realtime";

export type ShipmentViewer = PresenceViewer;

/** Everyone else with this shipment's comments open. */
export function useShipmentViewers(shipmentId: string) {
  const target = shipmentId ? shipmentCommentsRealtime(shipmentId) : null;
  return usePresenceViewers(
    target ? { scope: target.scope, joinPath: target.presencePath } : null,
  );
}
