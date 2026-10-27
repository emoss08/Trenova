import { useT } from "@trenova/shared/i18n/use-t";
import { decidable } from "./activity-model";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  agentProposalTableGraphQLConfig,
  type AgentProposalRow,
} from "@/lib/graphql/agent-activity-tables";
import { decideAgentProposal } from "@/lib/graphql/agent-decisions";
import { invalidateAIControlCounts } from "../overview/use-ai-control-stats";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  announceRefusal,
  approvalFailure,
  type ProposalDecisionOutcome,
} from "@/lib/approval-outcome";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { useQueryClient } from "@tanstack/react-query";
import { CheckIcon, Edit02Icon, XCloseIcon } from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { ProposalEditor, type ProposalEditorRequest } from "@/components/assistant/proposal-editor";
import { presentProposal } from "@/components/assistant/proposal-presenters";
import type { AssistantProposal } from "@/types/assistant";
import { getProposalColumns } from "./agent-proposal-columns";
import { ProposalActionsContext, ProposalPanel, type ProposalActions } from "./proposal-sheet";
import { ReasonDialog, type ReasonDialogRequest } from "./reason-dialog";

export default function AgentProposalTable() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getProposalColumns(t), [t]);
  // The server decides a proposal under update, not approve; asking for the
  // wrong operation hid the buttons from everyone who actually held the right.
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);
  const [editor, setEditor] = useState<ProposalEditorRequest | null>(null);

  const afterDecision = async () => {
    await Promise.all([
      invalidateProposalViews(queryClient),
      invalidateAIControlCounts(queryClient),
    ]);
  };

  // The approval is recorded either way. One refused because its record
  // moved on ran nothing, which the dialog closing would not say; a write
  // that failed as it ran shows in the table once it is read again.
  const sayIfNotRun = (result: ProposalDecisionOutcome) => {
    if (result.refused) {
      announceRefusal(t, approvalFailure(result));
    }
  };

  const decide = (
    proposal: AgentProposalRow,
    decision: "Accepted" | "Rejected",
    initialReason?: string,
  ) => {
    const accepting = decision === "Accepted";

    setDialog({
      initialReason,
      // A write the preview says would be refused is turned down with its
      // reasons rather than approved: the approval closes and the rejection
      // opens with them written.
      onAskAgent: accepting ? (reason) => decide(proposal, "Rejected", reason) : undefined,
      title: accepting ? t("Approve this change?") : t("Reject this change?"),
      description: accepting
        ? t(
            "{0} will run with the parameters the agent proposed. Give a short reason so the audit trail explains the approval.",
            proposal.toolName,
          )
        : t(
            "{0} will not run and the agent's run is closed. Say why so the next person and the agent's owner can learn from it.",
            proposal.toolName,
          ),
      confirmLabel: accepting ? t("Approve and run") : t("Reject"),
      reasonLabel: t("Reason"),
      requireReason: !accepting,
      destructive: !accepting,
      preview: {
        kind: "proposal",
        scope: "approver",
        id: proposal.id,
        approving: accepting,
        reviewRequired: proposal.reviewRequired,
      },
      onConfirm: async (reason, previewDigest) => {
        const result = await decideAgentProposal(proposal.id, {
          decision,
          reasonCode: reason || (accepting ? "approved_from_activity" : "rejected_from_activity"),
          previewDigest,
        });
        sayIfNotRun(result);
        await afterDecision();
      },
    });
  };

  // The values open as a form built from the tool's schema. Only real
  // changes are sent; the server validates them against the tool before it
  // records the decision, and runs the tool with them.
  const modify = (proposal: AgentProposalRow) => {
    const args = (proposal.toolParams ?? {}) as Record<string, unknown>;
    setEditor({
      summary: presentProposal({
        toolName: proposal.toolName,
        arguments: args,
      } as AssistantProposal).summary,
      fields: proposal.parameterFields,
      arguments: args,
      withReason: { label: t("Reason"), required: false },
      preview: { scope: "approver", proposalId: proposal.id },
      onConfirm: async (modifications, reason, previewDigest) => {
        const result = await decideAgentProposal(proposal.id, {
          decision: "Modified",
          modifications,
          reasonCode: reason || "modified_from_activity",
          previewDigest,
        });
        sayIfNotRun(result);
        await afterDecision();
      },
    });
  };

  const contextMenuActions: RowAction<AgentProposalRow>[] = [
    {
      id: "approve",
      label: t("Approve"),
      icon: CheckIcon,
      onClick: (row) => decide(row.original, "Accepted"),
      hidden: (row) => !decidable(row.original, canDecide),
    },
    {
      id: "modify",
      label: t("Approve with changes"),
      icon: Edit02Icon,
      onClick: (row) => modify(row.original),
      hidden: (row) =>
        !decidable(row.original, canDecide) || row.original.parameterFields.length === 0,
    },
    {
      id: "reject",
      label: t("Reject"),
      icon: XCloseIcon,
      variant: "destructive",
      onClick: (row) => decide(row.original, "Rejected"),
      hidden: (row) => !decidable(row.original, canDecide),
    },
  ];

  const actions: ProposalActions = {
    canDecide,
    approve: (proposal) => decide(proposal, "Accepted"),
    modify,
    reject: (proposal) => decide(proposal, "Rejected"),
  };

  return (
    <ProposalActionsContext.Provider value={actions}>
      <DataTable<AgentProposalRow>
        name="Agent Proposal"
        emptyTitle={t("No agent proposals yet")}
        queryKey="agent-proposal-list"
        graphql={agentProposalTableGraphQLConfig}
        resource={Resource.AgentProposal}
        columns={columns}
        contextMenuActions={contextMenuActions}
        enableCreateAction={false}
        enableReadOnlyPanel
        TablePanel={ProposalPanel}
        refetchIntervalMs={30_000}
        initialColumnVisibility={{ runId: false }}
      />
      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
      <ProposalEditor request={editor} onClose={() => setEditor(null)} />
    </ProposalActionsContext.Provider>
  );
}
