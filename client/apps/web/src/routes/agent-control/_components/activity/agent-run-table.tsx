import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { agentRunTableGraphQLConfig, type AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { RefreshCcw01Icon } from "@trenova/shared/components/icons";
import { useMemo } from "react";
import { getRunColumns } from "./agent-run-columns";
import { runIsWorking } from "./activity-model";
import { AgentRunPanel, useReplayRun } from "./agent-run-sheet";

export default function AgentRunTable() {
  const t = useT();
  const columns = useMemo(() => getRunColumns(t), [t]);
  const replay = useReplayRun();

  const contextMenuActions: RowAction<AgentRunRow>[] = [
    {
      id: "replay",
      label: t("Replay against the current agent"),
      icon: RefreshCcw01Icon,
      onClick: (row) => void replay.run(row.original.id),
      hidden: (row) =>
        !replay.allowed || !row.original.agentDefinitionId || runIsWorking(row.original.status),
    },
  ];

  return (
    <DataTable<AgentRunRow>
      name="Agent Run"
      emptyTitle={t("No agent runs yet")}
      queryKey="agent-run-list"
      graphql={agentRunTableGraphQLConfig}
      resource={Resource.AgentRun}
      columns={columns}
      contextMenuActions={contextMenuActions}
      enableCreateAction={false}
      enableReadOnlyPanel
      TablePanel={AgentRunPanel}
      refetchIntervalMs={30_000}
      initialColumnVisibility={{ modelIdentifier: false, completedAt: false, trigger: false }}
    />
  );
}
