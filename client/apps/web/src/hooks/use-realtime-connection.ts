import { leaveExpiredSession } from "@/lib/session-exit";
import type { RealtimeEventRouter } from "@/lib/realtime-events";
import { loadRealtimeEvents } from "@/lib/realtime-events-loader";
import { retryDelay } from "@/lib/query-retry";
import { useT } from "@trenova/shared/i18n/use-t";
import { APP_ENV } from "@trenova/shared/lib/constants";
import { reportFailure } from "@trenova/shared/lib/failure-reporter";
import {
  REALTIME_SESSION_ENDED_EVENT,
  realtimeClient,
  type RealtimeConnectionState,
} from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useRealtimeStore } from "@/stores/realtime-store";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useLayoutEffect, useRef } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  CORE_QUERY_KEYS,
  RESOURCE_EVENT_NAME,
  queryKeyPrefix,
  queryKeyRootId,
  type QueryKeyRoot,
} from "@trenova/shared/hooks/realtime-patching";
import { releaseTurnReaders } from "@/components/assistant/turn-readers";

const COALESCE_DELAY_MS = 300;
// A root refetched less than this long ago waits for the trailing flush, so a
// burst of events on one conversation is one refetch a second, not one per
// event; the first change still lands after COALESCE_DELAY_MS.
const MIN_REFETCH_INTERVAL_MS = 1000;
// Events held while the event reader loads. Past this the oldest go, and the reader's arrival
// refetches what every screen leans on instead, as after a gap in the stream.
const MAX_HELD_EVENTS = 200;

export function useRealtimeConnection() {
  const t = useT();

  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useAuthStore((state) => state.user);
  const queryClient = useQueryClient();
  // Keyed by the root's identity rather than the root itself, because a root
  // is now either a string or a key prefix and two equal prefixes are two
  // different arrays.
  const pendingKeysRef = useRef<Map<string, { root: QueryKeyRoot; activeOnly: boolean }>>(
    new Map(),
  );
  // Held in a ref so a new navigate identity never tears down and rebuilds the
  // realtime subscription.
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  useLayoutEffect(() => {
    navigateRef.current = navigate;
  });
  const flushTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    if (
      !isAuthenticated ||
      !user ||
      !user.id ||
      !user.currentOrganizationId ||
      !user.businessUnitId
    ) {
      realtimeClient.disconnect();
      useRealtimeStore.getState().setConnectionState("disconnected");
      // The session is gone, and with it every reply the person had open:
      // a reader left behind would keep reattaching against a dead session.
      releaseTurnReaders();
      return;
    }

    const pendingKeys = pendingKeysRef.current;
    const lastFlushedAt = new Map<string, number>();
    let disposed = false;

    // Refetches every pending root not refetched within the last
    // MIN_REFETCH_INTERVAL_MS, and comes back for the rest when their
    // interval is up.
    const flush = () => {
      flushTimeoutRef.current = null;
      const now = Date.now();
      const due: Array<{ root: QueryKeyRoot; activeOnly: boolean }> = [];
      let nextIn = Number.POSITIVE_INFINITY;
      pendingKeys.forEach((entry, id) => {
        const since = now - (lastFlushedAt.get(id) ?? Number.NEGATIVE_INFINITY);
        if (since >= MIN_REFETCH_INTERVAL_MS) {
          due.push(entry);
          pendingKeys.delete(id);
          lastFlushedAt.set(id, now);
          return;
        }
        nextIn = Math.min(nextIn, MIN_REFETCH_INTERVAL_MS - since);
      });
      if (pendingKeys.size > 0) {
        flushTimeoutRef.current = window.setTimeout(flush, nextIn);
      }

      void Promise.all(
        due.map(({ root, activeOnly: onScreen }) =>
          queryClient.invalidateQueries({
            queryKey: queryKeyPrefix(root),
            refetchType: onScreen ? "active" : "all",
          }),
        ),
      );
    };

    // A root asked for by any event in the window refetches everywhere
    // unless every one of those events wanted only what is on screen.
    const enqueueInvalidation = (roots: readonly QueryKeyRoot[], activeOnly = false) => {
      roots.forEach((root) => {
        const id = queryKeyRootId(root);
        const pending = pendingKeys.get(id);
        pendingKeys.set(id, { root, activeOnly: activeOnly && (pending?.activeOnly ?? true) });
      });
      if (flushTimeoutRef.current !== null) return;

      flushTimeoutRef.current = window.setTimeout(flush, COALESCE_DELAY_MS);
    };

    const invalidateCoreKeys = () => {
      enqueueInvalidation(CORE_QUERY_KEYS);
    };

    // A gap the server could not replay (the tab was away longer than the
    // stream keeps) means some change was missed: refetch what every screen
    // leans on rather than trust anything cached.
    const onReset = () => {
      if (!disposed) {
        invalidateCoreKeys();
      }
    };

    const onConnectionState = (state: RealtimeConnectionState) => {
      if (disposed) {
        return;
      }
      if (APP_ENV === "development") {
        console.debug("[Trenova] realtime state:", state);
      }
      useRealtimeStore.getState().setConnectionState(state);
    };

    // What an event changes (toasts, Triage, the watchtower, comments, refetches) is a module
    // of its own, loaded as the connection opens rather than with the first page. Events that
    // arrive before it are held and handled in order once it is in. A load that fails (a
    // network blip, or a release that replaced the file under an open tab) is retried with
    // backoff until it succeeds or the connection closes; a failed release file also reaches
    // chunk recovery (lib/chunk-recovery.ts) through Vite's preload error, which reloads into
    // the new build when that is safe.
    let events: RealtimeEventRouter | null = null;
    const held: unknown[] = [];
    let droppedWhileLoading = false;
    let failedLoads = 0;
    let retryTimeout: number | null = null;
    const identity = {
      id: user.id,
      currentOrganizationId: user.currentOrganizationId,
      businessUnitId: user.businessUnitId,
    };

    const loadEvents = () => {
      retryTimeout = null;
      void loadRealtimeEvents().then(
        ({ createRealtimeEventRouter }) => {
          if (disposed) return;
          const router = createRealtimeEventRouter({
            queryClient,
            user: identity,
            t,
            navigate: navigateRef,
            invalidate: enqueueInvalidation,
          });
          events = router;
          held.splice(0).forEach(router.handle);
          if (droppedWhileLoading) {
            onReset();
          }
        },
        (error: unknown) => {
          if (disposed) return;
          // Once per connection: a file that stays missing would otherwise report every retry.
          if (failedLoads === 0) {
            reportFailure(error, { source: "background", operation: "realtime events" });
          }
          retryTimeout = window.setTimeout(loadEvents, retryDelay(failedLoads));
          failedLoads += 1;
        },
      );
    };
    loadEvents();

    const onResourceEvent = (data: unknown) => {
      if (events) {
        events.handle(data);
        return;
      }
      held.push(data);
      if (held.length > MAX_HELD_EVENTS) {
        held.shift();
        droppedWhileLoading = true;
      }
    };

    const offState = realtimeClient.onStateChange(onConnectionState);
    const offResource = realtimeClient.on(RESOURCE_EVENT_NAME, onResourceEvent);
    const offReset = realtimeClient.on("reset", onReset);
    // The server closed the stream because its session ended (signed out elsewhere, revoked,
    // expired): the tab ends the session as it would for a 401.
    const offSessionEnded = realtimeClient.on(REALTIME_SESSION_ENDED_EVENT, leaveExpiredSession);
    realtimeClient.connect({
      identity: `${user.id}:${user.currentOrganizationId}:${user.businessUnitId}`,
      joinUsers: true,
    });
    onConnectionState(realtimeClient.getState());

    return () => {
      disposed = true;
      events?.dispose();
      events = null;
      held.length = 0;
      if (retryTimeout !== null) {
        clearTimeout(retryTimeout);
        retryTimeout = null;
      }
      offState();
      offResource();
      offReset();
      offSessionEnded();
      if (flushTimeoutRef.current !== null) {
        clearTimeout(flushTimeoutRef.current);
        flushTimeoutRef.current = null;
      }
      pendingKeys.clear();
    };
  }, [isAuthenticated, queryClient, user, t]);
}
