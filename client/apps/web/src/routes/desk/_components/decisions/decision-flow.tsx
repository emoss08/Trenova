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
import {
  batchPreviewDigests,
  unreviewedRequired,
} from "@/components/assistant/proposal-preview/preview-gate";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { usePermission } from "@/hooks/use-permission";
import { decideAgentPlan, decideAgentProposal } from "@/lib/graphql/agent-decisions";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { formatTimeAgo } from "@/lib/time-utils";
import {
  ReasonDialog,
  type ReasonDialogRequest,
} from "@/routes/agent-control/_components/activity/reason-dialog";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { AlertCircleIcon, CheckIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { QueueClear, QueueHeader, QueueLayout, QueueReading } from "../queue/queue";
import { QueueBar, QueueBarSpacer, QueueButton, QueueKey, QueueNav } from "../queue/queue-bar";
import {
  QueueGroupHeader,
  QueueList,
  QueueRow,
  QueueUnseenDot,
  type QueueSection,
} from "../queue/queue-list";
import { useQueueCursor, useQueueLeave } from "../queue/use-queue-cursor";
import { DecisionCard, type CardGate } from "./decision-card";
import { asAssistantProposal } from "./decision-presenters";
import {
  decideAgentProposals,
  isPendingPlan,
  isPendingProposal,
  startOfToday,
  usePendingDecisions,
  useRecentDecisions,
  type PendingDecisionNode,
} from "./use-pending-decisions";

const SEEN_KEY = "desk.decisions.seen";
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

function itemOf(node: PendingDecisionNode): Item {
  const held = holdSwitchOf(node.hold, {
    agentId: node.run?.definition?.id ?? node.run?.agentDefinitionId ?? null,
    agentName: node.run?.definition?.name ?? null,
  });
  if (isPendingPlan(node)) {
    return { node, id: node.id, tool: "plan", title: node.title, reversible: true, held };
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
 * What waits on a person, one decision at a time. The queue on the left is
 * grouped by the kind of write, with how much of today's work is decided;
 * the card in the middle is the one decision in front of them, and the bar
 * under it approves or declines it, or every one like it at once.
 *
 * Approving sends the digest of the preview on screen, so a record that moved
 * since is caught rather than overwritten. Declining asks why, because the
 * reason goes back to the agent and to whoever owns it.
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

  // Deciding every write like this one together: the ids, or null for one at a time.
  const like = useMemo(
    () =>
      current && current.held === null && isPendingProposal(current.node)
        ? items.filter(
            (item) =>
              item.tool === current.tool && item.held === null && isPendingProposal(item.node),
          )
        : [],
    [current, items],
  );
  const [batch, setBatch] = useState<string[] | null>(null);
  if (batch && current && !batch.includes(current.id)) {
    setBatch(null);
  }

  const [gate, setGate] = useState<CardGate | null>(null);
  const [shownDigests, setShownDigests] = useState<ReadonlyMap<string, string>>(new Map());
  const onPreviewShown = useCallback((id: string, digest: string) => {
    setShownDigests((previous) =>
      previous.get(id) === digest ? previous : new Map(previous).set(id, digest),
    );
  }, []);
  // A batch approves every change in it together; one the server takes only
  // with the digest of a preview the person was shown must have been opened
  // first, and these are the ones still to read.
  const batchUnreviewed = useMemo(
    () =>
      batch
        ? unreviewedRequired(
            items
              .filter((item) => batch.includes(item.id))
              .map((item) => ({
                id: item.id,
                reviewRequired: isPendingProposal(item.node) ? item.node.reviewRequired : true,
              })),
            shownDigests,
          )
        : [],
    [batch, items, shownDigests],
  );

  const { leaving, leave } = useQueueLeave();
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

  /** Takes decided cards off the queue, the way the design does, then reads it again. */
  const finish = useCallback(
    (ids: readonly string[], ok: boolean) => {
      if (ok) ids.forEach((id) => approved.current.add(id));
      const next = cursor.after(ids);
      leave(ok ? "ok" : "no", () => {
        setGone((previous) => new Set([...previous, ...ids]));
        setBatch(null);
        setGate(null);
        setCurrentId(next);
        void invalidateProposalViews(queryClient);
      });
    },
    [cursor, leave, queryClient, setCurrentId],
  );

  const approve = useCallback(async () => {
    if (!current || busy || !canDecide || current.held) return;
    const ids = batch ?? [current.id];
    if (ids.length === 1 && !gate?.approvable) return;
    if (ids.length > 1 && batchUnreviewed.length > 0) return;
    setBusy(true);
    try {
      if (ids.length > 1) {
        const results = await decideAgentProposals(ids, {
          decision: "Accepted",
          reasonCode: "approved_from_desk_batch",
          previewDigests: batchPreviewDigests(ids, shownDigests),
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
        if (done.length > 0) finish(done, true);
        return;
      }
      const input = {
        decision: "Accepted" as const,
        reasonCode: "approved_from_desk",
        previewDigest: gate?.digest,
      };
      if (isPendingPlan(current.node)) {
        await decideAgentPlan(current.id, input);
      } else {
        await decideAgentProposal(current.id, input);
      }
      finish([current.id], true);
    } catch (error) {
      if (gate?.handleError(error)) {
        toast.info(t("This changed since you opened it"), {
          description: t("Look at it again before you approve."),
        });
      } else {
        toast.error(t("The approval didn't go through"), { description: errorText(error) });
      }
    } finally {
      setBusy(false);
    }
  }, [batch, batchUnreviewed.length, busy, canDecide, current, finish, gate, shownDigests, t]);

  const decline = useCallback(
    (initialReason?: string) => {
      if (!current || busy || !canDecide || current.held) return;
      const ids = batch ?? [current.id];
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
          if (ids.length > 1) {
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
    [batch, busy, canDecide, current, finish, t],
  );

  const editable =
    current !== null &&
    current.held === null &&
    isPendingProposal(current.node) &&
    current.node.parameterFields.length > 0 &&
    batch === null;
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
          await decideAgentProposal(node.id, {
            decision: "Modified",
            modifications,
            reasonCode: reason || "modified_from_desk",
            previewDigest,
          });
          finish([node.id], true);
        },
      });
    },
    [current, editable, finish, t],
  );

  const toggleBatch = useCallback(() => {
    setBatch((value) => (value ? null : like.length > 1 ? like.map((item) => item.id) : null));
  }, [like]);

  useEffect(() => {
    if (dialog || editor) return;
    const onKey = (event: KeyboardEvent) => {
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
        toggleBatch();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [approve, decline, dialog, editor, go, modify, toggleBatch]);

  const decided = recent.data ?? [];
  const total = decided.length + items.length + (queue.hasNextPage ? 1 : 0);
  const unopened = batch ? batch.filter((id) => !seen.has(id)).length : 0;

  const approveLabel = batch
    ? t("Approve {0}", batch.length)
    : gate && gate.refused > 0 && gate.refused < gate.count
      ? t("Approve {0}, skip {1}", gate.count - gate.refused, gate.refused)
      : t("Approve");

  const sections: QueueSection[] = groups.map((group) => ({
    key: group.tool,
    header: <QueueGroupHeader label={<code>{group.tool}</code>} count={group.items.length} />,
    rows: group.items.map((item) => ({
      key: item.id,
      render: () => (
        <QueueRow
          selected={current?.id === item.id}
          marked={batch?.includes(item.id) ?? false}
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
      className: "dk-done",
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
        ) : current ? (
          <DecisionCard
            key={current.id}
            node={current.node}
            leaving={leaving}
            onGate={setGate}
            onPreviewShown={onPreviewShown}
            onDeclineWith={(reason) => decline(reason)}
            onModify={editable ? modify : undefined}
            footer={
              like.length > 1 && canDecide ? (
                <div className="dk-dc2-like">
                  {batch ? (
                    <>
                      <b>{t("Deciding {0} together", batch.length)}</b>
                      <span>
                        {t("All {0}", current.tool)}
                        {batchUnreviewed.length > 0
                          ? ` · ${t(
                              "{0, plural, one {# change must be opened before these can be approved} other {# changes must be opened before these can be approved}}",
                              batchUnreviewed.length,
                            )}`
                          : unopened > 0 && ` · ${t("{0} not opened yet", unopened)}`}
                      </span>
                      <Button
                        variant="bare"
                        size="bare"
                        className={cn(LINK_CLASS, "ml-auto")}
                        onClick={() => setBatch(null)}
                      >
                        {t("Just this one")}
                      </Button>
                    </>
                  ) : (
                    <>
                      <span>{t("{0} more {1} waiting", like.length - 1, current.tool)}</span>
                      <Button
                        variant="bare"
                        size="bare"
                        className={cn(LINK_CLASS, "ml-auto")}
                        onClick={() => setBatch(like.map((item) => item.id))}
                      >
                        {t("Decide all {0} together", like.length)}
                      </Button>
                      <QueueKey>X</QueueKey>
                    </>
                  )}
                </div>
              ) : undefined
            }
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
            {current.held ? (
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
                <QueueButton shortcut="D" disabled={busy} onClick={() => decline()}>
                  {batch ? t("Decline {0}", batch.length) : t("Decline")}
                </QueueButton>
                <QueueButton
                  ink
                  shortcut="A"
                  disabled={
                    busy || (batch === null ? !gate?.approvable : batchUnreviewed.length > 0)
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
