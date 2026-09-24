import { realtimeClient, type RealtimeTypingEvent } from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { shipmentCommentsRealtime } from "@/lib/shipment-comment-realtime";

const TYPING_PUBLISH_INTERVAL_MS = 3_000;
const TYPING_EXPIRY_MS = 5_000;
const TYPING_PRUNE_INTERVAL_MS = 1_000;
const TYPING_EVENT = "typing";

export interface TypingUser {
  userId: string;
  name: string;
}

/**
 * Who is typing in this shipment's comments, and a way to say the current user
 * is. The server stamps each signal with the sender's own identity and only
 * delivers it to tabs that have joined the thread, so this hook joins too.
 */
export function useShipmentTyping(shipmentId: string) {
  const userId = useAuthStore((s) => s.user?.id);
  const [typingByUser, setTypingByUser] = useState<
    Map<string, { name: string; expiresAt: number }>
  >(new Map());
  const lastPublishRef = useRef(0);

  const realtime = useMemo(
    () => (shipmentId ? shipmentCommentsRealtime(shipmentId) : null),
    [shipmentId],
  );

  useEffect(() => {
    if (!realtime || !userId) return;

    const leave = realtimeClient.joinScope(realtime.scope, realtime.presencePath);
    const unsubscribe = realtimeClient.on<RealtimeTypingEvent>(TYPING_EVENT, (event) => {
      if (event.scope !== realtime.scope || !event.userId || event.userId === userId) return;

      setTypingByUser((previous) => {
        const next = new Map(previous);
        if (event.stop) {
          next.delete(event.userId);
        } else {
          next.set(event.userId, {
            name: event.name || "Someone",
            expiresAt: Date.now() + TYPING_EXPIRY_MS,
          });
        }
        return next;
      });
    });

    const pruneInterval = window.setInterval(() => {
      setTypingByUser((previous) => {
        const now = Date.now();
        let changed = false;
        const next = new Map(previous);
        for (const [typingUserId, entry] of next) {
          if (entry.expiresAt <= now) {
            next.delete(typingUserId);
            changed = true;
          }
        }
        return changed ? next : previous;
      });
    }, TYPING_PRUNE_INTERVAL_MS);

    return () => {
      if (lastPublishRef.current !== 0) {
        realtimeClient.sendTyping(realtime.typingPath, true);
        lastPublishRef.current = 0;
      }
      unsubscribe();
      leave();
      window.clearInterval(pruneInterval);
      setTypingByUser(new Map());
    };
  }, [realtime, userId]);

  const publishTyping = useCallback(() => {
    if (!realtime || !userId) return;
    const now = Date.now();
    if (now - lastPublishRef.current < TYPING_PUBLISH_INTERVAL_MS) return;
    lastPublishRef.current = now;
    realtimeClient.sendTyping(realtime.typingPath, false);
  }, [realtime, userId]);

  const publishStopTyping = useCallback(() => {
    if (!realtime || !userId) return;
    lastPublishRef.current = 0;
    realtimeClient.sendTyping(realtime.typingPath, true);
  }, [realtime, userId]);

  const typingUsers = useMemo<TypingUser[]>(
    () =>
      Array.from(typingByUser, ([typingUserId, entry]) => ({
        userId: typingUserId,
        name: entry.name,
      })),
    [typingByUser],
  );

  return { typingUsers, publishTyping, publishStopTyping };
}
