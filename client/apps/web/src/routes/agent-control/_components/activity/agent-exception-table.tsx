import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  agentExceptionTableGraphQLConfig,
  type AgentExceptionRow,
} from "@/lib/graphql/agent-activity-tables";
import { resolveAgentException } from "@/lib/graphql/agent-decisions";
import type { AgentResolutionState } from "@trenova/graphql/generated/graphql";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircleIcon, EyeIcon, XCircleIcon } from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { getExceptionColumns } from "./agent-exception-columns";
import { invalidateAIControlCounts } from "../overview/use-ai-control-stats";
import { ExceptionActionsContext, ExceptionPanel } from "./exception-sheet";
import { ReasonDialog, type ReasonDialogRequest } from "./reason-dialog";

export default function AgentExceptionTable() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getExceptionColumns(t), [t]);
  const { allowed: canResolve } = usePermission(Resource.AgentException, Operation.Update);
  const [dialog, setDialog] = useState<ReasonDialogRequest | null>(null);

  const transition = (exception: AgentExceptionRow, state: AgentResolutionState) => {
    const copy: Record<AgentResolutionState, { title: string; confirm: string; message: string }> =
      {
        Open: { title: t("Reopen this case?"), confirm: t("Reopen"), message: t("Case reopened") },
        InReview: {
          title: t("Take this case?"),
          confirm: t("Mark in review"),
          message: t("Case marked in review"),
        },
        Resolved: {
          title: t("Resolve this case?"),
          confirm: t("Resolve"),
          message: t("Case resolved"),
        },
        Dismissed: {
          title: t("Dismiss this case?"),
          confirm: t("Dismiss"),
          message: t("Case dismissed"),
        },
      };

    setDialog({
      title: copy[state].title,
      description: t(
        "The agent could not settle this on its own. Add a note about what you did so the outcome is on record.",
      ),
      confirmLabel: copy[state].confirm,
      reasonLabel: t("Notes"),
      requireReason: state === "Dismissed",
      destructive: state === "Dismissed",
      onConfirm: async (notes) => {
        await resolveAgentException(exception.id, {
          resolutionState: state,
          resolutionNotes: notes || undefined,
        });
        toast.success(copy[state].message);
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["agent-exception-list"] }),
          invalidateAIControlCounts(queryClient),
        ]);
      },
    });
  };

  const contextMenuActions: RowAction<AgentExceptionRow>[] = [
    {
      id: "review",
      label: t("Mark in review"),
      icon: EyeIcon,
      onClick: (row) => transition(row.original, "InReview"),
      hidden: (row) => !canResolve || row.original.resolutionState !== "Open",
    },
    {
      id: "resolve",
      label: t("Resolve"),
      icon: CheckCircleIcon,
      onClick: (row) => transition(row.original, "Resolved"),
      hidden: (row) =>
        !canResolve ||
        row.original.resolutionState === "Resolved" ||
        row.original.resolutionState === "Dismissed",
    },
    {
      id: "dismiss",
      label: t("Dismiss"),
      icon: XCircleIcon,
      variant: "destructive",
      onClick: (row) => transition(row.original, "Dismissed"),
      hidden: (row) =>
        !canResolve ||
        row.original.resolutionState === "Resolved" ||
        row.original.resolutionState === "Dismissed",
    },
  ];

  return (
    <ExceptionActionsContext.Provider value={{ canResolve, transition }}>
      <DataTable<AgentExceptionRow>
        name="Agent Exception"
        emptyTitle={t("No agent exceptions yet")}
        queryKey="agent-exception-list"
        graphql={agentExceptionTableGraphQLConfig}
        resource={Resource.AgentException}
        columns={columns}
        contextMenuActions={contextMenuActions}
        enableCreateAction={false}
        enableReadOnlyPanel
        TablePanel={ExceptionPanel}
        refetchIntervalMs={30_000}
      />
      <ReasonDialog request={dialog} onClose={() => setDialog(null)} />
    </ExceptionActionsContext.Provider>
  );
}
