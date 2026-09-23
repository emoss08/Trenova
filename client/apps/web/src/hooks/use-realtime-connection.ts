import { useT } from "@trenova/shared/i18n/use-t";
import { downloadReportRun } from "@/hooks/use-reports";
import { APP_ENV } from "@trenova/shared/lib/constants";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useRealtimeStore, type RealtimeConnectionState } from "@/stores/realtime-store";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import {
  CORE_QUERY_KEYS,
  RESOURCE_EVENT_NAME,
  RESOURCE_QUERY_KEY_MAP,
  isBulkAction,
  parseInvalidationEvent,
  patchEntityInListRows,
  queryKeyPrefix,
  queryKeyRootId,
  resolveEntityID,
  shouldPatchEvent,
  type QueryKeyRoot,
  type ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import {
  handleShipmentCommentEvent,
  isShipmentCommentEvent,
} from "@/lib/shipment-comment-realtime";
import { isOtherUsersTurnEvent } from "@/lib/assistant-turn-realtime";
import { parseReplyReady } from "@/components/assistant/reply-ready";
import { announceReplyReady } from "@/components/assistant/reply-ready-toast";
import { releaseTurnReaders } from "@/components/assistant/turn-readers";

const COALESCE_DELAY_MS = 300;

function mapConnectionState(state: string): RealtimeConnectionState {
  if (state === "connected") return "connected";
  if (state === "initialized" || state === "connecting") return "connecting";
  return "disconnected";
}

export function useRealtimeConnection() {
  const t = useT();

  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useAuthStore((state) => state.user);
  const queryClient = useQueryClient();
  // Keyed by the root's identity rather than the root itself, because a root
  // is now either a string or a key prefix and two equal prefixes are two
  // different arrays.
  const pendingKeysRef = useRef<Map<string, QueryKeyRoot>>(new Map());
  // Held in a ref so a new navigate identity never tears down and rebuilds the
  // realtime subscription.
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  navigateRef.current = navigate;
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
      // The session is gone, and with it every reply the person had open:
      // a reader left behind would keep reattaching against a dead session.
      releaseTurnReaders();
      return;
    }

    useRealtimeStore.getState().setConnectionState("connecting");
    const pendingKeys = pendingKeysRef.current;
    const presenceChannelName = apiService.realtimeService.getUsersPresenceChannelName(
      user.currentOrganizationId,
      user.businessUnitId,
    );
    let disposed = false;

    const enqueueInvalidation = (roots: readonly QueryKeyRoot[]) => {
      roots.forEach((root) => pendingKeys.set(queryKeyRootId(root), root));
      if (flushTimeoutRef.current !== null) return;

      flushTimeoutRef.current = window.setTimeout(() => {
        const rootsToInvalidate = Array.from(pendingKeys.values());
        pendingKeys.clear();
        flushTimeoutRef.current = null;

        void Promise.all(
          rootsToInvalidate.map((root) =>
            queryClient.invalidateQueries({
              queryKey: queryKeyPrefix(root),
              refetchType: "all",
            }),
          ),
        );
      }, COALESCE_DELAY_MS);
    };

    const invalidateCoreKeys = () => {
      enqueueInvalidation(CORE_QUERY_KEYS);
    };

    const applyEntityPatch = (root: QueryKeyRoot, event: ResourceInvalidationEvent) => {
      const entityID = resolveEntityID(event);
      const entity = event.entity;
      if (!entityID || !entity) return false;

      let patched = false;
      queryClient.setQueriesData(
        { queryKey: queryKeyPrefix(root) },
        (current: unknown): unknown => {
          const result = patchEntityInListRows(current, event);
          patched = result.patched || patched;
          return result.data;
        },
      );

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
      if (disposed || client.getState() !== "connected") {
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

    const onConnectionState = (state: string) => {
      if (disposed) {
        return;
      }

      useRealtimeStore.getState().setConnectionState(mapConnectionState(state));

      if (state === "connected") {
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
              id?: string;
              targetUserId?: string | null;
              eventType?: string;
              title?: string;
              message?: string;
              data?: Record<string, unknown> | null;
            };

            const isForCurrentUser = !notif.targetUserId || notif.targetUserId === user.id;
            const replyReady = isForCurrentUser ? parseReplyReady(notif) : null;

            if (replyReady) {
              void announceReplyReady({
                reply: replyReady,
                notificationId: notif.id ?? null,
                queryClient,
                navigate: (to) => void navigateRef.current(to),
                t,
              });
            } else if (isForCurrentUser && notif.title) {
              const runId =
                notif.eventType === "report_run_completed" && typeof notif.data?.runId === "string"
                  ? notif.data.runId
                  : null;

              // A bulk billing transfer runs unattended, so its result has to
              // come back to whoever started it with a way into the report —
              // the shipments that did not transfer are their next piece of work.
              const transferRunId =
                notif.eventType?.startsWith("billing_transfer_run_") &&
                typeof notif.data?.runId === "string"
                  ? notif.data.runId
                  : null;

              toast.info(notif.title, {
                description: notif.message,
                action: runId
                  ? {
                      label: t("Download"),
                      onClick: () => downloadReportRun({ id: runId }),
                    }
                  : transferRunId
                    ? {
                        label: t("View report"),
                        onClick: () =>
                          navigateRef.current(
                            `/billing-queue?transferRun=${encodeURIComponent(transferRunId)}`,
                          ),
                      }
                    : undefined,
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

      if (isShipmentCommentEvent(evt)) {
        handleShipmentCommentEvent(queryClient, evt, user.id);
        return;
      }

      if (isOtherUsersTurnEvent(evt, user.id)) {
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
        queryKeys.forEach((root) => {
          patchedAny = applyEntityPatch(root, evt) || patchedAny;
        });

        if (!patchedAny) {
          enqueueInvalidation(queryKeys);
        }
        return;
      }

      enqueueInvalidation(queryKeys);
    };

    const onDebugConnectionState = (state: string) => {
      console.debug("[Trenova] realtime state:", state);
    };
    if (APP_ENV === "development") {
      client.connection.on(onDebugConnectionState);
    }
    client.connection.on(onConnectionState);
    dataEventsChannel.subscribe(onResourceEvent);
    onConnectionState(client.getState());

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
  }, [isAuthenticated, queryClient, user, t]);
}
