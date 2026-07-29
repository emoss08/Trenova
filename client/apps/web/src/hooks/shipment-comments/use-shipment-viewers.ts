import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useEffect, useMemo, useState } from "react";
import { apiService } from "@/services/api";

interface PresenceEvent {
  clientId?: string | null;
  connectionId?: string | null;
  action?: string;
  data?: unknown;
}

export interface ShipmentViewer {
  userId: string;
  name: string;
}

export function useShipmentViewers(shipmentId: string) {
  const user = useAuthStore((s) => s.user);
  const [viewersByUser, setViewersByUser] = useState<
    Map<string, { name: string; connections: Set<string> }>
  >(new Map());

  const channelName = useMemo(() => {
    if (!user?.currentOrganizationId || !user.businessUnitId || !shipmentId) return null;
    return apiService.realtimeService.getTypingChannelName(
      user.currentOrganizationId,
      user.businessUnitId,
      "shipment",
      `${shipmentId}:comments`,
    );
  }, [user?.currentOrganizationId, user?.businessUnitId, shipmentId]);

  useEffect(() => {
    if (!channelName || !user) return;
    if (!apiService.realtimeService.isConnectedOrConnecting()) return;

    let channel: ReturnType<typeof apiService.realtimeService.getChannel>;
    try {
      channel = apiService.realtimeService.getChannel(channelName);
    } catch {
      return;
    }

    const onPresenceEvent = (message: PresenceEvent) => {
      if (!message.clientId || !message.connectionId) return;
      const rawName =
        typeof message.data === "object" && message.data !== null && "name" in message.data
          ? (message.data as { name?: unknown }).name
          : undefined;
      const name = typeof rawName === "string" && rawName !== "" ? rawName : "Someone";

      setViewersByUser((previous) => {
        const next = new Map(previous);
        const entry = next.get(message.clientId!) ?? { name, connections: new Set<string>() };
        if (message.action === "leave" || message.action === "absent") {
          entry.connections.delete(message.connectionId!);
          if (entry.connections.size === 0) {
            next.delete(message.clientId!);
          } else {
            next.set(message.clientId!, entry);
          }
        } else {
          entry.connections.add(message.connectionId!);
          entry.name = name === "Someone" ? entry.name : name;
          next.set(message.clientId!, { ...entry, connections: new Set(entry.connections) });
        }
        return next;
      });
    };

    const unsubscribe = channel.presence.subscribe(onPresenceEvent);
    const enter = async () => {
      try {
        await channel.presence.enter({ userId: user.id, name: user.name });
      } catch {
        // Ignore enter races during connect/teardown.
      }
    };
    void enter();

    return () => {
      try {
        unsubscribe();
      } catch {
        // Ignore teardown races.
      }
      try {
        void channel.presence.leave();
      } catch {
        // Ignore leave races when the connection already closed.
      }
      setViewersByUser(new Map());
    };
  }, [channelName, user]);

  const viewers = useMemo<ShipmentViewer[]>(
    () =>
      Array.from(viewersByUser, ([userId, entry]) => ({ userId, name: entry.name })).filter(
        (viewer) => viewer.userId !== user?.id,
      ),
    [viewersByUser, user?.id],
  );

  return { viewers };
}
