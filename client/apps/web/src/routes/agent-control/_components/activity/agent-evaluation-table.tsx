import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import {
  AGENT_EVALUATION_LIST_KEY,
  agentEvaluationTableGraphQLConfig,
  type AgentEvaluationRow,
} from "@/lib/graphql/agent-evaluations";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { EyeIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { getEvaluationColumns } from "./agent-evaluation-columns";
import { EvaluationDetailDialog } from "./evaluation-detail-dialog";

/**
 * Recorded runs replayed against their agent as it is now. Each row is one
 * replay; opening it shows what the replay would have done beside what the
 * original did and how people decided on it.
 */
export default function AgentEvaluationTable() {
  const t = useT();
  const columns = useMemo(() => getEvaluationColumns(t), [t]);
  const [open, setOpen] = useState<string | null>(null);

  const contextMenuActions: RowAction<AgentEvaluationRow>[] = [
    {
      id: "open",
      label: t("Open comparison"),
      icon: EyeIcon,
      onClick: (row: Row<AgentEvaluationRow>) => setOpen(row.original.id),
    },
  ];

  return (
    <>
      <DataTable<AgentEvaluationRow>
        name="Agent Evaluation"
        queryKey={AGENT_EVALUATION_LIST_KEY}
        graphql={agentEvaluationTableGraphQLConfig}
        resource={Resource.AgentRun}
        columns={columns}
        contextMenuActions={contextMenuActions}
        enableCreateAction={false}
        refetchIntervalMs={15_000}
        initialColumnVisibility={{ model: false, sourceRunId: false }}
      />
      <EvaluationDetailDialog evaluationId={open} onClose={() => setOpen(null)} />
    </>
  );
}
