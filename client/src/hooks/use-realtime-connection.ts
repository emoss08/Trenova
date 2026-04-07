import { APP_ENV } from "@/lib/constants";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { toast } from "sonner";

const RESOURCE_EVENT_NAME = "resource.invalidation";
const COALESCE_DELAY_MS = 300;

const RESOURCE_QUERY_KEY_MAP: Record<string, string[]> = {
  shipments: ["shipment-list"],
  users: ["user-list"],
  customers: ["customer-list"],
  tractors: ["tractor-list"],
  trailers: ["trailer-list"],
  workers: ["worker-list"],
  "audit-logs": ["audit-entry-list"],
  billing_queue: ["billing-queue-list", "billingQueue"],
};

const PATCHABLE_FIELDS_BY_RESOURCE: Record<string, Set<string>> = {
  users: new Set([
    "status",
    "name",
    "emailAddress",
    "username",
    "thumbnailUrl",
    "lastLoginAt",
    "updatedAt",
  ]),
  customers: new Set(["status", "name", "code", "emailAddress", "updatedAt"]),
  tractors: new Set(["status", "code", "updatedAt"]),
  trailers: new Set(["status", "code", "updatedAt"]),
  workers: new Set(["status", "firstName", "lastName", "updatedAt"]),
};

const CORE_QUERY_KEYS = Array.from(new Set(Object.values(RESOURCE_QUERY_KEY_MAP).flat()));

interface ResourceInvalidationEvent {
  type?: string;
  organizationId: string;
  businessUnitId: string;
  resource: string;
  action?: string;
  fields?: string[];
  entityId?: string;
  recordId?: string;
  entity?: Record<string, unknown>;
}

const parseInvalidationEvent = (payload: unknown): ResourceInvalidationEvent | null => {
  if (!payload) return null;

  let data: unknown = payload;
  if (typeof payload === "string") {
    try {
      data = JSON.parse(payload);
    } catch {
      return null;
    }
  }

  if (
    typeof data !== "object" ||
    data === null ||
    !("organizationId" in data) ||
    !("businessUnitId" in data) ||
    !("resource" in data)
  ) {
    return null;
  }

  return data as ResourceInvalidationEvent;
};

const hasRowsShape = (value: unknown): value is { results: Record<string, unknown>[] } =>
  !!value &&
  typeof value === "object" &&
  "results" in value &&
  Array.isArray((value as { results: unknown[] }).results);

const isBulkAction = (action: string) => action.startsWith("bulk_");

const resolveEntityID = (event: ResourceInvalidationEvent) => {
  const fromEvent = event.entityId || event.recordId;
  if (fromEvent) return fromEvent;

  const entity = event.entity;
  if (!entity || typeof entity !== "object") return "";

  return typeof entity.id === "string" ? entity.id : "";
};

const shouldPatchEvent = (event: ResourceInvalidationEvent) => {
  const action = event.action ?? "";
  const entityID = resolveEntityID(event);
  if (action !== "updated" || !entityID || !event.entity) return false;

  const patchableFields = PATCHABLE_FIELDS_BY_RESOURCE[event.resource];
  if (!patchableFields) return false;

  if (!event.fields || event.fields.length === 0) {
    return true;
  }

  return event.fields.every((field) => patchableFields.has(field));
};

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
      return;
    }

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
        if (!hasRowsShape(current)) return current;

        const index = current.results.findIndex((row) => row.id === entityID);
        if (index < 0) {
          return current;
        }

        patched = true;
        const nextResults = [...current.results];
        nextResults[index] = {
          ...nextResults[index],
          ...entity,
        };

        return {
          ...current,
          results: nextResults,
        };
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

            const isForCurrentUser =
              !notif.targetUserId || notif.targetUserId === user.id;

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
      if (flushTimeoutRef.current !== null) {
        clearTimeout(flushTimeoutRef.current);
        flushTimeoutRef.current = null;
      }
      pendingKeys.clear();
    };
  }, [isAuthenticated, queryClient, user]);
}
