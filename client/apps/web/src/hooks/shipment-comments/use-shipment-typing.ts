import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { apiService } from "@/services/api";

const TYPING_PUBLISH_INTERVAL_MS = 3_000;
const TYPING_EXPIRY_MS = 5_000;
const TYPING_PRUNE_INTERVAL_MS = 1_000;
const TYPING_EVENT = "typing";

interface TypingPayload {
  userId?: string;
  name?: string;
  stop?: boolean;
}

export interface TypingUser {
  userId: string;
  name: string;
}

export function useShipmentTyping(shipmentId: string) {
  const user = useAuthStore((s) => s.user);
  const [typingByUser, setTypingByUser] = useState<
    Map<string, { name: string; expiresAt: number }>
  >(new Map());
  const lastPublishRef = useRef(0);

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

    const onTyping = (message: { name?: string; data?: unknown }) => {
      if (message.name !== TYPING_EVENT) return;
      const payload = message.data as TypingPayload | undefined;
      if (!payload?.userId || payload.userId === user.id) return;

      setTypingByUser((previous) => {
        const next = new Map(previous);
        if (payload.stop) {
          next.delete(payload.userId!);
        } else {
          next.set(payload.userId!, {
            name: payload.name ?? "Someone",
            expiresAt: Date.now() + TYPING_EXPIRY_MS,
          });
        }
        return next;
      });
    };

    channel.subscribe(onTyping);
    const pruneInterval = window.setInterval(() => {
      setTypingByUser((previous) => {
        const now = Date.now();
        let changed = false;
        const next = new Map(previous);
        for (const [userId, entry] of next) {
          if (entry.expiresAt <= now) {
            next.delete(userId);
            changed = true;
          }
        }
        return changed ? next : previous;
      });
    }, TYPING_PRUNE_INTERVAL_MS);

    return () => {
      try {
        channel.unsubscribe(onTyping);
      } catch {
        // Ignore teardown races when the channel is already disposed.
      }
      window.clearInterval(pruneInterval);
      setTypingByUser(new Map());
    };
  }, [channelName, user]);

  const publish = useCallback(
    (payload: TypingPayload) => {
      if (!channelName) return;
      if (!apiService.realtimeService.isConnectedOrConnecting()) return;
      try {
        void apiService.realtimeService.getChannel(channelName).publish(TYPING_EVENT, payload);
      } catch {
        // Typing signals are best-effort; drop silently on connection races.
      }
    },
    [channelName],
  );

  const publishTyping = useCallback(() => {
    if (!user) return;
    const now = Date.now();
    if (now - lastPublishRef.current < TYPING_PUBLISH_INTERVAL_MS) return;
    lastPublishRef.current = now;
    publish({ userId: user.id, name: user.name });
  }, [publish, user]);

  const publishStopTyping = useCallback(() => {
    if (!user) return;
    lastPublishRef.current = 0;
    publish({ userId: user.id, stop: true });
  }, [publish, user]);

  useEffect(() => {
    return () => {
      if (!user || !channelName) return;
      if (!apiService.realtimeService.isConnectedOrConnecting()) return;
      try {
        void apiService.realtimeService
          .getChannel(channelName)
          .publish(TYPING_EVENT, { userId: user.id, stop: true });
      } catch {
        // Best-effort stop signal on unmount.
      }
    };
  }, [channelName, user]);

  const typingUsers = useMemo<TypingUser[]>(
    () => Array.from(typingByUser, ([userId, entry]) => ({ userId, name: entry.name })),
    [typingByUser],
  );

  return { typingUsers, publishTyping, publishStopTyping };
}
