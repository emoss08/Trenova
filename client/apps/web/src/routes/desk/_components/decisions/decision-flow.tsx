import {
  ProposalEditor,
  type EditorFocus,
  type ProposalEditorRequest,
} from "@/components/assistant/proposal-editor";
import { batchPreviewDigests } from "@/components/assistant/proposal-preview/preview-gate";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
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

/** How long a decided card takes to leave before the next one comes in. */
const LEAVE_MS = 320;
const SEEN_KEY = "desk.decisions.seen";

type Item = {
  node: PendingDecisionNode;
  id: string;
  tool: string;
  title: string;
  reversible: boolean;
};

function itemOf(node: PendingDecisionNode): Item {
  if (isPendingPlan(node)) {
    return { node, id: node.id, tool: "plan", title: node.title, reversible: true };
  }
  const view = presentProposal(asAssistantProposal(node));
  return { node, id: node.id, tool: node.toolName, title: view.title, reversible: view.reversible };
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

  const [currentId, setCurrentId] = useState<string | null>(null);
  const index = Math.max(
    0,
    items.findIndex((item) => item.id === currentId),
  );
  const current = items[index] ?? null;

  // A notification lands on one proposal; the queue opens on it, and the
  // address forgets it so going back doesn't land there again.
  const requested = searchParams.get("proposal");
  if (requested && requested !== currentId && items.some((item) => item.id === requested)) {
    setCurrentId(requested);
  }
  useEffect(() => {
    if (requested && currentId === requested) {
      setSearchParams(
        (params) => {
          const next = new URLSearchParams(params);
          next.delete("proposal");
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
      current && isPendingProposal(current.node)
        ? items.filter((item) => item.tool === current.tool && isPendingProposal(item.node))
        : [],
    [current, items],
  );
  const [batch, setBatch] = useState<string[] | null>(null);
  if (batch && current && !batch.includes(current.id)) {
    setBatch(null);
  }

  const [gate, setGate] = useState<CardGate | null>(null);
  const shownDigests = useRef(new Map<string, string>());
  const onPreviewShown = useCallback((id: string, digest: string) => {
    shownDigests.current.set(id, digest);
  }, []);

  const [leaving, setLeaving] = useState<"ok" | "no" | null>(null);
  const [busy, setBusy] = useState(false);
  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);
  const [editor, setEditor] = useState<ProposalEditorRequest | null>(null);

  const go = useCallback(
    (step: number) => {
      if (items.length === 0) return;
      setCurrentId(items[(index + step + items.length) % items.length].id);
    },
    [index, items],
  );

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
      setLeaving(ok ? "ok" : "no");
      window.setTimeout(() => {
        const rest = items.filter((item) => !ids.includes(item.id));
        setGone((previous) => new Set([...previous, ...ids]));
        setLeaving(null);
        setBatch(null);
        setGate(null);
        setCurrentId(rest[Math.min(index, rest.length - 1)]?.id ?? null);
        void invalidateProposalViews(queryClient);
      }, LEAVE_MS);
    },
    [index, items, queryClient],
  );

  const approve = useCallback(async () => {
    if (!current || busy || !canDecide) return;
    const ids = batch ?? [current.id];
    if (ids.length === 1 && !gate?.approvable) return;
    setBusy(true);
    try {
      if (ids.length > 1) {
        const results = await decideAgentProposals(ids, {
          decision: "Accepted",
          reasonCode: "approved_from_desk_batch",
          previewDigests: batchPreviewDigests(ids, shownDigests.current),
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
  }, [batch, busy, canDecide, current, finish, gate, t]);

  const decline = useCallback(
    (initialReason?: string) => {
      if (!current || busy || !canDecide) return;
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

  return (
    <div className="dk-dc2">
      <aside className="dk-dc2-q" aria-label={t("Queue")}>
        <div className="dk-dc2-qh">
          <b>{t("Queue")}</b>
          <span>{t("{0} of {1} decided", decided.length, total)}</span>
        </div>
        <div className="dk-dc2-prog">
          <i style={{ width: `${total ? (decided.length / total) * 100 : 0}%` }} />
        </div>
        <div className="dk-dc2-ql">
          {groups.map((group) => (
            <div key={group.tool} className="dk-dc2-g">
              <div className="dk-dc2-gh">
                <code>{group.tool}</code>
                <i>{group.items.length}</i>
              </div>
              {group.items.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={cn(
                    "dk-dc2-qi",
                    current?.id === item.id && "dk-on",
                    batch?.includes(item.id) && "dk-in",
                  )}
                  aria-current={current?.id === item.id ? "true" : undefined}
                  onClick={() => setCurrentId(item.id)}
                >
                  <DeskAgentTile agent={item.node.run?.definition ?? null} size="xs" />
                  <span>
                    <b>{item.title}</b>
                    <em>
                      {formatTimeAgo(item.node.createdAt * 1000)}
                      {item.reversible ? "" : ` · ${t("final")}`}
                    </em>
                  </span>
                  {!seen.has(item.id) && (
                    <span className="dk-dc2-new">
                      <span className="sr-only">{t("Not opened yet")}</span>
                    </span>
                  )}
                </button>
              ))}
            </div>
          ))}
          {queue.hasNextPage && (
            <button
              type="button"
              className="dk-ec-link dk-dc2-qmore"
              onClick={() => void queue.fetchNextPage()}
              disabled={queue.isFetchingNextPage}
            >
              {t("Load more")}
            </button>
          )}
          {!queue.isLoading && items.length === 0 && (
            <div className="dk-dc2-qe">{t("Queue clear")}</div>
          )}
          {decided.length > 0 && (
            <div className="dk-dc2-g dk-done">
              <div className="dk-dc2-gh">
                {t("Decided today")}
                <i>{decided.length}</i>
              </div>
              {decided.map((row) => {
                const ok = row.decision !== "Rejected";
                const failed = ok && row.proposal.status === "ExecutionFailed";
                const who = row.decidedByName || t("someone");
                return (
                  <div
                    key={row.id}
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
                      <DeskIcon
                        name={failed ? "alert" : ok ? "check" : "x"}
                        size={10}
                        stroke={2.6}
                      />
                    </span>
                    <em className="sr-only">
                      {failed
                        ? t("Approved, didn't go through")
                        : ok
                          ? t("Approved")
                          : t("Declined")}
                    </em>
                    {presentProposal(asAssistantProposal(row.proposal)).title}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </aside>

      <main className="dk-dc2-main">
        {queue.isLoading ? (
          <div className="dk-dc2-clear">
            <span className="dk-dc2-qe">{t("Reading the queue…")}</span>
          </div>
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
                        {unopened > 0 && ` · ${t("{0} not opened yet", unopened)}`}
                      </span>
                      <button type="button" className="dk-ec-link" onClick={() => setBatch(null)}>
                        {t("Just this one")}
                      </button>
                    </>
                  ) : (
                    <>
                      <span>{t("{0} more {1} waiting", like.length - 1, current.tool)}</span>
                      <button
                        type="button"
                        className="dk-ec-link"
                        onClick={() => setBatch(like.map((item) => item.id))}
                      >
                        {t("Decide all {0} together", like.length)}
                      </button>
                      <span className="dk-kbd" aria-hidden>
                        X
                      </span>
                    </>
                  )}
                </div>
              ) : undefined
            }
          />
        ) : (
          <div className="dk-dc2-clear">
            <span className="dk-dc2-cic">
              <DeskIcon name="check" size={22} stroke={2.2} />
            </span>
            <b>{t("Queue clear")}</b>
            <span>
              {decided.length > 0
                ? t(
                    "{0, plural, one {# thing was decided today.} other {# things were decided today.}} Agents will ask here before they change anything else.",
                    decided.length,
                  )
                : t("Agents will ask here before they change anything.")}
            </span>
          </div>
        )}
        {current && (
          <div className="dk-dc2-bar">
            <button
              type="button"
              className="dk-dc2-nav"
              onClick={() => go(-1)}
              title={t("Previous (K)")}
              aria-label={t("Previous")}
              aria-keyshortcuts="K"
            >
              <DeskIcon name="chevR" size={12} stroke={2.4} />
            </button>
            <button
              type="button"
              className="dk-dc2-nav dk-dn"
              onClick={() => go(1)}
              title={t("Next (J)")}
              aria-label={t("Next")}
              aria-keyshortcuts="J"
            >
              <DeskIcon name="chevR" size={12} stroke={2.4} />
            </button>
            <span className="dk-dc2-pos">
              {index + 1} / {items.length}
              {queue.hasNextPage ? "+" : ""}
            </span>
            <span style={{ flex: 1 }} />
            {canDecide ? (
              <>
                {editable && (
                  <button type="button" className="dk-ec-link" onClick={() => modify()}>
                    {t("Change values")}
                  </button>
                )}
                <button
                  type="button"
                  className="dk-dc2-btn"
                  disabled={busy}
                  onClick={() => decline()}
                  aria-keyshortcuts="D"
                >
                  {batch ? t("Decline {0}", batch.length) : t("Decline")}
                  <span className="dk-kbd" aria-hidden>
                    D
                  </span>
                </button>
                <button
                  type="button"
                  className="dk-dc2-btn dk-ink"
                  disabled={busy || (batch === null && !gate?.approvable)}
                  onClick={() => void approve()}
                  aria-keyshortcuts="A"
                >
                  {approveLabel}
                  <span className="dk-kbd" aria-hidden>
                    A
                  </span>
                </button>
              </>
            ) : (
              <span className="dk-dc2-qe">{t("You can read these, but not decide them.")}</span>
            )}
          </div>
        )}
      </main>

      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
      <ProposalEditor request={editor} onClose={() => setEditor(null)} />
    </div>
  );
}
