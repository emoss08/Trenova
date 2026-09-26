import { ProposalEditor, type ProposalEditorRequest } from "@/components/assistant/proposal-editor";
import { batchPreviewDigests } from "@/components/assistant/proposal-preview/preview-gate";
import {
  prefetchPlanPreview,
  prefetchProposalPreview,
} from "@/components/assistant/proposal-preview/use-proposal-preview";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import { usePermission } from "@/hooks/use-permission";
import { decideAgentPlan, decideAgentProposal } from "@/lib/graphql/agent-decisions";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import {
  ReasonDialog,
  type ReasonDialogRequest,
} from "@/routes/agent-control/_components/activity/reason-dialog";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";
import { BatchBar } from "./batch-bar";
import { DecisionDetail } from "./decision-detail";
import { asAssistantProposal, presentDecision, type DecisionRowView } from "./decision-presenters";
import { DecisionRow } from "./decision-row";
import { DecisionsToolbar } from "./decisions-toolbar";
import { useDecisionKeys, type QueueAsk } from "./use-decision-keys";
import {
  decideAgentProposals,
  isPendingPlan,
  isPendingProposal,
  usePendingDecisions,
  type PendingDecisionNode,
} from "./use-pending-decisions";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * What is waiting on a person, as a queue with keys. The list on the left
 * is walked with j and k and marked with x; the row in focus is read whole
 * on the right, and a, r and m act on it. Shift+A approves everything
 * marked, one tool at a time, and every row of the batch reports its own
 * outcome. A plan is one row however many steps it has.
 */
export function DecisionQueue() {
  const t = useT();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const [now] = useState(nowInSeconds);

  const filter = useMemo(
    () => ({
      agentDefinitionId: searchParams.get("agent") ?? "",
      toolName: searchParams.get("tool") ?? "",
    }),
    [searchParams],
  );
  const queue = usePendingDecisions(filter);
  const nodes = useMemo(() => queue.data?.nodes ?? [], [queue.data?.nodes]);
  const rows = useMemo(
    () => nodes.map(presentDecision).filter((row): row is DecisionRowView => row !== null),
    [nodes],
  );
  const ids = useMemo(() => rows.map((row) => row.id), [rows]);
  const byId = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);

  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);
  const [editor, setEditor] = useState<ProposalEditorRequest | null>(null);
  const [busy, setBusy] = useState(false);
  const keysEnabled = canDecide && dialog === null && editor === null;

  // The keys hand their ask to whatever the surface can do at the time; the
  // ref keeps the handler stable while the decisions below are defined.
  const askRef = useRef<(ask: QueueAsk) => void>(() => {});
  const onAsk = useCallback((ask: QueueAsk) => askRef.current(ask), []);
  const { selection, focus, toggle, select } = useDecisionKeys(ids, {
    enabled: keysEnabled,
    onAsk,
  });

  const focused = selection.focusedId !== null ? (byId.get(selection.focusedId) ?? null) : null;

  // The digest of every proposal preview the detail pane has put on screen,
  // by proposal. A batch approval sends these and only these: a proposal the
  // person never opened goes without one and is recorded as unreviewed.
  const shownDigests = useRef(new Map<string, string>());
  const onPreviewShown = useCallback((proposalId: string, digest: string) => {
    shownDigests.current.set(proposalId, digest);
  }, []);

  // Walking the queue reads the row ahead before the key that reaches it, so
  // its preview is on screen the moment it is focused.
  const lastFocusIndex = useRef(-1);
  useEffect(() => {
    const index = selection.focusedId === null ? -1 : ids.indexOf(selection.focusedId);
    if (index === -1) {
      lastFocusIndex.current = -1;
      return;
    }
    const direction = index < lastFocusIndex.current ? -1 : 1;
    lastFocusIndex.current = index;
    const ahead = byId.get(ids[index + direction] ?? "");
    if (!ahead) {
      return;
    }
    if (isPendingPlan(ahead)) {
      void prefetchPlanPreview(queryClient, "approver", ahead.id);
    } else {
      void prefetchProposalPreview(queryClient, "approver", ahead.id);
    }
  }, [byId, ids, queryClient, selection.focusedId]);

  const selectedRows = rows.filter((row) => selection.selectedIds.includes(row.id));
  const selectedTools = new Set(selectedRows.map((row) => row.toolName));
  const mixedTools = selectedTools.size > 1;

  const afterDecision = useCallback(
    async (message: string) => {
      toast.success(message);
      await invalidateProposalViews(queryClient);
    },
    [queryClient],
  );

  const decideOne = useCallback(
    (node: PendingDecisionNode, decision: "Accepted" | "Rejected") => {
      const accepting = decision === "Accepted";
      const label = isPendingPlan(node)
        ? node.title
        : presentProposal(asAssistantProposal(node)).summary;
      setDialog({
        title: accepting ? t("Approve this change?") : t("Reject this change?"),
        description: accepting
          ? t("{0} Give a short reason so the audit trail explains the approval.", label)
          : t("{0} Say why so the next person and the agent's owner can learn from it.", label),
        confirmLabel: accepting ? t("Approve and run") : t("Reject"),
        reasonLabel: t("Reason"),
        requireReason: !accepting,
        destructive: !accepting,
        // The same preview the detail pane shows, read from the same cache
        // entry, so the digest the key sends is the one on screen.
        preview: {
          kind: isPendingPlan(node) ? "plan" : "proposal",
          scope: "approver",
          id: node.id,
          approving: accepting,
          density: "compact",
        },
        onConfirm: async (reason, previewDigest) => {
          const reasonCode = reason || (accepting ? "approved_from_desk" : "rejected_from_desk");
          if (isPendingPlan(node)) {
            await decideAgentPlan(node.id, { decision, reasonCode, previewDigest });
          } else {
            await decideAgentProposal(node.id, { decision, reasonCode, previewDigest });
          }
          await afterDecision(accepting ? t("Change approved") : t("Change rejected"));
        },
      });
    },
    [afterDecision, t],
  );

  const modifyOne = useCallback(
    (node: PendingDecisionNode) => {
      if (!isPendingProposal(node)) {
        return;
      }
      const proposal = asAssistantProposal(node);
      if (proposal.fields.length === 0) {
        toast.info(t("This change has nothing to edit; approve or reject it as proposed."));
        return;
      }
      setEditor({
        summary: presentProposal(proposal).summary,
        fields: proposal.fields,
        arguments: proposal.arguments,
        withReason: { label: t("Reason"), required: false },
        preview: { scope: "approver", proposalId: node.id },
        onConfirm: async (modifications, reason, previewDigest) => {
          await decideAgentProposal(node.id, {
            decision: "Modified",
            modifications,
            reasonCode: reason || "modified_from_desk",
            previewDigest,
          });
          await afterDecision(t("Change approved with your values"));
        },
      });
    },
    [afterDecision, t],
  );

  const decideMany = useCallback(
    (targetIds: string[], decision: "Accepted" | "Rejected") => {
      const accepting = decision === "Accepted";
      const targets = targetIds.filter((id) => {
        const node = byId.get(id);
        return node !== undefined && isPendingProposal(node);
      });
      if (targets.length === 0) {
        return;
      }
      if (targets.length === 1) {
        const only = byId.get(targets[0]);
        if (only) {
          decideOne(only, decision);
        }
        return;
      }
      // Read when the dialog opens, so the count it states is the count sent.
      const previewDigests = batchPreviewDigests(targets, shownDigests.current);
      const unreviewed = targets.length - previewDigests.length;
      const runsAsProposed = t(
        "Each will run exactly as proposed, one after another. A change that cannot run is reported on its own; the rest still go.",
      );
      setDialog({
        title: accepting
          ? t("Approve {0} changes?", targets.length)
          : t("Reject {0} changes?", targets.length),
        description: accepting
          ? unreviewed > 0
            ? `${runsAsProposed} ${t(
                "{0, plural, one {# of them you have not opened; it is recorded as approved without reviewing what it changes.} other {# of them you have not opened; they are recorded as approved without reviewing what they change.}}",
                unreviewed,
              )}`
            : runsAsProposed
          : t("None will run and each agent run is closed. Say why once for all of them."),
        confirmLabel: accepting ? t("Approve all") : t("Reject all"),
        reasonLabel: t("Reason"),
        requireReason: !accepting,
        destructive: !accepting,
        onConfirm: async (reason) => {
          setBusy(true);
          try {
            const results = await decideAgentProposals(targets, {
              decision,
              reasonCode:
                reason || (accepting ? "approved_from_desk_batch" : "rejected_from_desk_batch"),
              previewDigests,
            });
            const failed = results.filter((result) => result.error);
            const succeeded = results.length - failed.length;
            if (failed.length === 0) {
              toast.success(
                accepting
                  ? t("{0, plural, one {# change approved} other {# changes approved}}", succeeded)
                  : t("{0, plural, one {# change rejected} other {# changes rejected}}", succeeded),
              );
            } else {
              toast.warning(t("{0} of {1} went through", succeeded, results.length), {
                description: failed.map((result) => result.error).join(" · "),
              });
            }
            select([]);
            await invalidateProposalViews(queryClient);
          } finally {
            setBusy(false);
          }
        },
      });
    },
    [byId, decideOne, queryClient, select, t],
  );

  // A key asked for a decision; the surface carries it out.
  const carryOut = useCallback(
    (pending: QueueAsk) => {
      if (pending.ids.length > 1) {
        if (pending.kind === "modify") {
          toast.info(t("A change applies to one proposal at a time."));
          return;
        }
        decideMany(pending.ids, pending.kind === "accept" ? "Accepted" : "Rejected");
        return;
      }
      const node = byId.get(pending.ids[0]);
      if (!node) {
        return;
      }
      if (pending.kind === "modify") {
        modifyOne(node);
      } else {
        decideOne(node, pending.kind === "accept" ? "Accepted" : "Rejected");
      }
    },
    [byId, decideMany, decideOne, modifyOne, t],
  );
  useEffect(() => {
    askRef.current = carryOut;
  }, [carryOut]);

  // A notification lands on one proposal; the queue opens on it.
  const requested = searchParams.get("proposal");
  useEffect(() => {
    if (requested && byId.has(requested) && selection.focusedId !== requested) {
      focus(requested);
      setSearchParams(
        (current) => {
          const next = new URLSearchParams(current);
          next.delete("proposal");
          return next;
        },
        { replace: true },
      );
    }
  }, [byId, focus, requested, selection.focusedId, setSearchParams]);

  const actions = {
    busy: busy || !canDecide,
    onAccept: () => focused && decideOne(focused, "Accepted"),
    onReject: () => focused && decideOne(focused, "Rejected"),
    onModify: () => focused && modifyOne(focused),
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <DecisionsToolbar
        filter={filter}
        totalCount={queue.data?.totalCount ?? null}
        onFilterChange={(next) =>
          setSearchParams(
            (current) => {
              const params = new URLSearchParams(current);
              if (next.agentDefinitionId) params.set("agent", next.agentDefinitionId);
              else params.delete("agent");
              if (next.toolName) params.set("tool", next.toolName);
              else params.delete("tool");
              return params;
            },
            { replace: true },
          )
        }
      />
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize="44%" minSize="320px">
          <div className="relative flex h-full min-h-0 flex-col">
            {queue.isLoading ? (
              <div className="flex flex-col gap-2 p-3">
                <Skeleton className="h-14" />
                <Skeleton className="h-14" />
                <Skeleton className="h-14" />
              </div>
            ) : rows.length === 0 ? (
              <EmptySheet
                title={
                  filter.agentDefinitionId || filter.toolName
                    ? t("Nothing matches")
                    : t("Nothing is waiting on you")
                }
                description={
                  filter.agentDefinitionId || filter.toolName
                    ? t("Clear the filter to see the whole queue.")
                    : t("Changes an agent proposes will queue here for your approval.")
                }
                sketch={
                  <div className="flex flex-col gap-3">
                    <GhostLine className="w-3/4" />
                    <GhostLine className="w-1/2" />
                    <GhostLine className="w-2/3" />
                  </div>
                }
              />
            ) : (
              <ScrollArea className="min-h-0 flex-1">
                <div
                  role="listbox"
                  aria-label={t("Decisions")}
                  className="divide-border flex flex-col divide-y pb-16"
                >
                  {rows.map((row) => (
                    <DecisionRow
                      key={row.id}
                      row={row}
                      focused={row.id === selection.focusedId}
                      selected={selection.selectedIds.includes(row.id)}
                      now={now}
                      onFocus={() => focus(row.id)}
                      onToggle={() => toggle(row.id)}
                    />
                  ))}
                  {queue.hasNextPage && (
                    <div className="flex justify-center py-3">
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => void queue.fetchNextPage()}
                        isLoading={queue.isFetchingNextPage}
                      >
                        {t("Show more")}
                      </Button>
                    </div>
                  )}
                </div>
              </ScrollArea>
            )}
            {canDecide && (
              <BatchBar
                count={selectedRows.length}
                mixedTools={mixedTools}
                busy={busy}
                onAccept={() => decideMany(selection.selectedIds, "Accepted")}
                onReject={() => decideMany(selection.selectedIds, "Rejected")}
                onClear={() => select([])}
              />
            )}
          </div>
        </ResizablePanel>
        <ResizableHandle />
        <ResizablePanel minSize="40%">
          <div className="flex h-full min-h-0 flex-col">
            <DecisionDetail node={focused} actions={actions} onPreviewShown={onPreviewShown} />
          </div>
        </ResizablePanel>
      </ResizablePanelGroup>

      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
      <ProposalEditor request={editor} onClose={() => setEditor(null)} />
    </div>
  );
}
