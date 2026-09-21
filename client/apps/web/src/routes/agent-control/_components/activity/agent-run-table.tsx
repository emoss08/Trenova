import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { agentRunTableGraphQLConfig, type AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import { AGENT_EVALUATION_LIST_KEY, replayAgentRun } from "@/lib/graphql/agent-evaluations";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { RotateCcwIcon } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import { getRunColumns } from "./agent-run-columns";

export default function AgentRunTable() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getRunColumns(t), [t]);
  const { allowed: canReplay } = usePermission(Resource.AgentRun, Operation.Create);

  // A replay spends model calls and counts against the agent's budget, so it
  // is a deliberate action on a finished run rather than a button on every
  // row. The outcome lands in Evaluations.
  const replay = async (row: Row<AgentRunRow>) => {
    await replayAgentRun(row.original.id);
    toast.success(t("Replay started"), {
      description: t("The comparison appears under Evaluations when it finishes."),
    });
    await queryClient.invalidateQueries({ queryKey: [AGENT_EVALUATION_LIST_KEY] });
  };

  const contextMenuActions: RowAction<AgentRunRow>[] = [
    {
      id: "replay",
      label: t("Replay against the current agent"),
      icon: RotateCcwIcon,
      onClick: (row) => void replay(row),
      hidden: (row) =>
        !canReplay ||
        !row.original.agentDefinitionId ||
        row.original.status === "Pending" ||
        row.original.status === "GatheringContext" ||
        row.original.status === "Diagnosing",
    },
  ];

  return (
    <DataTable<AgentRunRow>
      name="Agent Run"
      queryKey="agent-run-list"
      graphql={agentRunTableGraphQLConfig}
      resource={Resource.AgentRun}
      columns={columns}
      contextMenuActions={contextMenuActions}
      enableCreateAction={false}
      refetchIntervalMs={30_000}
      initialColumnVisibility={{ modelIdentifier: false, completedAt: false }}
    />
  );
}
