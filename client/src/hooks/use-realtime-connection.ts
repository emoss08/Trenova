import { APP_ENV } from "@/lib/constants";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { useRealtimeStore, type RealtimeConnectionState } from "@/stores/realtime-store";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { toast } from "sonner";
import {
  CORE_QUERY_KEYS,
  RESOURCE_EVENT_NAME,
  RESOURCE_QUERY_KEY_MAP,
  isBulkAction,
  parseInvalidationEvent,
  patchEntityInListRows,
  resolveEntityID,
  shouldPatchEvent,
  type ResourceInvalidationEvent,
} from "./realtime-patching";

const COALESCE_DELAY_MS = 300;

function mapConnectionState(state: string): RealtimeConnectionState {
  if (state === "connected") return "connected";
  if (state === "initialized" || state === "connecting") return "connecting";
  return "disconnected";
}

export function useRealtimeConnection() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useAuthStore((state) => state.user);
  const queryClient = useQueryClient();
  const pendingKeysRef = useRef<Set<string>>(new Set());
  const flushTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    if (
      !isAuthenticated ||
      !user ||
      !user.id ||
      !user.currentOrganizationId ||
      !user.businessUnitId
    ) {
      apiService.realtimeService.safeClose();
      useRealtimeStore.getState().setConnectionState("disconnected");
      return;
    }

    useRealtimeStore.getState().setConnectionState("connecting");
    const pendingKeys = pendingKeysRef.current;
    const presenceChannelName = apiService.realtimeService.getUsersPresenceChannelName(
      user.currentOrganizationId,
      user.businessUnitId,
    );
    let disposed = false;

    const enqueueInvalidation = (queryKeys: string[]) => {
      queryKeys.forEach((queryKey) => pendingKeys.add(queryKey));
      if (flushTimeoutRef.current !== null) return;

      flushTimeoutRef.current = window.setTimeout(() => {
        const keysToInvalidate = Array.from(pendingKeys);
        pendingKeys.clear();
        flushTimeoutRef.current = null;

        void Promise.all(
          keysToInvalidate.map((queryKey) =>
            queryClient.invalidateQueries({
              queryKey: [queryKey],
              refetchType: "all",
            }),
          ),
        );
      }, COALESCE_DELAY_MS);
    };

    const invalidateCoreKeys = () => {
      enqueueInvalidation(CORE_QUERY_KEYS);
    };

    const applyEntityPatch = (queryKey: string, event: ResourceInvalidationEvent) => {
      const entityID = resolveEntityID(event);
      const entity = event.entity;
      if (!entityID || !entity) return false;

      let patched = false;
      queryClient.setQueriesData({ queryKey: [queryKey] }, (current: unknown): unknown => {
        const result = patchEntityInListRows(current, event);
        patched = result.patched || patched;
        return result.data;
      });

      return patched;
    };

    const client = apiService.realtimeService.connect();
    const presenceChannel = apiService.realtimeService.getChannel(presenceChannelName);
    const dataEventsChannel = apiService.realtimeService.getChannel(
      apiService.realtimeService.getDataEventsChannelName(
        user.currentOrganizationId,
        user.businessUnitId,
      ),
    );

    const enterPresence = async () => {
      if (disposed || client.connection.state !== "connected") {
        return;
      }

      try {
        await presenceChannel.presence.enter({
          userId: user.id,
          name: user.name,
          emailAddress: user.emailAddress,
        });
      } catch {
        // Ignore teardown/reconnect races; connection listener will retry when stable.
      }
    };

    const onConnectionState = () => {
      if (disposed) {
        return;
      }

      useRealtimeStore.getState().setConnectionState(mapConnectionState(client.connection.state));

      if (client.connection.state === "connected") {
        void enterPresence();
        invalidateCoreKeys();
      }
    };

    const onResourceEvent = (message: { name?: string; data?: unknown }) => {
      if (message.name === RESOURCE_EVENT_NAME) {
        const notifEvt = parseInvalidationEvent(message.data);
        if (
          notifEvt &&
          notifEvt.resource === "notifications" &&
          notifEvt.action === "created" &&
          notifEvt.entity
        ) {
          if (
            notifEvt.organizationId !== user.currentOrganizationId ||
            notifEvt.businessUnitId !== user.businessUnitId
          ) {
            // Ignore notifications from other tenants
          } else {
            const notif = notifEvt.entity as {
              targetUserId?: string | null;
              title?: string;
              message?: string;
            };

            const isForCurrentUser = !notif.targetUserId || notif.targetUserId === user.id;

            if (isForCurrentUser && notif.title) {
              toast.info(notif.title, {
                description: notif.message,
              });
            }
          }

          void queryClient.invalidateQueries({
            queryKey: queries.notification._def,
          });
        }
      }

      if (message.name !== RESOURCE_EVENT_NAME) return;

      const evt = parseInvalidationEvent(message.data);
      if (!evt) return;
      if (
        evt.organizationId !== user.currentOrganizationId ||
        evt.businessUnitId !== user.businessUnitId
      ) {
        return;
      }
      useRealtimeStore.getState().setLastEventAt(Date.now());

      const queryKeys = RESOURCE_QUERY_KEY_MAP[evt.resource] ?? [];
      if (queryKeys.length === 0) return;

      const action = evt.action ?? "";
      if (isBulkAction(action)) {
        enqueueInvalidation(queryKeys);
        return;
      }

      if (shouldPatchEvent(evt)) {
        let patchedAny = false;
        queryKeys.forEach((queryKey) => {
          patchedAny = applyEntityPatch(queryKey, evt) || patchedAny;
        });

        if (!patchedAny) {
          enqueueInvalidation(queryKeys);
        }
        return;
      }

      enqueueInvalidation(queryKeys);
    };

    const onDebugConnectionState = (stateChange: { current: string }) => {
      console.debug("[Trenova] realtime state:", stateChange.current);
    };
    if (APP_ENV === "development") {
      client.connection.on(onDebugConnectionState);
    }
    client.connection.on(onConnectionState);
    void dataEventsChannel.subscribe(onResourceEvent);
    onConnectionState();

    return () => {
      disposed = true;
      client.connection.off(onConnectionState);
      if (APP_ENV === "development") {
        client.connection.off(onDebugConnectionState);
      }
      dataEventsChannel.unsubscribe(onResourceEvent);
      void apiService.realtimeService.leavePresenceIfPossible(presenceChannelName);
      useRealtimeStore.getState().setConnectionState("disconnected");
      if (flushTimeoutRef.current !== null) {
        clearTimeout(flushTimeoutRef.current);
        flushTimeoutRef.current = null;
      }
      pendingKeys.clear();
    };
  }, [isAuthenticated, queryClient, user]);
}
