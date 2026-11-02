import { DataTable } from "@/components/data-table/data-table";
import {
  AI_CORRECTION_LIST_KEY,
  aiCorrectionTableGraphQLConfig,
  type AICorrectionRow,
} from "@/lib/graphql/extraction-accuracy";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getCorrectionColumns } from "./correction-columns";
import { CorrectionPanel } from "./correction-panel";

/**
 * Every shipment created from a document's draft, with how many of the draft's
 * fields a person had to change. Accuracy and drift are measured from these.
 */
export function CorrectionsTable() {
  const t = useT();
  const columns = useMemo(() => getCorrectionColumns(t), [t]);

  return (
    <DataTable<AICorrectionRow>
      name="AI Correction"
      emptyTitle={t("No AI corrections yet")}
      queryKey={AI_CORRECTION_LIST_KEY}
      graphql={aiCorrectionTableGraphQLConfig}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      TablePanel={CorrectionPanel}
      enableCreateAction={false}
      initialColumnVisibility={{ documentFingerprint: false }}
    />
  );
}
