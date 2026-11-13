import {
  holdActionText,
  holdReasonText,
  holdSwitchOf,
  type HoldSwitch,
} from "@/components/assistant/approval-queue";
import {
  ProposalEditor,
  type EditorFocus,
  type ProposalEditorRequest,
} from "@/components/assistant/proposal-editor";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { usePermission } from "@/hooks/use-permission";
import {
  decideAgentPlan,
  decideAgentProposal,
  decideAgentProposals,
} from "@/lib/graphql/agent-decisions";
import { announceRefusal, approvalFailure } from "@/lib/approval-outcome";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { formatTimeAgo } from "@/lib/time-utils";
import {
  ReasonDialog,
  type ReasonDialogRequest,
} from "@/routes/agent-control/_components/activity/reason-dialog";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import { AlertCircleIcon, CheckIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { askAgentMessage } from "@/components/assistant/proposal-preview/preview-warnings";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { NEEDS_DETAIL_TRANSITION_CLASS, NEEDS_LIST_TRANSITION_CLASS } from "../desk-motion";
import { DeskListSkeleton } from "../list/desk-list-skeleton";
import { DeskViewDetail, DeskViewList } from "../list/desk-views";
import { LIST_SETTLED_CLASS, useListEntrance } from "../list/use-list-entrance";
import {
  NO_DECISION_FILTERS,
  PLAN_GROUP,
  changeLabel,
  decisionFilterCount,
  isDecisionNarrowed,
  pendingFilter,
  type DecisionFilters,
} from "./decision-filters";
import { QueueClear, QueueReading } from "../queue/queue";
import { QueueBar, QueueBarSpacer, QueueButton, QueueNav } from "../queue/queue-bar";
import { QueueGroupHeader, QueueList, type QueueSection } from "../queue/queue-list";
import { useQueueCursor, useQueueLeave } from "../queue/use-queue-cursor";
import { MAX_BATCH } from "./batch-review-model";
import { BulkReviewDialog, type BulkReviewRequest } from "./bulk-review-dialog";
import { PlanCard, type CardGate } from "./decision-card";
import { DecisionChange } from "./decision-change";
import { DecisionDock } from "./decision-dock";
import { bulkGateOf, currentGateOf, flagOf, isHeld, refusalsOf } from "./decision-gates";
import { asAssistantProposal } from "./decision-presenters";
import { DecisionGroupHeading, DecisionRow, rowSubject } from "./decision-row";
import { useDecisionPreviews } from "./use-decision-previews";
import {
  isPendingPlan,
  isPendingProposal,
  usePendingDecisions,
  useRecentDecisions,
  type PendingDecisionNode,
} from "./use-pending-decisions";

const SEEN_KEY = "desk.decisions.seen";
const EMPTY: ReadonlySet<string> = new Set();
const LINK_CLASS =
  "text-sm font-medium whitespace-nowrap text-dsk-fg underline decoration-dsk-fg/30 underline-offset-2 hover:decoration-current";

type Item = {
  node: PendingDecisionNode;
  id: string;
  tool: string;
  title: string;
  /** What the change does, in a sentence; a row says it until its preview is read. */
  summary: string;
  reversible: boolean;
  /** The switch keeping it from being decided; null when it can be decided now. */
  held: HoldSwitch | null;
};

function itemOf(node: PendingDecisionNode): Item {
  const held = holdSwitchOf(node.hold, {
    agentId: node.run?.definition?.id ?? node.run?.agentDefinitionId ?? null,
    agentName: node.run?.definition?.name ?? null,
  });
  if (isPendingPlan(node)) {
    return {
      node,
      id: node.id,
      tool: PLAN_GROUP,
      title: node.title,
      summary: node.summary,
      reversible: true,
      held,
    };
  }
  const view = presentProposal(asAssistantProposal(node));
  return {
    node,
    id: node.id,
    tool: node.toolName,
    title: view.title,
    summary: view.summary,
    reversible: view.reversible,
    held,
  };
}

function readSeen(): Set<string> {
  try {
    return new Set(JSON.parse(localStorage.getItem(SEEN_KEY) ?? "[]") as string[]);
  } catch {
    return new Set();
  }
}

function writeSeen(seen: Set<string>) {
  try {
    localStorage.setItem(SEEN_KEY, JSON.stringify([...seen].slice(-500)));
  } catch {
    // A browser that keeps nothing shows every row as new; nothing else depends on it.
  }
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * The Decisions lens of Needs you: what agents ask to change. Its list,
 * under the page's head, is grouped by the kind of write. Each row leads with
 * what is its own, and is where changes are ticked to decide several: a
 * kind's heading ticks the kind, and the dock at the foot of the list
 * declines the ticked changes or opens the one review an approval of several
 * takes. The page beside the list only ever holds one change, read whole,
 * with the bar under it deciding that one. A plan is decided as one and takes
 * no tick.
 *
 * It fills the two cells a lens owns in the page's frame (list/desk-views):
 * the list under the page's head, and the change beside it. The head, the
 * search and the filter are the page's, which hands down what the queue is
 * narrowed to.
 *
 * Approving sends the digest of each preview the person was shown, so a
 * record that moved since is caught rather than overwritten. Declining asks
 * why, because the reason goes back to the agent and to whoever owns it.
 */
export function DecisionFlow({
  shown = true,
  filter = NO_DECISION_FILTERS,
  onFilter,
  since,
  keysPaused = false,
}: {
  /** Whether this lens is the one in front; it is kept, hidden, while another is. */
  shown?: boolean;
  /** The words, the agent and the kind of change the queue is narrowed to. */
  filter?: DecisionFilters;
  onFilter: (filter: DecisionFilters) => void;
  /** The start of the reader's day, which "Decided today" lists from. */
  since: number;
  /** Set while the head's filter menu is open: the queue's keys stand down. */
  keysPaused?: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const { allowed: canResume } = usePermission(Resource.AgentControl, Operation.Update);
  const { allowed: canEditAgent } = usePermission(Resource.AgentDefinition, Operation.Update);
  // The page's head reads the same queue for how far along today is; this is
  // that read, not a second one.
  const queue = usePendingDecisions(pendingFilter(filter));
  const recent = useRecentDecisions(since);
  const narrowed = isDecisionNarrowed(filter);

  const [gone, setGone] = useState<ReadonlySet<string>>(new Set());
  const groups = useMemo(() => {
    const out: { tool: string; items: Item[] }[] = [];
    for (const node of queue.data?.nodes ?? []) {
      if (gone.has(node.id)) continue;
      const item = itemOf(node);
      let group = out.find((candidate) => candidate.tool === item.tool);
      if (!group) {
        group = { tool: item.tool, items: [] };
        out.push(group);
      }
      group.items.push(item);
    }
    return out;
  }, [gone, queue.data?.nodes]);
  const items = useMemo(() => groups.flatMap((group) => group.items), [groups]);

  const cursor = useQueueCursor(items);
  const { current, index, go, select: setCurrentId } = cursor;
  const currentId = current?.id ?? null;

  // A notification lands on one proposal, and the Watchtower on one agent's
  // run; the queue opens on it, and the address forgets it so going back
  // doesn't land there again.
  const requestedProposal = searchParams.get("proposal");
  const requestedRun = searchParams.get("run");
  const requested =
    requestedProposal ??
    (requestedRun ? (items.find((item) => item.node.runId === requestedRun)?.id ?? null) : null);
  if (requested && requested !== currentId && items.some((item) => item.id === requested)) {
    setCurrentId(requested);
  }
  useEffect(() => {
    if (requested && currentId === requested) {
      setSearchParams(
        (params) => {
          const next = new URLSearchParams(params);
          next.delete("proposal");
          next.delete("run");
          return next;
        },
        { replace: true },
      );
    }
  }, [currentId, requested, setSearchParams]);

  // On a narrow window the list and the change take turns; a link to one
  // change opens on the change.
  const [showList, setShowList] = useState(
    () => !searchParams.has("proposal") && !searchParams.has("run"),
  );
  const openChange = (id: string) => {
    setCurrentId(id);
    setShowList(false);
  };

  // Rows not opened yet carry a dot; opening one puts it away for good.
  const [seen, setSeen] = useState(readSeen);
  const currentKey = current?.id;
  if (currentKey && !seen.has(currentKey)) {
    setSeen(new Set(seen).add(currentKey));
  }
  useEffect(() => writeSeen(seen), [seen]);

  // What each waiting change would do. A plan reads its own, in its card.
  const proposals = useMemo(
    () => items.flatMap((item) => (isPendingProposal(item.node) ? [item.node] : [])),
    [items],
  );
  const previews = useDecisionPreviews(
    useMemo(() => proposals.map((node) => node.id), [proposals]),
  );

  // Ticked in the list to decide together: any change that is not a plan and
  // not held, across kinds, as many as one decision carries.
  const [tickedIds, setTickedIds] = useState<ReadonlySet<string>>(EMPTY);
  const tickable = useMemo(
    () => new Map(proposals.filter((node) => !isHeld(node)).map((node) => [node.id, node])),
    [proposals],
  );
  const tickedNodes = useMemo(
    () => [...tickedIds].flatMap((id) => tickable.get(id) ?? []),
    [tickable, tickedIds],
  );
  const ticked = useMemo(() => new Set(tickedNodes.map((node) => node.id)), [tickedNodes]);
  const tick = useCallback((ids: readonly string[], value: boolean) => {
    setTickedIds((previous) => {
      const next = new Set(previous);
      for (const id of ids) {
        if (value) {
          next.add(id);
        } else {
          next.delete(id);
        }
      }
      return new Set([...next].slice(0, MAX_BATCH));
    });
  }, []);
  const bulk = bulkGateOf(tickedNodes, previews);

  const [gate, setGate] = useState<CardGate | null>(null);
  const onPlan = current !== null && isPendingPlan(current.node);
  const currentProposal = current !== null && isPendingProposal(current.node) ? current.node : null;
  const currentGate = onPlan
    ? gate
    : currentProposal
      ? currentGateOf(currentProposal, previews, t)
      : null;
  // A change that would be refused as it stands goes back to its agent with the reasons.
  const refusals = currentProposal ? refusalsOf(previews.entry(currentProposal.id)) : [];

  const { leaving, leave } = useQueueLeave();
  const [leavingIds, setLeavingIds] = useState<ReadonlySet<string>>(EMPTY);
  const [busy, setBusy] = useState(false);
  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);
  const [editor, setEditor] = useState<ProposalEditorRequest | null>(null);
  const [review, setReview] = useState<BulkReviewRequest | null>(null);

  // An approved write runs after the decision is recorded, so whether it
  // went through is learned when the decisions are read again: a failure is
  // said once, with the write's own reason.
  const approved = useRef(new Set<string>());
  const decidedRows = recent.data;
  useEffect(() => {
    for (const row of decidedRows ?? []) {
      if (!approved.current.has(row.proposal.id)) continue;
      if (row.proposal.status !== "ExecutionFailed") continue;
      approved.current.delete(row.proposal.id);
      toast.error(
        t(
          "Approved, but {0} didn't go through",
          presentProposal(asAssistantProposal(row.proposal)).title.toLowerCase(),
        ),
        { description: row.proposal.executionError || undefined },
      );
    }
  }, [decidedRows, t]);

  /**
   * Takes decided changes off the queue, the way the design does, then reads
   * it again. The approvals in `watch` are still to run, so a failure is said
   * when the decisions are read again.
   */
  const finish = useCallback(
    (ids: readonly string[], ok: boolean, watch: readonly string[] = ok ? ids : []) => {
      watch.forEach((id) => approved.current.add(id));
      const next = cursor.after(ids);
      setLeavingIds(new Set(ids));
      leave(ok ? "ok" : "no", () => {
        setGone((previous) => new Set([...previous, ...ids]));
        setLeavingIds(EMPTY);
        setTickedIds((previous) => new Set([...previous].filter((id) => !ids.includes(id))));
        setGate(null);
        setCurrentId(next);
        void invalidateProposalViews(queryClient);
      });
    },
    [cursor, leave, queryClient, setCurrentId],
  );

  /**
   * Takes an approved change off the queue. One recorded but not carried
   * out, refused because its record moved on or failed as it ran, says so now
   * rather than when the decisions are read again.
   */
  const settle = useCallback(
    (item: Item, notRun: { refused: boolean; why: string } | null) => {
      if (notRun === null) {
        finish([item.id], true);
        return;
      }
      if (notRun.refused) {
        announceRefusal(t, notRun.why);
      } else {
        toast.error(t("Approved, but {0} didn't go through", item.title.toLowerCase()), {
          description: notRun.why || undefined,
        });
      }
      finish([item.id], false);
    },
    [finish, t],
  );

  const approve = async () => {
    if (!current || busy || !canDecide || current.held || !currentGate?.approvable) return;
    if (refusals.length > 0) return;
    setBusy(true);
    try {
      const input = {
        decision: "Accepted" as const,
        reasonCode: "approved_from_desk",
        previewDigest: currentGate.digest,
      };
      if (isPendingPlan(current.node)) {
        const { plan, refused } = await decideAgentPlan(current.id, input);
        settle(current, plan.status === "Failed" ? { refused, why: plan.failureError } : null);
      } else {
        const result = await decideAgentProposal(current.id, input);
        const failure = approvalFailure(result);
        settle(current, failure === null ? null : { refused: result.refused, why: failure });
      }
    } catch (error) {
      if (currentGate.handleError(error)) {
        toast.info(t("This changed since you opened it"), {
          description: t("Look at it again before you approve."),
        });
      } else {
        toast.error(t("The approval didn't go through"), { description: errorText(error) });
      }
    } finally {
      setBusy(false);
    }
  };

  /**
   * Opens the one review an approval of several takes: every ticked change
   * that can be approved, as it was read. Confirming sends the digest of each.
   */
  const reviewTicked = () => {
    if (busy || !canDecide || bulk.reading || bulk.approveIds.length === 0) return;
    const ids = bulk.approveIds;
    const digests = bulk.digests;
    setReview({
      nodes: ids.flatMap((id) => tickable.get(id) ?? []),
      entries: ids.map(
        (id) => previews.entry(id) ?? { proposalId: id, preview: null, error: null },
      ),
      onConfirm: async () => {
        try {
          const results = await decideAgentProposals(ids, {
            decision: "Accepted",
            reasonCode: "approved_from_desk_batch",
            previewDigests: digests,
          });
          const failed = results.filter((result) => result.error);
          if (failed.length > 0) {
            toast.warning(
              t("{0} of {1} went through", results.length - failed.length, results.length),
              { description: failed.map((result) => result.error).join(" · ") },
            );
          }
          const done = results.filter((result) => !result.error).map((result) => result.proposalId);
          // A change recorded as approved but refused or failed is decided too;
          // it leaves the queue, and its failure was just said.
          const notRun = failed
            .filter((result) => result.decision)
            .map((result) => result.proposalId);
          if (done.length + notRun.length > 0) {
            finish([...done, ...notRun], done.length > 0, done);
          }
        } catch (error) {
          toast.error(t("The approval didn't go through"), { description: errorText(error) });
          throw error;
        }
      },
    });
  };

  const askWhy = useCallback(
    (
      ids: readonly string[],
      initialReason: string | undefined,
      run: (reason: string) => Promise<void>,
    ) => {
      setDialog({
        title: ids.length > 1 ? t("Decline {0} changes?", ids.length) : t("Decline this change?"),
        description: t("Say why. The agent sees it, and so does whoever looks after the agent."),
        confirmLabel: ids.length > 1 ? t("Decline {0}", ids.length) : t("Decline"),
        reasonLabel: t("Reason"),
        requireReason: true,
        destructive: true,
        initialReason,
        onConfirm: run,
      });
    },
    [t],
  );

  /** Turns down the open change, with the reason already written when it is being sent back. */
  const decline = useCallback(
    (initialReason?: string) => {
      if (!current || busy || !canDecide || current.held) return;
      const node = current.node;
      askWhy([node.id], initialReason, async (reason) => {
        const input = { decision: "Rejected" as const, reasonCode: reason };
        if (isPendingPlan(node)) {
          await decideAgentPlan(node.id, input);
        } else {
          await decideAgentProposal(node.id, input);
        }
        finish([node.id], false);
      });
    },
    [askWhy, busy, canDecide, current, finish],
  );

  /** Turns down every ticked change with one reason. */
  const declineTicked = useCallback(() => {
    if (busy || !canDecide || bulk.declineIds.length === 0) return;
    const ids = bulk.declineIds;
    askWhy(ids, undefined, async (reason) => {
      const results = await decideAgentProposals(ids, { decision: "Rejected", reasonCode: reason });
      finish(
        results.filter((result) => !result.error).map((result) => result.proposalId),
        false,
      );
    });
  }, [askWhy, bulk.declineIds, busy, canDecide, finish]);

  /** Sends the open change back to its agent with why it would be refused. */
  const sendBack = () => {
    if (!currentProposal || refusals.length === 0) return;
    decline(askAgentMessage(currentProposal.toolName, refusals, t));
  };

  const editable =
    current !== null &&
    current.held === null &&
    isPendingProposal(current.node) &&
    current.node.parameterFields.length > 0;
  const modify = useCallback(
    (focus?: EditorFocus) => {
      if (!current || !isPendingProposal(current.node) || !editable) return;
      const node = current.node;
      const proposal = asAssistantProposal(node);
      setEditor({
        summary: presentProposal(proposal).summary,
        fields: proposal.fields,
        arguments: proposal.arguments,
        focus,
        withReason: { label: t("Reason"), required: false },
        preview: { scope: "approver", proposalId: node.id },
        onConfirm: async (modifications, reason, previewDigest) => {
          const result = await decideAgentProposal(node.id, {
            decision: "Modified",
            modifications,
            reasonCode: reason || "modified_from_desk",
            previewDigest,
          });
          const failure = approvalFailure(result);
          settle(current, failure === null ? null : { refused: result.refused, why: failure });
        },
      });
    },
    [current, editable, settle, t],
  );

  /** Ticks or unticks the open change, when it is one that can be decided with others. */
  const tickCurrent = useCallback(() => {
    if (!current || !tickable.has(current.id)) return;
    tick([current.id], !ticked.has(current.id));
  }, [current, tick, tickable, ticked]);

  // The listener stays attached while the queue is open and reads the latest
  // handlers, so a key pressed in the same frame the Approve button enables
  // acts on the state the button shows rather than the previous render's.
  const onKey = useLatestCallback((event: KeyboardEvent) => {
    if (event.defaultPrevented) return;
    const target = event.target as HTMLElement | null;
    if (target?.closest("input,textarea,select,[contenteditable=true],[role=dialog]")) return;
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    const key = event.key.toLowerCase();
    if (key === "j" || event.key === "ArrowDown") {
      event.preventDefault();
      go(1);
    } else if (key === "k" || event.key === "ArrowUp") {
      event.preventDefault();
      go(-1);
    } else if (key === "a") {
      event.preventDefault();
      void approve();
    } else if (key === "d" || key === "r") {
      event.preventDefault();
      decline();
    } else if (key === "m") {
      event.preventDefault();
      modify();
    } else if (key === "x") {
      event.preventDefault();
      tickCurrent();
    } else if (event.key === "Escape" && ticked.size > 0) {
      event.preventDefault();
      tick([...ticked], false);
    }
  });

  useEffect(() => {
    if (dialog || editor || review || keysPaused) return;
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [dialog, editor, keysPaused, onKey, review]);

  // What was decided today is not narrowed by the search or the filters, so
  // it is listed only under the whole queue.
  const decided = narrowed ? [] : (recent.data ?? []);
  const approveLabel =
    currentGate && currentGate.refused > 0 && currentGate.refused < currentGate.count
      ? t("Approve {0}, skip {1}", currentGate.count - currentGate.refused, currentGate.refused)
      : t("Approve");
  const selecting = ticked.size > 0;
  const tickedKinds = new Set(tickedNodes.map((node) => node.toolName));

  const sections: QueueSection[] = groups.map((group) => {
    const canTick = group.items.filter((item) => tickable.has(item.id));
    const clear = canTick.filter((item) => flagOf(previews.entry(item.id)) === null);
    const all = canTick.length === 0 ? null : canTick.every((item) => ticked.has(item.id));
    const ids = canTick.map((item) => item.id);

    return {
      key: group.tool,
      header: (
        <DecisionGroupHeading
          label={changeLabel(t, group.tool)}
          count={group.items.length}
          selecting={selecting}
          clear={clear.length < canTick.length ? clear.length : 0}
          all={all}
          onSelectClear={() => {
            tick(ids, false);
            tick(
              clear.map((item) => item.id),
              true,
            );
          }}
          onToggleAll={() => tick(ids, all !== true)}
        />
      ),
      rows: group.items.map((item) => ({
        key: item.id,
        render: () => {
          const entry = previews.entry(item.id);
          const agentName = item.node.run?.definition?.name ?? "";
          const subject = rowSubject(entry, { title: item.title, summary: agentName }, t);
          const detail = subject.detail || item.summary;

          return (
            <DecisionRow
              selected={current?.id === item.id}
              ticked={ticked.has(item.id)}
              tickable={tickable.has(item.id)}
              selecting={selecting}
              agent={item.node.run?.definition ?? null}
              title={subject.title}
              detail={
                <>
                  {detail !== "" && detail !== subject.title && `${detail} · `}
                  {formatTimeAgo(item.node.createdAt * 1000)}
                  {item.held && ` · ${t("Held, not yet decidable")}`}
                </>
              }
              flag={flagOf(entry)}
              unseen={!seen.has(item.id)}
              leaving={leavingIds.has(item.id) ? leaving : null}
              onOpen={() => openChange(item.id)}
              onTick={(value) => tick([item.id], value)}
            />
          );
        },
      })),
    };
  });
  if (decided.length > 0) {
    sections.push({
      key: "decided",
      header: <QueueGroupHeader label={t("Decided today")} count={decided.length} />,
      rows: decided.map((row) => {
        const ok = row.decision !== "Rejected";
        const failed = ok && row.proposal.status === "ExecutionFailed";
        const who = row.decidedByName || t("someone");
        const Mark = failed ? AlertCircleIcon : ok ? CheckIcon : XCloseIcon;
        return {
          key: `decided:${row.id}`,
          render: () => (
            <div
              className="dk-dc2-qd"
              title={
                failed
                  ? `${t("Approved by {0}, but it didn't go through", who)}${row.proposal.executionError ? `: ${row.proposal.executionError}` : ""}`
                  : ok
                    ? t("Approved by {0}", who)
                    : t("Declined by {0}", who)
              }
            >
              <span className={failed ? "dk-fail" : ok ? "dk-ok" : "dk-no"}>
                <Mark className="size-2.5" strokeWidth={2.6} />
              </span>
              <em className="sr-only">
                {failed ? t("Approved, didn't go through") : ok ? t("Approved") : t("Declined")}
              </em>
              {presentProposal(asAssistantProposal(row.proposal)).title}
            </div>
          ),
        };
      }),
    });
  }

  // Rows rise in when the lens is drawn; a row scrolled into view later mounts still.
  const settled = useListEntrance("Decisions", items.length > 0, shown);

  return (
    <>
      <DeskViewList className={NEEDS_LIST_TRANSITION_CLASS.Decisions}>
        {queue.isLoading ? (
          <DeskListSkeleton label={t("Reading the queue…")} />
        ) : (
          <QueueList
            label={t("Waiting decisions")}
            sections={sections}
            selectedKey={current?.id ?? null}
            className={cn(
              settled && LIST_SETTLED_CLASS,
              "transition-opacity",
              queue.isPlaceholderData && "opacity-60",
            )}
            empty={
              narrowed ? (
                <div className="dk-w-empty">
                  <div>
                    <b>
                      {filter.search !== ""
                        ? t("Nothing matches “{0}”", filter.search)
                        : t("Nothing waiting matches these filters")}
                    </b>
                    {filter.search !== ""
                      ? t("Search reads what each change says and the record it is for.")
                      : t("Take a filter off to see more of what waits.")}
                    <div className="mt-3 flex justify-center">
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => onFilter(NO_DECISION_FILTERS)}
                      >
                        {decisionFilterCount(filter) > 0 && filter.search !== ""
                          ? t("Clear the search and filters")
                          : filter.search !== ""
                            ? t("Clear the search")
                            : t("Clear filters")}
                      </Button>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="dk-dc2-qe">{t("Queue clear")}</div>
              )
            }
            footer={
              (queue.hasNextPage || selecting) && (
                // While changes are ticked the list keeps room under its last
                // row, so the dock never covers one.
                <div className={cn("flex", selecting && "pb-30")}>
                  {queue.hasNextPage && (
                    <Button
                      variant="bare"
                      size="bare"
                      className={cn(LINK_CLASS, "mx-2 mt-2.5")}
                      onClick={() => void queue.fetchNextPage()}
                      disabled={queue.isFetchingNextPage}
                    >
                      {t("Load more")}
                    </Button>
                  )}
                </div>
              )
            }
          />
        )}
        <DecisionDock
          summary={{
            count: bulk.count,
            what:
              tickedKinds.size === 1
                ? humanizeToolName([...tickedKinds][0] ?? "")
                : t(
                    "{0, plural, one {# kind of change} other {# kinds of change}}",
                    tickedKinds.size,
                  ),
            approvable: bulk.approveIds.length,
            flagged: bulk.flagged,
            blocked: bulk.blocked,
            reading: bulk.reading,
          }}
          canDecide={canDecide}
          busy={busy}
          onClear={() => tick([...ticked], false)}
          onDecline={declineTicked}
          onReview={reviewTicked}
        />
      </DeskViewList>
      {/* The Desk's layout is the page's one main landmark; this is a part of it. */}
      <DeskViewDetail
        reading={shown && !showList && current !== null}
        // A column, so the change scrolls in its card and the bar keeps to the foot. In a
        // narrow window the frame hides a detail nothing is open in; flex only while it shows.
        className={cn(
          "relative flex-col max-[900px]:data-reading:flex min-[901px]:flex",
          NEEDS_DETAIL_TRANSITION_CLASS.Decisions,
        )}
      >
        <button
          type="button"
          className="dk-w-det-back mx-4 mt-3 mb-0 self-start ui-focus-ring"
          onClick={() => setShowList(true)}
        >
          <DeskIcon name="chevL" size={12} />
          {t("All that needs you")}
        </button>
        {queue.isLoading ? (
          <QueueReading label={t("Reading the queue…")} />
        ) : current && isPendingPlan(current.node) ? (
          <PlanCard
            key={current.id}
            node={current.node}
            leaving={leaving}
            onGate={setGate}
            onDeclineWith={decline}
          />
        ) : currentProposal ? (
          <DecisionChange
            key={currentProposal.id}
            node={currentProposal}
            entry={previews.entry(currentProposal.id)}
            reading={previews.reading(currentProposal.id)}
            failed={previews.failed(currentProposal.id)}
            onRetry={() => previews.refetch(currentProposal.id)}
            leaving={leavingIds.has(currentProposal.id) ? leaving : null}
            onModify={editable ? modify : undefined}
            canChange={currentGate?.approvable ?? false}
          />
        ) : narrowed ? null : (
          <QueueClear
            title={t("Queue clear")}
            body={
              decided.length > 0
                ? t(
                    "{0, plural, one {# thing was decided today.} other {# things were decided today.}} Agents will ask here before they change anything else.",
                    decided.length,
                  )
                : t("Agents will ask here before they change anything.")
            }
          />
        )}
        {current && (
          <QueueBar>
            <QueueNav onPrevious={() => go(-1)} onNext={() => go(1)} />
            <span className="dk-dc2-pos">
              {index + 1} / {items.length}
              {queue.hasNextPage ? "+" : ""}
            </span>
            <QueueBarSpacer />
            {current.held ? (
              <HeldBar
                held={current.held}
                canTurnOff={current.held.scope === "organization" ? canResume : canEditAgent}
              />
            ) : canDecide ? (
              <>
                {editable && (
                  <QueueButton
                    quiet
                    shortcut="M"
                    className="@max-[900px]/mainc:hidden"
                    onClick={() => modify()}
                  >
                    {t("Change values")}
                  </QueueButton>
                )}
                <QueueButton shortcut="D" disabled={busy} onClick={() => decline()}>
                  {t("Decline")}
                </QueueButton>
                {refusals.length > 0 ? (
                  <QueueButton ink disabled={busy} onClick={sendBack}>
                    {t("Send back to the agent")}
                  </QueueButton>
                ) : (
                  <QueueButton
                    ink
                    shortcut="A"
                    disabled={busy || !currentGate?.approvable}
                    onClick={() => void approve()}
                  >
                    {approveLabel}
                  </QueueButton>
                )}
              </>
            ) : (
              <span className="dk-dc2-qe">{t("You can read these, but not decide them.")}</span>
            )}
          </QueueBar>
        )}
      </DeskViewDetail>

      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
      <ProposalEditor request={editor} onClose={() => setEditor(null)} />
      <BulkReviewDialog request={review} onClose={() => setReview(null)} />
    </>
  );
}

/**
 * In place of the decision, for a change a switch holds: which switch, and a
 * link to it for someone who may turn it off. The server refuses a decision
 * while the switch is on, so there is nothing here to approve or decline.
 */
function HeldBar({ held, canTurnOff }: { held: HoldSwitch; canTurnOff: boolean }) {
  const t = useT();

  return (
    <>
      <span className="dk-dc2-qe" role="status">
        {holdReasonText(held, t)}
        {!canTurnOff && ` · ${t("An administrator can turn this off.")}`}
      </span>
      {canTurnOff && (
        <Link to={held.to} className={LINK_CLASS}>
          {holdActionText(held, t)}
        </Link>
      )}
    </>
  );
}
