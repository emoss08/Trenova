import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  agentPlanTableGraphQLConfig,
  type AgentPlanRow,
} from "@/lib/graphql/agent-activity-tables";
import { decideAgentPlan } from "@/lib/graphql/agent-decisions";
import { aiControlStatsQueryKey } from "../overview/use-ai-control-stats";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { invalidateProposalViews } from "@/lib/proposal-cache";
import { useQueryClient } from "@tanstack/react-query";
import { CheckIcon, XIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { getPlanColumns } from "./agent-plan-columns";
import { ReasonDialog, type ReasonDialogRequest } from "./reason-dialog";

/**
 * Plans are decided under the proposal right, because a plan is nothing but
 * several proposals answered at once. Approving one runs every step in the
 * order the agent asked and stops at the first that fails.
 */
export default function AgentPlanTable() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getPlanColumns(t), [t]);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Update);
  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);

  const afterDecision = async (message: string) => {
    toast.success(message);
    await Promise.all([
      invalidateProposalViews(queryClient),
      queryClient.invalidateQueries({ queryKey: aiControlStatsQueryKey }),
    ]);
  };

  const decide = (row: Row<AgentPlanRow>, decision: "Accepted" | "Rejected") => {
    const plan = row.original;
    const accepting = decision === "Accepted";

    setDialog({
      title: accepting
        ? t("Approve all {0} changes?", plan.stepCount)
        : t("Reject all {0} changes?", plan.stepCount),
      description: accepting
        ? t(
            "{0} will run step by step in the order the agent asked, stopping at the first that fails. Give a short reason so the audit trail explains the approval.",
            plan.title,
          )
        : t(
            "None of the steps in {0} will run and the agent's run is closed. Say why so the next person and the agent's owner can learn from it.",
            plan.title,
          ),
      confirmLabel: accepting ? t("Approve and run") : t("Reject"),
      reasonLabel: t("Reason"),
      requireReason: !accepting,
      destructive: !accepting,
      onConfirm: async (reason) => {
        await decideAgentPlan(plan.id, {
          decision,
          reasonCode: reason || (accepting ? "approved_from_activity" : "rejected_from_activity"),
        });
        await afterDecision(accepting ? t("Plan approved") : t("Plan rejected"));
      },
    });
  };

  const contextMenuActions: RowAction<AgentPlanRow>[] = [
    {
      id: "approve",
      label: t("Approve all"),
      icon: CheckIcon,
      onClick: (row) => decide(row, "Accepted"),
      hidden: (row) => !canDecide || row.original.status !== "Pending",
    },
    {
      id: "reject",
      label: t("Reject all"),
      icon: XIcon,
      variant: "destructive",
      onClick: (row) => decide(row, "Rejected"),
      hidden: (row) => !canDecide || row.original.status !== "Pending",
    },
  ];

  return (
    <>
      <DataTable<AgentPlanRow>
        name="Agent Plan"
        queryKey="agent-plan-list"
        graphql={agentPlanTableGraphQLConfig}
        resource={Resource.AgentProposal}
        columns={columns}
        contextMenuActions={contextMenuActions}
        enableCreateAction={false}
        refetchIntervalMs={30_000}
        initialColumnVisibility={{ runId: false, decidedAt: false }}
      />
      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
    </>
  );
}
