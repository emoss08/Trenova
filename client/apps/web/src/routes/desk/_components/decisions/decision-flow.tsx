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
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { AlertCircleIcon, CheckIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { QueueClear, QueueHeader, QueueLayout, QueueReading } from "../queue/queue";
import { QueueBar, QueueBarSpacer, QueueButton, QueueNav } from "../queue/queue-bar";
import {
  QueueGroupHeader,
  QueueList,
  QueueRow,
  QueueUnseenDot,
  type QueueSection,
} from "../queue/queue-list";
import { useQueueCursor, useQueueLeave } from "../queue/use-queue-cursor";
import { MAX_BATCH } from "./batch-review-model";
import { PlanCard, type CardGate } from "./decision-card";
import { DecisionSheet, type SheetGate } from "./decision-sheet";
import { asAssistantProposal } from "./decision-presenters";
import {
  isPendingPlan,
  isPendingProposal,
  startOfToday,
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
  reversible: boolean;
  /** The switch keeping it from being decided; null when it can be decided now. */
  held: HoldSwitch | null;
};

/** The group a plan is filed under, beside the groups named after one kind of write. */
const PLAN_GROUP = "plan";

function itemOf(node: PendingDecisionNode): Item {
  const held = holdSwitchOf(node.hold, {
    agentId: node.run?.definition?.id ?? node.run?.agentDefinitionId ?? null,
    agentName: node.run?.definition?.name ?? null,
  });
  if (isPendingPlan(node)) {
    return { node, id: node.id, tool: PLAN_GROUP, title: node.title, reversible: true, held };
  }
  const view = presentProposal(asAssistantProposal(node));
  return {
    node,
    id: node.id,
    tool: node.toolName,
    title: view.title,
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
 * What waits on a person. The queue on the left is grouped by the kind of
 * write, with how much of today's work is decided. In the middle is the
 * sheet of the open change's kind: one change reads whole, several read as
 * one with what they share shown once, and the open one drawn whole. The bar
 * under it approves or declines the open change, or the ticked ones together.
 * A plan is decided as one and keeps a card of its own.
 *
 * Approving sends the digest of each preview the person was shown, so a
 * record that moved since is caught rather than overwritten. Declining asks
 * why, because the reason goes back to the agent and to whoever owns it.
 */
export function DecisionFlow() {
  const t = useT();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const { allowed: canResume } = usePermission(Resource.AgentControl, Operation.Update);
  const { allowed: canEditAgent } = usePermission(Resource.AgentDefinition, Operation.Update);
  const queue = usePendingDecisions({});
  const [since] = useState(startOfToday);
  const recent = useRecentDecisions(since);

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

  // Rows not opened yet carry a dot; opening one puts it away for good.
  const [seen, setSeen] = useState(readSeen);
  const currentKey = current?.id;
  if (currentKey && !seen.has(currentKey)) {
    setSeen(new Set(seen).add(currentKey));
  }
  useEffect(() => writeSeen(seen), [seen]);

  // The sheet in front of the person: every change of the open one's kind,
  // fifty at a time around it. A plan is decided as one and keeps a card.
  const sheet = useMemo(() => {
    if (!current || !isPendingProposal(current.node)) return null;
    const kind = items.filter((item) => item.tool === current.tool && isPendingProposal(item.node));
    const position = Math.max(
      0,
      kind.findIndex((item) => item.id === current.id),
    );
    const start = Math.floor(position / MAX_BATCH) * MAX_BATCH;
    const nodes = kind
      .slice(start, start + MAX_BATCH)
      .flatMap((item) => (isPendingProposal(item.node) ? [item.node] : []));
    return { key: `${current.tool}:${start}`, nodes, total: kind.length };
  }, [current, items]);

  // Ticked changes belong to one sheet; opening another kind lets them go.
  const [tickedState, setTickedState] = useState<{ key: string; ids: ReadonlySet<string> }>({
    key: "",
    ids: EMPTY,
  });
  const ticked = sheet !== null && tickedState.key === sheet.key ? tickedState.ids : EMPTY;
  const sheetKey = sheet?.key ?? "";
  const tick = useCallback(
    (ids: readonly string[], value: boolean) => {
      if (sheetKey === "") return;
      setTickedState((previous) => {
        const next = new Set(previous.key === sheetKey ? previous.ids : EMPTY);
        for (const id of ids) {
          if (value) {
            next.add(id);
          } else {
            next.delete(id);
          }
        }
        return { key: sheetKey, ids: next };
      });
    },
    [sheetKey],
  );

  const [gate, setGate] = useState<CardGate | null>(null);
  const [sheetGate, setSheetGate] = useState<SheetGate | null>(null);
  const onPlan = current !== null && isPendingPlan(current.node);
  // The open change is decided alone while nothing is ticked; the ticked
  // changes are decided together once there are any.
  const currentGate = onPlan ? gate : (sheetGate?.current ?? null);
  const tickedGate = ticked.size > 0 ? (sheetGate?.ticked ?? null) : null;

  const { leaving, leave } = useQueueLeave();
  const [leavingIds, setLeavingIds] = useState<ReadonlySet<string>>(EMPTY);
  const [busy, setBusy] = useState(false);
  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);
  const [editor, setEditor] = useState<ProposalEditorRequest | null>(null);

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
        setTickedState((previous) => ({
          key: previous.key,
          ids: new Set([...previous.ids].filter((id) => !ids.includes(id))),
        }));
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

  const approve = useCallback(async () => {
    if (!current || busy || !canDecide) return;
    if (tickedGate !== null ? !tickedGate.approvable : current.held || !currentGate?.approvable) {
      return;
    }
    setBusy(true);
    try {
      if (tickedGate !== null) {
        const results = await decideAgentProposals(tickedGate.approveIds, {
          decision: "Accepted",
          reasonCode: "approved_from_desk_batch",
          previewDigests: tickedGate.digests,
        });
        const failed = results.filter((result) => result.error);
        if (failed.length > 0) {
          toast.warning(
            t("{0} of {1} went through", results.length - failed.length, results.length),
            {
              description: failed.map((result) => result.error).join(" · "),
            },
          );
        }
        const done = results.filter((result) => !result.error).map((result) => result.proposalId);
        // A change recorded as approved but refused or failed is decided too;
        // it leaves the queue, and its failure was just said.
        const notRun = failed
          .filter((result) => result.decision)
          .map((result) => result.proposalId);
        if (done.length + notRun.length > 0) finish([...done, ...notRun], done.length > 0, done);
        return;
      }
      const input = {
        decision: "Accepted" as const,
        reasonCode: "approved_from_desk",
        previewDigest: currentGate?.digest,
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
      if (currentGate?.handleError(error)) {
        toast.info(t("This changed since you opened it"), {
          description: t("Look at it again before you approve."),
        });
      } else {
        toast.error(t("The approval didn't go through"), { description: errorText(error) });
      }
    } finally {
      setBusy(false);
    }
  }, [busy, canDecide, current, currentGate, finish, settle, t, tickedGate]);

  /**
   * Turns down the ticked changes, or the open one while none is ticked or
   * when `alone` says the reason is about the open one only.
   */
  const decline = useCallback(
    (initialReason?: string, alone = false) => {
      if (!current || busy || !canDecide) return;
      const together = tickedGate !== null && !alone;
      const ids = together ? tickedGate.declineIds : current.held ? [] : [current.id];
      if (ids.length === 0) return;
      const node = current.node;
      setDialog({
        title: ids.length > 1 ? t("Decline {0} changes?", ids.length) : t("Decline this change?"),
        description: t("Say why. The agent sees it, and so does whoever looks after the agent."),
        confirmLabel: ids.length > 1 ? t("Decline {0}", ids.length) : t("Decline"),
        reasonLabel: t("Reason"),
        requireReason: true,
        destructive: true,
        initialReason,
        onConfirm: async (reason) => {
          if (together) {
            const results = await decideAgentProposals(ids, {
              decision: "Rejected",
              reasonCode: reason,
            });
            finish(
              results.filter((result) => !result.error).map((result) => result.proposalId),
              false,
            );
            return;
          }
          const input = { decision: "Rejected" as const, reasonCode: reason };
          if (isPendingPlan(node)) {
            await decideAgentPlan(node.id, input);
          } else {
            await decideAgentProposal(node.id, input);
          }
          finish([node.id], false);
        },
      });
    },
    [busy, canDecide, current, finish, t, tickedGate],
  );

  const editable =
    current !== null &&
    current.held === null &&
    isPendingProposal(current.node) &&
    current.node.parameterFields.length > 0 &&
    ticked.size === 0;
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
    if (!current || current.held || !isPendingProposal(current.node) || sheet === null) return;
    if (sheet.nodes.length < 2) return;
    tick([current.id], !ticked.has(current.id));
  }, [current, sheet, tick, ticked]);

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
    if (dialog || editor) return;
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [dialog, editor, onKey]);

  const decided = recent.data ?? [];
  const total = decided.length + items.length + (queue.hasNextPage ? 1 : 0);
  const approveLabel = tickedGate
    ? t("Approve {0}", tickedGate.approveIds.length)
    : currentGate && currentGate.refused > 0 && currentGate.refused < currentGate.count
      ? t("Approve {0}, skip {1}", currentGate.count - currentGate.refused, currentGate.refused)
      : t("Approve");

  const sections: QueueSection[] = groups.map((group) => ({
    key: group.tool,
    header: (
      <QueueGroupHeader
        label={group.tool === PLAN_GROUP ? t("Plans") : humanizeToolName(group.tool)}
        count={group.items.length}
      />
    ),
    rows: group.items.map((item) => ({
      key: item.id,
      render: () => (
        <QueueRow
          selected={current?.id === item.id}
          marked={ticked.has(item.id)}
          onSelect={() => setCurrentId(item.id)}
          leading={<DeskAgentTile agent={item.node.run?.definition ?? null} size="xs" />}
          title={item.title}
          subtitle={
            <>
              {formatTimeAgo(item.node.createdAt * 1000)}
              {item.reversible ? "" : ` · ${t("Can't be undone")}`}
              {item.held && ` · ${t("Held, not yet decidable")}`}
            </>
          }
          trailing={!seen.has(item.id) && <QueueUnseenDot label={t("Not opened yet")} />}
        />
      ),
    })),
  }));
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

  return (
    <>
      <QueueLayout
        label={t("Queue")}
        aside={
          <>
            <QueueHeader
              title={t("Queue")}
              progressLabel={t("{0} of {1} decided", decided.length, total)}
              done={decided.length}
              total={total}
            />
            <QueueList
              label={t("Waiting decisions")}
              sections={sections}
              selectedKey={current?.id ?? null}
              empty={!queue.isLoading && <div className="dk-dc2-qe">{t("Queue clear")}</div>}
              footer={
                queue.hasNextPage && (
                  <Button
                    variant="bare"
                    size="bare"
                    className={cn(LINK_CLASS, "mx-2 mt-2.5")}
                    onClick={() => void queue.fetchNextPage()}
                    disabled={queue.isFetchingNextPage}
                  >
                    {t("Load more")}
                  </Button>
                )
              }
            />
          </>
        }
      >
        {queue.isLoading ? (
          <QueueReading label={t("Reading the queue…")} />
        ) : current && isPendingPlan(current.node) ? (
          <PlanCard
            key={current.id}
            node={current.node}
            leaving={leaving}
            onGate={setGate}
            onDeclineWith={(reason) => decline(reason, true)}
          />
        ) : current && sheet !== null ? (
          <DecisionSheet
            key={sheet.key}
            nodes={sheet.nodes}
            total={sheet.total}
            currentId={current.id}
            ticked={ticked}
            onTick={tick}
            onSelect={setCurrentId}
            leaving={leaving}
            leavingIds={leavingIds}
            onGate={setSheetGate}
            onDeclineWith={(reason) => decline(reason, true)}
            onModify={editable ? modify : undefined}
          />
        ) : (
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
            {current.held && tickedGate === null ? (
              <HeldBar
                held={current.held}
                canTurnOff={current.held.scope === "organization" ? canResume : canEditAgent}
              />
            ) : canDecide ? (
              <>
                {editable && (
                  <Button
                    variant="bare"
                    size="bare"
                    className={cn(LINK_CLASS, "@max-[900px]/mainc:hidden")}
                    onClick={() => modify()}
                  >
                    {t("Change values")}
                  </Button>
                )}
                <QueueButton
                  shortcut="D"
                  disabled={busy || (tickedGate !== null && tickedGate.declineIds.length === 0)}
                  onClick={() => decline()}
                >
                  {tickedGate ? t("Decline {0}", tickedGate.declineIds.length) : t("Decline")}
                </QueueButton>
                <QueueButton
                  ink
                  shortcut="A"
                  disabled={
                    busy ||
                    (tickedGate !== null ? !tickedGate.approvable : !currentGate?.approvable)
                  }
                  onClick={() => void approve()}
                >
                  {approveLabel}
                </QueueButton>
              </>
            ) : (
              <span className="dk-dc2-qe">{t("You can read these, but not decide them.")}</span>
            )}
          </QueueBar>
        )}
      </QueueLayout>

      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
      <ProposalEditor request={editor} onClose={() => setEditor(null)} />
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
