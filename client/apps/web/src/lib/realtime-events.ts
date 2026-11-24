import { downloadReportRun } from "@/hooks/use-reports";
import { aiAuditExportNotice, downloadAIAuditExport } from "@/lib/ai-audit-exports";
import { isOtherUsersTurnEvent } from "@/lib/assistant-turn-realtime";
import {
  boundCaseRecordsIn,
  CASE_QUERY_ROOTS,
  mayTouchCase,
  touchesCase,
} from "@/lib/case-realtime";
import { deliverToDesktop } from "@/lib/desktop-notifications";
import {
  billingTransferRunIdOf,
  reportRunIdOf,
  shouldToastNotification,
} from "@/lib/notification-toast";
import {
  handleShipmentCommentEvent,
  isShipmentCommentEvent,
} from "@/lib/shipment-comment-realtime";
import { cachedThreadLists } from "@/lib/thread-list";
import { createInboundMessagePatcher, isInboundMessageEvent } from "@/lib/triage/realtime";
import { announceUnlessWatched, billingTransferResultKey } from "@/lib/watched-results";
import { createWatchtowerPatcher, isWatchtowerEvent } from "@/lib/watchtower-realtime";
import { useRealtimeStore } from "@/stores/realtime-store";
import { useTriageStore } from "@/stores/triage-store";
import { parseReplyReady } from "@/components/assistant/reply-ready";
import { announceReplyReady } from "@/components/assistant/reply-ready-toast";
import { toast } from "@trenova/shared/components/ui/toast";
import {
  invalidationFor,
  parseInvalidationEvent,
  type QueryKeyRoot,
} from "@trenova/shared/hooks/realtime-patching";
import type { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { notification } from "@trenova/shared/lib/queries/notification";
import type { QueryClient } from "@tanstack/react-query";
import type { useNavigate } from "@tanstack/react-router";

export type RealtimeEventRouterOptions = {
  queryClient: QueryClient;
  user: { id: string; currentOrganizationId: string; businessUnitId: string };
  t: ReturnType<typeof useT>;
  /** Read at the moment of navigating, so a new navigate identity is always the one used. */
  navigate: { readonly current: ReturnType<typeof useNavigate> };
  invalidate: (roots: readonly QueryKeyRoot[], activeOnly?: boolean) => void;
};

export type RealtimeEventRouter = {
  handle: (data: unknown) => void;
  dispose: () => void;
};

/**
 * createRealtimeEventRouter turns each resource event into what it changes in the open tab:
 * a toast or a desktop notice for a notification, a patch to Triage, the watchtower queue or a
 * shipment's comments, and otherwise a throttled refetch of the queries the resource feeds.
 * It is loaded once the realtime connection opens, never before the first page renders.
 */
export function createRealtimeEventRouter({
  queryClient,
  user,
  t,
  navigate,
  invalidate,
}: RealtimeEventRouterOptions): RealtimeEventRouter {
  // A message that arrives while Triage is open joins its folder marked as
  // new; anywhere else it only moves the counts.
  const inboundMessages = createInboundMessagePatcher({
    queryClient,
    meId: user.id,
    onArrival: (message) => {
      const triage = useTriageStore.getState();
      if (triage.mounted === 0) {
        return;
      }
      triage.markArrival(message.id);
    },
  });

  // A watchtower item that changed is read once and put where it now
  // belongs in the queue, and a run that moved redraws the rows it works.
  // Its refetches share the connection's throttle.
  const watchtowerItems = createWatchtowerPatcher({ queryClient, invalidate });

  const handle = (data: unknown) => {
    const notifEvt = parseInvalidationEvent(data);
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
            navigate: (to) => void navigate.current({ to }),
            t,
          });
        } else if (isForCurrentUser && notif.title) {
          const title = notif.title;
          deliverToDesktop(notif, () => {
            const link = notif.data?.link;
            if (typeof link === "string" && link.startsWith("/")) {
              void navigate.current({ to: link });
            }
          });

          if (shouldToastNotification(notif)) {
            const runId = reportRunIdOf(notif);
            const transferRunId = billingTransferRunIdOf(notif);

            // An AI audit export is only ever downloaded by the person who
            // asked for it, which is who this notice is addressed to, so the
            // toast downloads it directly; a failed one opens the exports.
            const auditExport = aiAuditExportNotice(notif);
            const noticeLink =
              typeof notif.data?.link === "string" && notif.data.link.startsWith("/")
                ? notif.data.link
                : null;

            const announce = () =>
              toast.info(title, {
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
                          navigate.current({
                            to: "/billing/queue",
                            search: { transferRun: transferRunId },
                          }),
                      }
                    : auditExport?.kind === "download"
                      ? {
                          label: t("Download"),
                          onClick: () =>
                            void downloadAIAuditExport(auditExport.exportId).catch((error) => {
                              toast.error(t("The file could not be downloaded"), {
                                description: graphQLErrorMessage(error, t("Try again shortly.")),
                              });
                            }),
                        }
                      : auditExport
                        ? {
                            label: t("View exports"),
                            onClick: () => navigate.current({ to: auditExport.link }),
                          }
                        : noticeLink
                          ? {
                              label: t("Open"),
                              onClick: () => navigate.current({ to: noticeLink }),
                            }
                          : undefined,
              });

            // A transfer the person is watching in its dialog announces itself there; the
            // notice waits, and interrupts only if the dialog closes before showing it.
            if (transferRunId) {
              announceUnlessWatched(billingTransferResultKey(transferRunId), announce);
            } else {
              announce();
            }
          }
        }
      }

      void queryClient.invalidateQueries({
        queryKey: notification._def,
      });
    }

    // A notification that cleared itself (the work it pointed at was finished) leaves
    // every panel it was open in. One event covers everyone it reached, so it is batched
    // with whatever else arrives in the same moment.
    if (
      notifEvt?.resource === "notifications" &&
      notifEvt.action === "resolved" &&
      notifEvt.organizationId === user.currentOrganizationId &&
      notifEvt.businessUnitId === user.businessUnitId
    ) {
      invalidate([notification._def], true);
      return;
    }

    const evt = notifEvt;
    if (!evt) return;
    if (
      evt.organizationId !== user.currentOrganizationId ||
      evt.businessUnitId !== user.businessUnitId
    ) {
      return;
    }
    useRealtimeStore.getState().setLastEventAt(Date.now());

    // A change to the record a Desk case is about can settle the case or
    // tick its checklist, whatever else the event moves.
    if (mayTouchCase(evt) && touchesCase(evt, boundCaseRecordsIn(cachedThreadLists(queryClient)))) {
      invalidate(CASE_QUERY_ROOTS);
    }

    if (isInboundMessageEvent(evt)) {
      inboundMessages.handle(evt);
      return;
    }

    if (isWatchtowerEvent(evt)) {
      watchtowerItems.handle(evt);
      return;
    }
    watchtowerItems.handleRun(evt);

    if (isShipmentCommentEvent(evt)) {
      handleShipmentCommentEvent(queryClient, evt, user.id);
      return;
    }

    if (isOtherUsersTurnEvent(evt, user.id)) {
      return;
    }

    const { roots: queryKeys, activeOnly, activeRoots } = invalidationFor(evt);
    if (activeRoots.length > 0) {
      invalidate(activeRoots, true);
    }
    if (queryKeys.length === 0) return;
    invalidate(queryKeys, activeOnly);
  };

  return {
    handle,
    dispose: () => {
      inboundMessages.dispose();
      watchtowerItems.dispose();
    },
  };
}
