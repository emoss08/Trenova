import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  AGENT_QUALITY_LIST_KEY,
  agentQualityTableGraphQLConfig,
  type AgentQualityRow,
} from "@/lib/graphql/agent-quality";
import { useT } from "@trenova/shared/i18n/use-t";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ListChecksIcon, ThumbsDownIcon } from "lucide-react";
import { useMemo } from "react";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { AgentQualityPanel } from "./agent-quality-panel";
import { getAgentQualityColumns } from "./quality-columns";

/**
 * Every agent: how people rated it against the window before, its suite
 * scores over the window, and how its last suite run went. Opening a row
 * shows the agent's figures and leads to its runs and its worst answers.
 */
export default function AgentsTable() {
  const t = useT();
  const navigate = useAIControlNavigation();
  const { allowed: canReadRatings } = usePermission(Resource.AgentFeedback, Operation.Read);
  const columns = useMemo(() => getAgentQualityColumns(t, canReadRatings), [canReadRatings, t]);

  const contextMenuActions: RowAction<AgentQualityRow>[] = [
    {
      id: "runs",
      label: t("Its suite runs"),
      icon: ListChecksIcon,
      onClick: (row) =>
        navigate({ tab: "quality", view: "runs", qualityAgent: row.original.agentDefinitionId }),
    },
    {
      id: "ratings",
      label: t("Its worst-rated answers"),
      icon: ThumbsDownIcon,
      onClick: (row) =>
        navigate({
          tab: "quality",
          view: "ratings",
          qualityAgent: row.original.agentDefinitionId,
        }),
      hidden: () => !canReadRatings,
    },
  ];

  return (
    <DataTable<AgentQualityRow>
      name="Agent Score"
      queryKey={AGENT_QUALITY_LIST_KEY}
      graphql={agentQualityTableGraphQLConfig}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={AgentQualityPanel}
      enableCreateAction={false}
      enableReadOnlyPanel
      initialColumnVisibility={{ enabled: false, openRegression: false }}
    />
  );
}
