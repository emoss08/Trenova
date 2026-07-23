import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { useEffect, useMemo, useState } from "react";

interface PresenceEvent {
  clientId?: string | null;
  connectionId?: string | null;
  action?: string;
}

const upsertConnection = (
  previous: Map<string, Set<string>>,
  userID: string,
  connectionID: string,
) => {
  const next = new Map(previous);
  const existing = new Set(next.get(userID) ?? []);
  existing.add(connectionID);
  next.set(userID, existing);
  return next;
};

const removeConnection = (
  previous: Map<string, Set<string>>,
  userID: string,
  connectionID: string,
) => {
  const next = new Map(previous);
  const existing = new Set(next.get(userID) ?? []);
  existing.delete(connectionID);
  if (existing.size === 0) {
    next.delete(userID);
  } else {
    next.set(userID, existing);
  }
  return next;
};

export function useOnlineUsers() {
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const [connectionsByUser, setConnectionsByUser] = useState<
    Map<string, Set<string>>
  >(new Map());
  const hasTenantContext = Boolean(
    isAuthenticated && user?.currentOrganizationId && user.businessUnitId,
  );

  useEffect(() => {
    if (
      !user?.currentOrganizationId ||
      !user.businessUnitId ||
      !isAuthenticated
    ) {
      return;
    }

    const client = apiService.realtimeService.connect();
    const state = client.getState();
    if (state === "closing" || state === "closed") {
      return;
    }

    const channel = client.channels.get(
      apiService.realtimeService.getUsersPresenceChannelName(
        user.currentOrganizationId,
        user.businessUnitId,
      ),
    );

    const onPresenceEvent = (message: PresenceEvent) => {
      if (!message.clientId || !message.connectionId) return;

      setConnectionsByUser((previous) => {
        if (message.action === "leave" || message.action === "absent") {
          return removeConnection(
            previous,
            message.clientId as string,
            message.connectionId as string,
          );
        }

        return upsertConnection(
          previous,
          message.clientId as string,
          message.connectionId as string,
        );
      });
    };

    // Subscribing asks the server for an initial member snapshot (delivered as
    // enter events) followed by live enter/leave/update transitions.
    const unsubscribe = channel.presence.subscribe(onPresenceEvent);

    return () => {
      try {
        unsubscribe();
      } catch {
        // Ignore teardown races when channel/client is already disposed.
      }
      // Drop members from the previous tenant/channel so a switch starts clean
      // before the next subscription's snapshot repopulates.
      setConnectionsByUser(new Map());
    };
  }, [isAuthenticated, user?.businessUnitId, user?.currentOrganizationId]);

  const onlineUserIDs = useMemo(
    () =>
      hasTenantContext ? new Set(connectionsByUser.keys()) : new Set<string>(),
    [connectionsByUser, hasTenantContext],
  );

  return { onlineUserIDs };
}
