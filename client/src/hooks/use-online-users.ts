import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { useEffect, useMemo, useState } from "react";

interface PresenceMember {
  clientId?: string | null;
  connectionId?: string | null;
}

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
    if (
      client.connection.state === "closing" ||
      client.connection.state === "closed"
    ) {
      return;
    }

    const channel = client.channels.get(
      apiService.realtimeService.getUsersPresenceChannelName(
        user.currentOrganizationId,
        user.businessUnitId,
      ),
    );

    let cancelled = false;

    const syncPresence = async () => {
      try {
        if (
          cancelled ||
          client.connection.state !== "connected" ||
          (channel.state !== "attached" && channel.state !== "attaching")
        ) {
          return;
        }

        const members = await channel.presence.get();
        if (cancelled) return;

        const byUser = new Map<string, Set<string>>();
        members.forEach((member: PresenceMember) => {
          if (!member.clientId || !member.connectionId) return;
          const existing = byUser.get(member.clientId) ?? new Set<string>();
          existing.add(member.connectionId);
          byUser.set(member.clientId, existing);
        });

        setConnectionsByUser(byUser);
      } catch {
        if (!cancelled) {
          setConnectionsByUser(new Map());
        }
      }
    };

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

    const initialize = async () => {
      try {
        await channel.presence.subscribe(onPresenceEvent);
        await syncPresence();
      } catch {
        if (!cancelled) {
          setConnectionsByUser(new Map());
        }
      }
    };

    void initialize();

    return () => {
      cancelled = true;
      try {
        channel.presence.unsubscribe(onPresenceEvent);
      } catch {
        // Ignore teardown races when channel/client is already disposed.
      }
    };
  }, [isAuthenticated, user?.businessUnitId, user?.currentOrganizationId]);

  const onlineUserIDs = useMemo(
    () =>
      hasTenantContext ? new Set(connectionsByUser.keys()) : new Set<string>(),
    [connectionsByUser, hasTenantContext],
  );

  return { onlineUserIDs };
}
