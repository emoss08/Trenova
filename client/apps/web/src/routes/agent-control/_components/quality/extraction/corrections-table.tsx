import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import {
  AI_CORRECTION_LIST_KEY,
  aiCorrectionTableGraphQLConfig,
  type AICorrectionRow,
} from "@/lib/graphql/extraction-eval";
import { useT } from "@trenova/shared/i18n/use-t";
import type { RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleCheckIcon, ListPlusIcon } from "lucide-react";
import { useMemo } from "react";
import { getCorrectionColumns } from "./correction-columns";
import { CorrectionPanel } from "./correction-panel";
import { usePromoteCorrection } from "./use-promote-correction";

/**
 * Every shipment created from a document's draft, with how many of the draft's
 * fields a person had to change. A correction worth keeping is frozen into the
 * evaluation set, so later models are measured against it.
 */
export function CorrectionsTable() {
  const t = useT();
  const columns = useMemo(() => getCorrectionColumns(t), [t]);
  const { allowed: canCreate } = usePermission(Resource.AgentEvalSuite, Operation.Create);
  const promote = usePromoteCorrection();

  const contextMenuActions: RowAction<AICorrectionRow>[] = [
    {
      id: "promote-active",
      label: t("Add as active case"),
      icon: CircleCheckIcon,
      onClick: (row) => promote.mutate({ correctionId: row.original.id, activate: true }),
      hidden: () => !canCreate,
    },
    {
      id: "promote-candidate",
      label: t("Add as candidate"),
      icon: ListPlusIcon,
      onClick: (row) => promote.mutate({ correctionId: row.original.id, activate: false }),
      hidden: () => !canCreate,
    },
  ];

  return (
    <DataTable<AICorrectionRow>
      name="AI Correction"
      queryKey={AI_CORRECTION_LIST_KEY}
      graphql={aiCorrectionTableGraphQLConfig}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={CorrectionPanel}
      enableCreateAction={false}
      initialColumnVisibility={{ documentFingerprint: false }}
    />
  );
}
