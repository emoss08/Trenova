import { recordPath } from "@/config/record-links";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { useIdempotencyKey } from "@/hooks/use-idempotency-key";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import {
  assignInboundMessages,
  blockInboundSender,
  correctInboundClassification,
  createInboundLoad,
  createMyTriageView,
  createTriageRule,
  decideInboundStops,
  deleteMyTriageView,
  deleteTriageRule,
  discardInboundLoad,
  discardInboundReplyDraft,
  draftInboundLoad,
  draftInboundMany,
  draftInboundReply,
  setTriageRuleEnabled,
  fileInboundAttachment,
  fileInboundMany,
  flagInboundFact,
  handOffTriage,
  createInboundMailbox,
  setInboundMailboxSigningSecret,
  updateTriageRule,
  linkInboundMessage,
  logInboundException,
  readUnreadInboundMessages,
  renameMyTriageView,
  reopenInboundConflict,
  reopenInboundMessages,
  resolveInboundConflict,
  reviewInboundMessages,
  saveInboundReplyDraft,
  sendInboundMailboxTest,
  sendInboundReply,
  snoozeInboundMessages,
  stopWaitingForInboundReply,
  unfileInboundAttachment,
  unlinkInboundMessage,
  waitForInboundReply,
  wakeInboundMessage,
  type FileInboundAttachmentInput,
  type InboundClassification,
  type InboundLinkedRecord,
  type InboundMailbox,
  type InboundProvider,
  type TriageHandOff,
  type InboundMessageFilter,
  type InboundMessageDetail,
  type InboundMessageRow,
  type InboundReply,
  type InboundSnoozeEvent,
  type LinkInboundMessageInput,
  type LogInboundExceptionInput,
  type SendInboundReplyInput,
  type TriageConflictResolution,
  type TriageLoadFieldInput,
  type TriageRuleAction,
  type TriageRule,
  type TriageRuleInput,
  type TriageStopsDecision,
  type TriageUser,
  type TriageView,
} from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { inbox } from "@/lib/queries/inbox";
import { applyRows, folderMembership, knownRow, type MembershipContext } from "@/lib/triage/cache";
import {
  generatePostmarkCredentials,
  postmarkSecret,
  webhookUrl,
} from "@/routes/admin/inbound-mailboxes/_components/webhook-url";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import { useTriageStore } from "@/stores/triage-store";
import type { AssistantThread } from "@/types/assistant";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { toSentenceFragment } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { createContext, useContext, useMemo } from "react";
import { toast } from "sonner";
import { classificationLabel } from "./classification";
import { nextAfterLeaving } from "./keys";
import { tomorrowMorning } from "./snooze-times";
import { useTriageNavigation } from "./triage-navigation";

/** How long a row takes to leave the list before the cache lets it go. */
export const LEAVE_MS = 240;
const TOAST_MS = 2400;
const UNDO_TOAST_MS = 5000;
/** The bar a reading must clear for a source connected from Triage to act on its own. */
const CONNECT_MIN_CONFIDENCE = 0.8;

/** A snooze until a time, or until something happens on the message's shipment. */
export type SnoozeChoice =
  | { kind: "until"; until: number }
  | {
      kind: "event";
      event: InboundSnoozeEvent;
      documentTypeId?: string;
      documentTypeName?: string;
    };

/**
 * Everything a person does to one message or many, in one place. The
 * buttons, the keys and every context menu call these and nothing else calls
 * the mutations, so a key and a button cannot drift apart.
 *
 * A change that moves a message between folders is drawn at once: the row
 * slides out of a folder it no longer belongs in, the reader moves on to the
 * next message, the counts move, and a toast says what happened with an Undo
 * that puts it all back and asks the server to do the inverse. A refusal from
 * the server puts it back too.
 *
 * An action that returns a promise leaves a failure to its caller, which is
 * a form or a composer that shows it in place; every other action reports a
 * failure itself.
 */
export type TriageActions = {
  handle: (ids: readonly string[], options?: { note?: string; teach?: TriageRuleAction }) => void;
  ignore: (ids: readonly string[], options?: { note?: string }) => void;
  reopen: (ids: readonly string[]) => void;
  /** Marks everything waiting in a folder handled, as the folder's menu does. */
  handleFolder: (filter: InboundMessageFilter) => void;
  /** Snoozes everything waiting in a folder. */
  snoozeFolder: (filter: InboundMessageFilter, choice: SnoozeChoice) => void;
  snooze: (ids: readonly string[], choice: SnoozeChoice) => void;
  /** S and the bulk bar: until eight tomorrow morning, in the person's time zone. */
  snoozeUntilTomorrow: (ids: readonly string[]) => void;
  wake: (id: string) => void;
  /** Parks a message until the sender answers on its thread, or until `until`. */
  waitForReply: (id: string, until: number) => void;
  stopWaiting: (id: string) => void;
  assign: (ids: readonly string[], owner: TriageUser | null) => void;
  assignToMe: (ids: readonly string[]) => void;
  correctKind: (id: string, classification: InboundClassification) => void;
  flagFact: (id: string, key: string, label: string) => void;
  link: (id: string, input: LinkInboundMessageInput) => Promise<InboundMessageDetail>;
  unlink: (id: string, record: InboundLinkedRecord) => void;
  file: (messageId: string, input: FileInboundAttachmentInput) => Promise<InboundMessageDetail>;
  unfile: (messageId: string, attachmentId: string) => void;
  /**
   * Files every message's documents on its matched shipment. It reports a
   * refusal itself; the promise is for a caller that drew the change first
   * (the briefing's deck) and has to put its own state back.
   */
  fileMany: (ids: readonly string[]) => Promise<number>;
  draftReply: (id: string) => Promise<InboundReply>;
  saveReplyDraft: (id: string, subject: string, body: string) => Promise<InboundReply>;
  discardReplyDraft: (id: string) => void;
  /** Sends once even if retried: the same reply resent after no answer reuses its key. */
  sendReply: (input: SendInboundReplyInput) => Promise<InboundReply>;
  /** Has Nova draft each message's reply or load; reports a refusal itself, like `fileMany`. */
  draftMany: (ids: readonly string[]) => Promise<number>;
  draftLoad: (id: string) => Promise<InboundMessageDetail>;
  discardLoad: (id: string) => void;
  /** Books the load once even if retried. */
  createLoad: (id: string, fields: TriageLoadFieldInput[]) => Promise<InboundMessageDetail>;
  logException: (id: string, input: LogInboundExceptionInput) => void;
  blockSender: (id: string) => void;
  decideStops: (id: string, decision: TriageStopsDecision) => Promise<InboundMessageDetail>;
  resolveConflict: (id: string, conflictId: string, resolution: TriageConflictResolution) => void;
  reopenConflict: (id: string, conflictId: string) => void;
  /** A rule drafted from a message, paused until someone turns it on. */
  draftRule: (input: TriageRuleInput) => void;
  setRuleEnabled: (rule: TriageRule, enabled: boolean) => void;
  deleteRule: (rule: TriageRule) => void;
  /** Turns a filled-in rule on, as the add row and the briefing's suggestion do. */
  createRule: (input: TriageRuleInput) => Promise<TriageRule>;
  /** Saves a blank changed on a rule: the sentence changes at once and goes back on a refusal. */
  updateRule: (rule: TriageRule, input: TriageRuleInput) => void;
  /** Hands everything still waiting to the person on call; the briefing says so in place. */
  handOff: (to: TriageUser) => Promise<TriageHandOff>;
  readUnread: () => void;
  sendTestEmail: (mailbox: InboundMailbox) => void;
  /** Creates a source from the connect flow, which shows a refusal in place. */
  connectSource: (input: ConnectSourceInput) => Promise<ConnectedSource>;
  /** Sends Trenova's own test to a source; the connect flow waits for it and shows a refusal. */
  testSource: (mailbox: InboundMailbox) => Promise<InboundMailbox>;
  /** Opens a Desk conversation about a message with an agent, for the Desk hand-off. */
  startDeskConversation: (
    agentId: string,
    message: Pick<InboundMessageRow, "id" | "subject">,
  ) => Promise<AssistantThread>;
  createView: (name: string, query: string) => Promise<TriageView>;
  renameView: (view: TriageView, name: string) => void;
  deleteView: (view: TriageView) => void;
  copyLink: (id: string) => void;
  copyText: (text: string) => void;
  openInNewTab: (id: string) => void;
  /** Opens the Desk hand-off on the message and the steps still open on it. */
  askInDesk: (message: Pick<InboundMessageRow, "id"> & Partial<Pick<InboundMessageRow, "actions">>) => void;
};

/** A new source as the connect flow names it. */
export type ConnectSourceInput = {
  name: string;
  /** Where Trenova receives the forwarded mail. */
  address: string;
  /** The shared address people already send to, kept as the source's purpose. */
  watch: string;
  provider: InboundProvider;
};

/** The source made, and where its provider posts mail when the server said so. */
export type ConnectedSource = { mailbox: InboundMailbox; webhookUrl: string | null };

type RowPatch = (row: InboundMessageRow, context: MembershipContext) => InboundMessageRow;

type OptimisticRun<TResult> = {
  ids: readonly string[];
  patch: RowPatch;
  run: () => Promise<TResult>;
  rowsOf: (result: TResult) => readonly InboundMessageRow[];
  label: string;
  /** Puts it back on the server. Absent, the toast offers no Undo. */
  undo?: (
    result: TResult,
    before: ReadonlyMap<string, InboundMessageRow | undefined>,
  ) => Promise<readonly InboundMessageRow[]>;
};

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function settle(row: InboundMessageRow, status: "Actioned" | "Ignored"): InboundMessageRow {
  return { ...row, status, needsReview: false, snooze: null, awaitingReply: null, dueAt: null };
}

function refreshSummaries(queryClient: QueryClient) {
  void queryClient.invalidateQueries({ queryKey: inbox.counts().queryKey });
  void queryClient.invalidateQueries({ queryKey: inbox.views().queryKey });
  void queryClient.invalidateQueries({
    queryKey: inbox.briefing().queryKey,
    refetchType: "active",
  });
}

function messageUrl(id: string): string {
  return new URL(recordPath("inbound_message", id), window.location.origin).toString();
}

const TriageActionsContext = createContext<TriageActions | null>(null);

export const TriageActionsProvider = TriageActionsContext.Provider;

/** The page's actions. Anything under the Triage page reads them here. */
export function useTriageActions(): TriageActions {
  const actions = useContext(TriageActionsContext);
  if (actions === null) {
    throw new Error("useTriageActions is used outside the Triage page");
  }

  return actions;
}

/** Builds the actions once for the page; `useTriageActions` hands them out. */
export function useTriageActionsController(): TriageActions {
  const t = useT();
  const queryClient = useQueryClient();
  const navigation = useTriageNavigation();
  const timeZone = useUserTimezone();
  const reviewKey = useIdempotencyKey();
  const replyKey = useIdempotencyKey();
  const loadKey = useIdempotencyKey();
  const me = useAuthStore((state) => state.user);

  return useMemo<TriageActions>(() => {
    const meId = me?.id ?? null;
    const meUser: TriageUser | null =
      me === null || !me.id
        ? null
        : {
            id: me.id,
            name: me.name,
            emailAddress: me.emailAddress,
            thumbnailUrl: me.thumbnailUrl ?? "",
          };
    const context = (): MembershipContext => ({ meId, now: Math.floor(Date.now() / 1000) });
    const fail = (error: unknown) => handleMutationError({ error, resourceName: "Message" });

    const placeDetail = (detail: InboundMessageDetail) => {
      const before = knownRow(queryClient, detail.id);
      applyRows(queryClient, [detail], new Map([[detail.id, before]]), context());
      queryClient.setQueryData(inbox.message(detail.id).queryKey, detail);
      refreshSummaries(queryClient);
    };

    const announce = (label: string, onUndo?: () => void) =>
      toast.success(label, {
        duration: onUndo === undefined ? TOAST_MS : UNDO_TOAST_MS,
        action: onUndo === undefined ? undefined : { label: t("Undo"), onClick: onUndo },
      });

    async function optimistic<TResult>({
      ids,
      patch,
      run,
      rowsOf,
      label,
      undo,
    }: OptimisticRun<TResult>): Promise<void> {
      const ctx = context();
      const before = new Map(ids.map((id) => [id, knownRow(queryClient, id)] as const));
      const after = ids.flatMap((id) => {
        const row = before.get(id);
        return row === undefined ? [] : [patch(row, ctx)];
      });
      const afterById = new Map(after.map((row) => [row.id, row] as const));
      const filter = navigation.filter();
      const leaving =
        filter === null
          ? []
          : after
              .filter((row) => folderMembership(row, filter, ctx) === false)
              .map((row) => row.id);
      const nextOpen = nextAfterLeaving(
        navigation.orderedIds(),
        new Set(leaving),
        navigation.openId(),
      );

      const pending = run();
      pending.catch(() => undefined);

      if (leaving.length > 0) {
        useTriageStore.getState().markLeaving(leaving);
        await delay(LEAVE_MS);
      }
      applyRows(queryClient, after, before, ctx);
      useTriageStore.getState().clearLeaving(leaving);
      if (leaving.length > 0) {
        const selection = useTriageStore.getState().selected;
        if (leaving.some((id) => selection.has(id))) {
          useTriageStore
            .getState()
            .setSelected([...selection].filter((id) => !leaving.includes(id)));
        }
      }
      if (nextOpen !== navigation.openId()) {
        navigation.open(navigation.isWide() ? nextOpen : null);
      }

      const toastId = announce(
        label,
        undo === undefined
          ? undefined
          : () => {
              void (async () => {
                const current = new Map(ids.map((id) => [id, knownRow(queryClient, id)] as const));
                const restore = ids.flatMap((id) => {
                  const row = before.get(id);
                  return row === undefined ? [] : [row];
                });
                applyRows(queryClient, restore, current, context());
                try {
                  const result = await pending;
                  const rows = await undo(result, before);
                  const known = new Map(
                    rows.map((row) => [row.id, knownRow(queryClient, row.id)] as const),
                  );
                  applyRows(queryClient, rows, known, context());
                } catch (error) {
                  fail(error);
                  void queryClient.invalidateQueries({ queryKey: inbox._def });
                } finally {
                  refreshSummaries(queryClient);
                }
              })();
            },
      );

      try {
        const result = await pending;
        const rows = rowsOf(result);
        applyRows(
          queryClient,
          rows,
          new Map(
            rows.map((row) => [row.id, afterById.get(row.id) ?? knownRow(queryClient, row.id)]),
          ),
          context(),
        );
        refreshSummaries(queryClient);
      } catch (error) {
        toast.dismiss(toastId);
        const restore = ids.flatMap((id) => {
          const row = before.get(id);
          return row === undefined ? [] : [row];
        });
        applyRows(queryClient, restore, afterById, context());
        refreshSummaries(queryClient);
        fail(error);
      }
    }

    const bulkRows = (result: { messages: InboundMessageRow[] }) => result.messages;
    const detailRows = (result: InboundMessageDetail) => [result];

    const reviewed = (count: number, status: "Actioned" | "Ignored", ruleAdded: boolean) => {
      if (status === "Ignored") {
        return count === 1
          ? t("Ignored")
          : t("{0, plural, one {Ignored # message} other {Ignored # messages}}", count);
      }
      if (ruleAdded) {
        return t("Marked handled · rule added");
      }
      return count === 1
        ? t("Marked handled")
        : t("{0, plural, one {Marked # message handled} other {Marked # messages handled}}", count);
    };

    const review = (
      ids: readonly string[],
      status: "Actioned" | "Ignored",
      options: { note?: string; teach?: TriageRuleAction },
    ) => {
      const note = options.note?.trim() ?? "";
      const teach = ids.length === 1 ? (options.teach ?? null) : null;
      const input = { status, note: note === "" ? null : note, teach };
      void optimistic({
        ids,
        patch: (row) => settle(row, status),
        run: () =>
          reviewKey({ ids, input }, (idempotencyKey) =>
            reviewInboundMessages({ ids }, input, { idempotencyKey }),
          ),
        rowsOf: bulkRows,
        label: reviewed(ids.length, status, teach !== null),
        undo: async (result) => {
          const reopened = await reopenInboundMessages({ ids });
          if (result.ruleId !== null) {
            await deleteTriageRule(result.ruleId);
            void queryClient.invalidateQueries({ queryKey: inbox.rules().queryKey });
          }
          return reopened.messages;
        },
      });
    };

    const snoozeLabel = (count: number, choice: SnoozeChoice): string => {
      if (choice.kind === "event") {
        return count === 1
          ? t("Snoozed until it happens")
          : t("{0, plural, one {Snoozed # message} other {Snoozed # messages}}", count);
      }
      return count === 1
        ? t("Snoozed")
        : t("{0, plural, one {Snoozed # message} other {Snoozed # messages}}", count);
    };

    const snooze = (ids: readonly string[], choice: SnoozeChoice) => {
      const input =
        choice.kind === "until"
          ? { until: choice.until }
          : { event: choice.event, documentTypeId: choice.documentTypeId ?? null };
      void optimistic({
        ids,
        patch: (row, ctx) => ({
          ...row,
          snooze: {
            until: choice.kind === "until" ? choice.until : null,
            event: choice.kind === "event" ? choice.event : null,
            watchShipment: choice.kind === "event" ? row.matchedShipment : null,
            documentTypeId: choice.kind === "event" ? (choice.documentTypeId ?? null) : null,
            documentTypeName: choice.kind === "event" ? (choice.documentTypeName ?? null) : null,
            snoozedAt: ctx.now,
            snoozedBy: meUser,
          },
        }),
        run: () => snoozeInboundMessages({ ids }, input),
        rowsOf: bulkRows,
        label: snoozeLabel(ids.length, choice),
        undo: async () => {
          const woken = await Promise.all(ids.map((id) => wakeInboundMessage(id)));
          return woken;
        },
      });
    };

    const assign = (ids: readonly string[], owner: TriageUser | null) => {
      void optimistic({
        ids,
        patch: (row) => ({ ...row, ownerId: owner?.id ?? null, owner }),
        run: () => assignInboundMessages({ ids }, owner?.id ?? null),
        rowsOf: bulkRows,
        label:
          owner === null
            ? t("Unassigned")
            : owner.id === meId
              ? t("Assigned to you")
              : t("Assigned to {0}", owner.name),
        undo: async (_, before) => {
          const byOwner = new Map<string | null, string[]>();
          for (const id of ids) {
            const previous = before.get(id)?.ownerId ?? null;
            byOwner.set(previous, [...(byOwner.get(previous) ?? []), id]);
          }
          const results = await Promise.all(
            [...byOwner].map(([ownerId, group]) => assignInboundMessages({ ids: group }, ownerId)),
          );
          return results.flatMap((result) => result.messages);
        },
      });
    };

    const detailMutation = (
      run: () => Promise<InboundMessageDetail>,
      label?: string,
      report = true,
    ) =>
      run().then(
        (detail) => {
          placeDetail(detail);
          if (label !== undefined) {
            announce(label);
          }
          return detail;
        },
        (error: unknown) => {
          if (report) {
            fail(error);
          }
          throw error;
        },
      );

    const quiet = (promise: Promise<unknown>) => {
      promise.catch(() => undefined);
    };

    const updateReply = (messageId: string, reply: InboundReply) => {
      queryClient.setQueryData<InboundMessageDetail>(
        inbox.message(messageId).queryKey,
        (detail) => {
          if (detail === undefined) {
            return detail;
          }
          if (reply.status === "Draft") {
            return { ...detail, replyDraft: reply };
          }
          const replies = detail.replies.some((row) => row.id === reply.id)
            ? detail.replies.map((row) => (row.id === reply.id ? reply : row))
            : [...detail.replies, reply];
          return { ...detail, replies, replyDraft: null };
        },
      );
      void queryClient.invalidateQueries({ queryKey: inbox.message(messageId).queryKey });
    };

    const copy = (text: string, done: string) => {
      void navigator.clipboard.writeText(text).then(
        () => announce(done),
        () => toast.error(t("Could not copy to the clipboard")),
      );
    };

    const folderWrite = (
      run: () => Promise<{ messages: InboundMessageRow[]; affected: number }>,
      label: (count: number) => string,
      undo: (ids: string[]) => Promise<readonly InboundMessageRow[]>,
    ) => {
      void run().then((result) => {
        const known = new Map(
          result.messages.map((row) => [row.id, knownRow(queryClient, row.id)] as const),
        );
        applyRows(queryClient, result.messages, known, context());
        void queryClient.invalidateQueries({ queryKey: inbox.messages._def });
        refreshSummaries(queryClient);
        const ids = result.messages.map((row) => row.id);
        announce(label(result.affected), () => {
          void undo(ids).then((restored) => {
            applyRows(
              queryClient,
              restored,
              new Map(restored.map((row) => [row.id, knownRow(queryClient, row.id)] as const)),
              context(),
            );
            void queryClient.invalidateQueries({ queryKey: inbox.messages._def });
            refreshSummaries(queryClient);
          }, fail);
        });
      }, fail);
    };

    const waitingIn = (filter: InboundMessageFilter): InboundMessageFilter => ({
      ...filter,
      lane: "Waiting",
    });

    return {
      handleFolder: (filter) =>
        folderWrite(
          () =>
            reviewKey({ folder: filter }, (idempotencyKey) =>
              reviewInboundMessages(
                { folder: waitingIn(filter) },
                { status: "Actioned", note: null, teach: null },
                { idempotencyKey },
              ),
            ),
          (count) =>
            t(
              "{0, plural, one {Marked # message handled} other {Marked # messages handled}}",
              count,
            ),
          async (ids) => (await reopenInboundMessages({ ids })).messages,
        ),
      snoozeFolder: (filter, choice) =>
        folderWrite(
          () =>
            snoozeInboundMessages(
              { folder: waitingIn(filter) },
              choice.kind === "until"
                ? { until: choice.until }
                : { event: choice.event, documentTypeId: choice.documentTypeId ?? null },
            ),
          (count) => t("{0, plural, one {Snoozed # message} other {Snoozed # messages}}", count),
          (ids) => Promise.all(ids.map((id) => wakeInboundMessage(id))),
        ),
      handle: (ids, options = {}) => review(ids, "Actioned", options),
      ignore: (ids, options = {}) => review(ids, "Ignored", options),
      reopen: (ids) =>
        void optimistic({
          ids,
          patch: (row) => ({ ...row, status: "InReview", needsReview: true }),
          run: () => reopenInboundMessages({ ids }),
          rowsOf: bulkRows,
          label: t("Reopened"),
        }),
      snooze,
      snoozeUntilTomorrow: (ids) =>
        snooze(ids, { kind: "until", until: tomorrowMorning(context().now, timeZone) }),
      wake: (id) =>
        void optimistic({
          ids: [id],
          patch: (row) => ({ ...row, snooze: null }),
          run: () => wakeInboundMessage(id),
          rowsOf: detailRows,
          label: t("Back in Waiting on you"),
        }),
      waitForReply: (id, until) =>
        void optimistic({
          ids: [id],
          patch: (row, ctx) => ({ ...row, awaitingReply: { since: ctx.now, until } }),
          run: () => waitForInboundReply(id, until),
          rowsOf: detailRows,
          label: t("Parked in Waiting on them"),
          undo: async () => [await stopWaitingForInboundReply(id)],
        }),
      stopWaiting: (id) =>
        void optimistic({
          ids: [id],
          patch: (row) => ({ ...row, awaitingReply: null }),
          run: () => stopWaitingForInboundReply(id),
          rowsOf: detailRows,
          label: t("Back in Waiting on you"),
        }),
      assign,
      assignToMe: (ids) => {
        if (meUser !== null) {
          assign(ids, meUser);
        }
      },
      correctKind: (id, classification) =>
        void optimistic({
          ids: [id],
          patch: (row) => ({ ...row, classification }),
          run: () => correctInboundClassification(id, classification),
          rowsOf: detailRows,
          label: t(
            "Got it — Nova will read these from {0} as {1} from now on",
            knownRow(queryClient, id)?.senderDomain ?? "",
            toSentenceFragment(classificationLabel(t, classification)),
          ),
          undo: async (_, before) => {
            const previous = before.get(id)?.classification ?? null;
            return previous === null ? [] : [await correctInboundClassification(id, previous)];
          },
        }),
      flagFact: (id, key, label) =>
        quiet(
          detailMutation(
            () => flagInboundFact(id, key),
            t("Thanks — Nova will re-check {0} on messages like this", toSentenceFragment(label)),
          ),
        ),
      link: (id, input) => detailMutation(() => linkInboundMessage(id, input), t("Linked"), false),
      unlink: (id, record) =>
        quiet(detailMutation(() => unlinkInboundMessage(id, record), t("Unlinked"))),
      file: (_, input) => detailMutation(() => fileInboundAttachment(input), t("Filed"), false),
      unfile: (_, attachmentId) =>
        quiet(detailMutation(() => unfileInboundAttachment(attachmentId), t("Unfiled"))),
      fileMany: (ids) =>
        fileInboundMany(ids).then(
          (result) => {
            const known = new Map(
              result.messages.map((row) => [row.id, knownRow(queryClient, row.id)] as const),
            );
            applyRows(queryClient, result.messages, known, context());
            refreshSummaries(queryClient);
            announce(
              t("{0, plural, one {Filed # message} other {Filed # messages}}", result.affected),
            );
            return result.affected;
          },
          (error: unknown) => {
            fail(error);
            throw error;
          },
        ),
      draftReply: (id) =>
        draftInboundReply(id).then(
          (reply) => {
            updateReply(id, reply);
            return reply;
          },
          (error: unknown) => {
            throw error;
          },
        ),
      saveReplyDraft: (id, subject, body) =>
        saveInboundReplyDraft(id, subject, body).then(
          (reply) => {
            updateReply(id, reply);
            return reply;
          },
          (error: unknown) => {
            throw error;
          },
        ),
      discardReplyDraft: (id) => quiet(detailMutation(() => discardInboundReplyDraft(id))),
      sendReply: (input) =>
        replyKey(input, (idempotencyKey) => sendInboundReply(input, { idempotencyKey })).then(
          (reply) => {
            updateReply(String(input.messageId), reply);
            refreshSummaries(queryClient);
            if (reply.status !== "Failed") {
              announce(t("Sent"));
            }
            return reply;
          },
          (error: unknown) => {
            throw error;
          },
        ),
      draftMany: (ids) =>
        draftInboundMany(ids).then(
          (count) => {
            announce(
              t(
                "{0, plural, one {Nova is drafting # message} other {Nova is drafting # messages}}",
                count,
              ),
            );
            void queryClient.invalidateQueries({
              queryKey: inbox.briefing().queryKey,
              refetchType: "active",
            });
            return count;
          },
          (error: unknown) => {
            fail(error);
            throw error;
          },
        ),
      draftLoad: (id) => detailMutation(() => draftInboundLoad(id), undefined, false),
      discardLoad: (id) => quiet(detailMutation(() => discardInboundLoad(id))),
      createLoad: (id, fields) =>
        detailMutation(
          () =>
            loadKey({ id, fields }, (idempotencyKey) =>
              createInboundLoad(id, fields, { idempotencyKey }),
            ),
          t("Load created as a draft"),
          false,
        ),
      logException: (id, input) =>
        quiet(detailMutation(() => logInboundException(id, input), t("Exception logged"))),
      blockSender: (id) =>
        void optimistic({
          ids: [id],
          patch: (row) => settle(row, "Ignored"),
          run: () => blockInboundSender(id),
          rowsOf: detailRows,
          label: t("Sender blocked"),
        }),
      decideStops: (id, decision) =>
        detailMutation(() => decideInboundStops(id, decision), undefined, false),
      resolveConflict: (id, conflictId, resolution) =>
        quiet(detailMutation(() => resolveInboundConflict(id, conflictId, resolution), t("Done"))),
      reopenConflict: (id, conflictId) =>
        quiet(detailMutation(() => reopenInboundConflict(id, conflictId))),
      draftRule: (input) =>
        void createTriageRule({ ...input, enabled: false }).then((rule) => {
          void queryClient.invalidateQueries({ queryKey: inbox.rules().queryKey });
          announce(t("Rule drafted — paused until you turn it on"), () => {
            void deleteTriageRule(rule.id).then(
              () => queryClient.invalidateQueries({ queryKey: inbox.rules().queryKey }),
              fail,
            );
          });
        }, fail),
      setRuleEnabled: (rule, enabled) => {
        const key = inbox.rules().queryKey;
        queryClient.setQueryData<TriageRule[]>(key, (rules) =>
          rules?.map((row) => (row.id === rule.id ? { ...row, enabled } : row)),
        );
        void setTriageRuleEnabled(rule.id, enabled).then(
          (updated) => {
            queryClient.setQueryData<TriageRule[]>(key, (rules) =>
              rules?.map((row) => (row.id === updated.id ? updated : row)),
            );
            announce(enabled ? t("Back on") : t("Paused — matches wait for a person"));
          },
          (error: unknown) => {
            queryClient.setQueryData<TriageRule[]>(key, (rules) =>
              rules?.map((row) => (row.id === rule.id ? rule : row)),
            );
            fail(error);
          },
        );
      },
      deleteRule: (rule) => {
        const key = inbox.rules().queryKey;
        const previous = queryClient.getQueryData<TriageRule[]>(key);
        queryClient.setQueryData<TriageRule[]>(key, (rules) =>
          rules?.filter((row) => row.id !== rule.id),
        );
        const pending = deleteTriageRule(rule.id);
        pending.catch((error: unknown) => {
          queryClient.setQueryData(key, previous);
          fail(error);
        });
        announce(t("Rule deleted"), () => {
          void pending
            .then(() =>
              createTriageRule({
                classification: rule.classification,
                senderScope: rule.senderScope,
                sender: rule.sender,
                mailboxId: rule.mailboxId,
                action: rule.action,
                assigneeId: rule.assigneeId,
                enabled: rule.enabled,
              }),
            )
            .then(() => queryClient.invalidateQueries({ queryKey: key }), fail);
        });
      },
      createRule: (input) =>
        createTriageRule(input).then(
          (rule) => {
            queryClient.setQueryData<TriageRule[]>(inbox.rules().queryKey, (rules) =>
              rules === undefined ? rules : [rule, ...rules.filter((row) => row.id !== rule.id)],
            );
            void queryClient.invalidateQueries({ queryKey: inbox.rules().queryKey });
            void queryClient.invalidateQueries({
              queryKey: inbox.briefing().queryKey,
              refetchType: "active",
            });
            announce(t("Rule on — Nova starts with the next match"));
            return rule;
          },
          (error: unknown) => {
            throw error;
          },
        ),
      updateRule: (rule, input) => {
        const key = inbox.rules().queryKey;
        const optimistic: TriageRule = {
          ...rule,
          classification: input.classification,
          senderScope: input.senderScope,
          sender: input.sender ?? "",
          mailboxId: input.mailboxId ?? null,
          action: input.action,
          assigneeId: input.assigneeId ?? null,
          assignee: input.assigneeId === rule.assigneeId ? rule.assignee : null,
        };
        queryClient.setQueryData<TriageRule[]>(key, (rules) =>
          rules?.map((row) => (row.id === rule.id ? optimistic : row)),
        );
        void updateTriageRule(rule.id, rule.version, { ...input, enabled: rule.enabled }).then(
          (updated) => {
            queryClient.setQueryData<TriageRule[]>(key, (rules) =>
              rules?.map((row) => (row.id === updated.id ? updated : row)),
            );
            announce(t("Rule changed"));
          },
          (error: unknown) => {
            queryClient.setQueryData<TriageRule[]>(key, (rules) =>
              rules?.map((row) => (row.id === rule.id ? rule : row)),
            );
            void queryClient.invalidateQueries({ queryKey: key });
            fail(error);
          },
        );
      },
      handOff: (to) =>
        handOffTriage(to.id).then(
          (result) => {
            void queryClient.invalidateQueries({ queryKey: inbox._def });
            return result;
          },
          (error: unknown) => {
            throw error;
          },
        ),
      readUnread: () =>
        void readUnreadInboundMessages().then((count) => {
          void queryClient.invalidateQueries({ queryKey: inbox._def });
          announce(
            t(
              "{0, plural, one {Nova is reading # message} other {Nova is reading # messages}}",
              count,
            ),
          );
        }, fail),
      sendTestEmail: (mailbox) => {
        announce(t("Sending a test to {0}…", mailbox.address));
        void sendInboundMailboxTest(mailbox.id).then((updated) => {
          queryClient.setQueryData<InboundMailbox[]>(inbox.mailboxes().queryKey, (rows) =>
            rows?.map((row) => (row.id === updated.id ? updated : row)),
          );
        }, fail);
      },
      connectSource: async ({ name, address, watch, provider }) => {
        const watched = watch.trim();
        const credentials = await createInboundMailbox(
          {
            name: name.trim(),
            address: address.trim(),
            provider,
            purpose: watched === "" ? null : t("Forwarded from {0}", watched),
            reviewPolicy: "ReviewBelowConfidence",
            minConfidence: CONNECT_MIN_CONFIDENCE,
            status: "Active",
          },
          null,
          null,
        );
        let created = credentials.mailbox;
        let url = webhookUrl(API_BASE_URL, window.location.origin, credentials.webhookPath);
        if (provider === "Postmark") {
          const postmark = generatePostmarkCredentials();
          created = await setInboundMailboxSigningSecret(created.id, postmarkSecret(postmark));
          url = webhookUrl(
            API_BASE_URL,
            window.location.origin,
            credentials.webhookPath,
            postmark,
          );
        }
        queryClient.setQueryData<InboundMailbox[]>(inbox.mailboxes().queryKey, (rows) =>
          rows === undefined ? [created] : [...rows.filter((row) => row.id !== created.id), created],
        );
        void queryClient.invalidateQueries({ queryKey: inbox.mailboxes().queryKey });
        refreshSummaries(queryClient);
        useTriageStore.getState().markSourceConnected(Math.floor(Date.now() / 1000));
        return { mailbox: created, webhookUrl: url };
      },
      testSource: (mailbox) =>
        sendInboundMailboxTest(mailbox.id).then(
          (updated) => {
            queryClient.setQueryData<InboundMailbox[]>(inbox.mailboxes().queryKey, (rows) =>
              rows?.map((row) => (row.id === updated.id ? updated : row)),
            );
            return updated;
          },
          (error: unknown) => {
            throw error;
          },
        ),
      startDeskConversation: (agentId, message) =>
        apiService.assistantService
          .startThread(agentId, {
            origin: "Desk",
            title: message.subject.trim() === "" ? t("About an inbound message") : message.subject,
            subjectType: "InboundMessage",
            subjectId: message.id,
          })
          .then(
            (thread) => {
              useAssistantStore.getState().setLastAgentId(agentId);
              void queryClient.invalidateQueries({
                queryKey: queries.assistant.threads().queryKey,
              });
              return thread;
            },
            (error: unknown) => {
              throw error;
            },
          ),
      createView: (name, query) =>
        createMyTriageView(name, query).then(
          (view) => {
            queryClient.setQueryData<TriageView[]>(inbox.views().queryKey, (views) => [
              ...(views ?? []),
              view,
            ]);
            announce(t("Saved as a view"));
            return view;
          },
          (error: unknown) => {
            throw error;
          },
        ),
      renameView: (view, name) => {
        const trimmed = name.trim();
        if (trimmed === "" || trimmed === view.name) {
          return;
        }
        queryClient.setQueryData<TriageView[]>(inbox.views().queryKey, (views) =>
          views?.map((row) => (row.id === view.id ? { ...row, name: trimmed } : row)),
        );
        void renameMyTriageView(view.id, trimmed).then(
          (renamed) =>
            queryClient.setQueryData<TriageView[]>(inbox.views().queryKey, (views) =>
              views?.map((row) => (row.id === renamed.id ? renamed : row)),
            ),
          (error: unknown) => {
            queryClient.setQueryData<TriageView[]>(inbox.views().queryKey, (views) =>
              views?.map((row) => (row.id === view.id ? view : row)),
            );
            fail(error);
          },
        );
      },
      deleteView: (view) => {
        const previous = queryClient.getQueryData<TriageView[]>(inbox.views().queryKey);
        queryClient.setQueryData<TriageView[]>(inbox.views().queryKey, (views) =>
          views?.filter((row) => row.id !== view.id),
        );
        const pending = deleteMyTriageView(view.id);
        pending.catch((error: unknown) => {
          queryClient.setQueryData(inbox.views().queryKey, previous);
          fail(error);
        });
        announce(t("View deleted"), () => {
          void pending
            .then(() => createMyTriageView(view.name, view.query))
            .then(
              (restored) =>
                queryClient.setQueryData<TriageView[]>(inbox.views().queryKey, (views) => [
                  ...(views ?? []),
                  restored,
                ]),
              fail,
            );
        });
      },
      copyLink: (id) => copy(messageUrl(id), t("Link copied")),
      copyText: (text) => copy(text, t("Copied {0}", text)),
      openInNewTab: (id) => {
        window.open(messageUrl(id), "_blank", "noopener,noreferrer");
      },
      askInDesk: (message) =>
        useTriageStore.getState().openOverlay({
          kind: "desk",
          messageId: message.id,
          openActionIds: (message.actions ?? [])
            .filter((action) => !action.done)
            .map((action) => action.id),
        }),
    };
  }, [t, queryClient, navigation, timeZone, reviewKey, replyKey, loadKey, me]);
}
