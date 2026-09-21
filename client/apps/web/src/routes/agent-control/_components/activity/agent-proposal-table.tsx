import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  agentProposalTableGraphQLConfig,
  type AgentProposalRow,
} from "@/lib/graphql/agent-activity-tables";
import { decideAgentProposal } from "@/lib/graphql/agent-decisions";
import { aiControlStatsQueryKey } from "../overview/use-ai-control-stats";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckIcon, XIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { getProposalColumns } from "./agent-proposal-columns";
import { ReasonDialog, type ReasonDialogRequest } from "./reason-dialog";

export default function AgentProposalTable() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getProposalColumns(t), [t]);
  // The server decides a proposal under update, not approve; asking for the
  // wrong operation hid the buttons from everyone who actually held the right.
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);

  const afterDecision = async (message: string) => {
    toast.success(message);
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["agent-proposal-list"] }),
      queryClient.invalidateQueries({ queryKey: ["agent-run-list"] }),
      queryClient.invalidateQueries({ queryKey: aiControlStatsQueryKey }),
    ]);
  };

  const decide = (row: Row<AgentProposalRow>, decision: "Accepted" | "Rejected") => {
    const proposal = row.original;
    const accepting = decision === "Accepted";

    setDialog({
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
      onConfirm: async (reason) => {
        await decideAgentProposal(proposal.id, {
          decision,
          reasonCode: reason || (accepting ? "approved_from_activity" : "rejected_from_activity"),
        });
        await afterDecision(accepting ? t("Change approved") : t("Change rejected"));
      },
    });
  };

  const contextMenuActions: RowAction<AgentProposalRow>[] = [
    {
      id: "approve",
      label: t("Approve"),
      icon: CheckIcon,
      onClick: (row) => decide(row, "Accepted"),
      hidden: (row) => !canDecide || row.original.status !== "Pending",
    },
    {
      id: "reject",
      label: t("Reject"),
      icon: XIcon,
      variant: "destructive",
      onClick: (row) => decide(row, "Rejected"),
      hidden: (row) => !canDecide || row.original.status !== "Pending",
    },
  ];

  return (
    <>
      <DataTable<AgentProposalRow>
        name="Agent Proposal"
        queryKey="agent-proposal-list"
        graphql={agentProposalTableGraphQLConfig}
        resource={Resource.AgentProposal}
        columns={columns}
        contextMenuActions={contextMenuActions}
        enableCreateAction={false}
        refetchIntervalMs={30_000}
        initialColumnVisibility={{ runId: false }}
      />
      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
    </>
  );
}
