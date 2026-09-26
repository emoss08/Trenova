import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  EXTRACTION_EVAL_CASE_LIST_KEY,
  extractionEvalCaseTableGraphQLConfig,
  type ExtractionEvalCaseRow,
} from "@/lib/graphql/extraction-eval";
import type { ExtractionEvalCaseStatus } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArchiveIcon, ArchiveRestoreIcon, CircleCheckIcon } from "lucide-react";
import { useMemo } from "react";
import { getCaseColumns } from "./case-columns";
import { CasePanel } from "./case-panel";
import { nextCaseStatuses } from "./extraction-model";
import { useCaseMutations } from "./use-case-mutations";

/**
 * The extraction evaluation set: documents frozen from corrections with the
 * values a person confirmed. Active cases run in every evaluation; a candidate
 * waits for someone to decide it is a fair test.
 */
export function CasesTable() {
  const t = useT();
  const columns = useMemo(() => getCaseColumns(t), [t]);
  const { allowed: canUpdate } = usePermission(Resource.AgentEvalSuite, Operation.Update);
  const { update } = useCaseMutations();

  const movable = (row: Row<ExtractionEvalCaseRow>, status: ExtractionEvalCaseStatus) =>
    canUpdate && nextCaseStatuses(row.original.status).includes(status);
  const move = (row: Row<ExtractionEvalCaseRow>, status: ExtractionEvalCaseStatus) =>
    update.mutate({ id: row.original.id, input: { version: row.original.version, status } });

  const contextMenuActions: RowAction<ExtractionEvalCaseRow>[] = [
    {
      id: "activate",
      label: t("Activate"),
      icon: CircleCheckIcon,
      onClick: (row) => move(row, "Active"),
      hidden: (row) => !movable(row, "Active") || row.original.status === "Retired",
    },
    {
      id: "restore",
      label: t("Restore"),
      icon: ArchiveRestoreIcon,
      onClick: (row) => move(row, "Active"),
      hidden: (row) => !movable(row, "Active") || row.original.status !== "Retired",
    },
    {
      id: "retire",
      label: t("Retire"),
      icon: ArchiveIcon,
      variant: "destructive",
      onClick: (row) => move(row, "Retired"),
      hidden: (row) => !movable(row, "Retired"),
    },
  ];

  return (
    <DataTable<ExtractionEvalCaseRow>
      name="Extraction Evaluation Case"
      queryKey={EXTRACTION_EVAL_CASE_LIST_KEY}
      graphql={extractionEvalCaseTableGraphQLConfig}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={CasePanel}
      enableCreateAction={false}
      initialColumnVisibility={{ pageCount: false }}
    />
  );
}
