import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { agentRunTableGraphQLConfig, type AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getRunColumns } from "./agent-run-columns";

export default function AgentRunTable() {
  const t = useT();
  const columns = useMemo(() => getRunColumns(t), [t]);

  return (
    <DataTable<AgentRunRow>
      name="Agent Run"
      queryKey="agent-run-list"
      graphql={agentRunTableGraphQLConfig}
      resource={Resource.AgentRun}
      columns={columns}
      enableCreateAction={false}
      refetchIntervalMs={30_000}
      initialColumnVisibility={{ modelIdentifier: false, completedAt: false }}
    />
  );
}
