import { DataTable } from "@/components/data-table/data-table";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  cancelExtractionEvalRun,
  EXTRACTION_ACCURACY_KEY,
  EXTRACTION_EVAL_RUN_LIST_KEY,
  extractionEvalRunTableGraphQLConfig,
  type ExtractionEvalRunRow,
} from "@/lib/graphql/extraction-eval";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleStopIcon } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import { isRunActive } from "./extraction-model";
import { getRunColumns } from "./run-columns";
import { RunPanel } from "./run-panel";

/** Every evaluation run, newest first: which model, how it scored, what it cost. */
export function RunsTable() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getRunColumns(t), [t]);
  const { allowed: canUpdate } = usePermission(Resource.AgentEvalSuite, Operation.Update);

  const cancel = useApiMutation({
    mutationFn: (id: string) => cancelExtractionEvalRun(id),
    onSuccess: async () => {
      toast.success(t("Evaluation canceled"));
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_RUN_LIST_KEY] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ACCURACY_KEY] }),
      ]);
    },
    resourceName: t("Evaluation run"),
  });

  const contextMenuActions: RowAction<ExtractionEvalRunRow>[] = [
    {
      id: "cancel",
      label: t("Cancel run"),
      icon: CircleStopIcon,
      variant: "destructive",
      onClick: (row) => cancel.mutate(row.original.id),
      hidden: (row) => !canUpdate || !isRunActive(row.original.status),
    },
  ];

  return (
    <DataTable<ExtractionEvalRunRow>
      name="Extraction Evaluation Run"
      queryKey={EXTRACTION_EVAL_RUN_LIST_KEY}
      graphql={extractionEvalRunTableGraphQLConfig}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={RunPanel}
      enableCreateAction={false}
      initialColumnVisibility={{ avgLatencyMs: false }}
    />
  );
}
