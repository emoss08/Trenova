import { realtimeClient, type RealtimePresenceMember } from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useEffect, useMemo, useState } from "react";
import { shipmentCommentsRealtime } from "@/lib/shipment-comment-realtime";

export interface ShipmentViewer {
  userId: string;
  name: string;
}

/**
 * Everyone else with this shipment's comments open. Joining is what makes the
 * current user visible to them; the server ties the membership to this tab's
 * stream, so a closed tab leaves on its own.
 */
export function useShipmentViewers(shipmentId: string) {
  const userId = useAuthStore((s) => s.user?.id);
  const [members, setMembers] = useState<RealtimePresenceMember[]>([]);

  useEffect(() => {
    if (!shipmentId || !userId) return;

    const { scope, presencePath } = shipmentCommentsRealtime(shipmentId);
    const unsubscribe = realtimeClient.subscribePresence(scope, setMembers);
    const leave = realtimeClient.joinScope(scope, presencePath);

    return () => {
      unsubscribe();
      leave();
      setMembers([]);
    };
  }, [shipmentId, userId]);

  const viewers = useMemo<ShipmentViewer[]>(() => {
    // One entry per person however many tabs they have open, named by
    // whichever tab knows their name.
    const names = new Map<string, string>();
    for (const member of members) {
      if (member.userId === userId) continue;
      if (!names.get(member.userId)) {
        names.set(member.userId, member.name);
      }
    }
    return Array.from(names, ([viewerId, name]) => ({
      userId: viewerId,
      name: name || "Someone",
    }));
  }, [members, userId]);

  return { viewers };
}
