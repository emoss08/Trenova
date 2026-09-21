import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  AGENT_MEMORY_LIST_KEY,
  agentMemoryTableGraphQLConfig,
  setAgentMemoryStatus,
  type AgentMemoryRow,
} from "@/lib/graphql/agent-memories";
import { aiControlStatsQueryKey } from "../overview/use-ai-control-stats";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { ArchiveRestoreIcon, ArchiveIcon } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import { getMemoryColumns } from "./memory-columns";
import { MemoryPanel } from "./memory-panel";

/**
 * What the organization has told its agents. Every row here is read into
 * the prompt of every agent that asks for memory, so the list is also the
 * place to see what an agent recorded on its own and to retire what no
 * longer holds. Retiring keeps the row: what an agent was told last month
 * is still worth being able to read.
 */
export default function MemoryTab() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getMemoryColumns(t), [t]);
  const { allowed: canUpdate } = usePermission(Resource.AgentMemory, Operation.Update);

  const setStatus = async (row: Row<AgentMemoryRow>, status: AgentMemoryRow["status"]) => {
    await setAgentMemoryStatus(row.original.id, status);
    toast.success(status === "Retired" ? t("Memory retired") : t("Memory restored"));
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [AGENT_MEMORY_LIST_KEY] }),
      queryClient.invalidateQueries({ queryKey: aiControlStatsQueryKey }),
    ]);
  };

  const contextMenuActions: RowAction<AgentMemoryRow>[] = [
    {
      id: "retire",
      label: t("Retire"),
      icon: ArchiveIcon,
      variant: "destructive",
      onClick: (row) => void setStatus(row, "Retired"),
      hidden: (row) => !canUpdate || row.original.status !== "Active",
    },
    {
      id: "restore",
      label: t("Restore"),
      icon: ArchiveRestoreIcon,
      onClick: (row) => void setStatus(row, "Active"),
      hidden: (row) => !canUpdate || row.original.status !== "Retired",
    },
  ];

  return (
    <DataTable<AgentMemoryRow>
      name="Memory"
      queryKey={AGENT_MEMORY_LIST_KEY}
      graphql={agentMemoryTableGraphQLConfig}
      resource={Resource.AgentMemory}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={MemoryPanel}
      initialColumnVisibility={{ toolName: false, expiresAt: false, lastUsedAt: false }}
    />
  );
}
