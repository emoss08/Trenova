import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import {
  AI_CORRECTION_DETAIL_KEY,
  fetchAICorrection,
  type AICorrectionRow,
} from "@/lib/graphql/extraction-eval";
import { useQuery } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { formatShare } from "../quality-model";
import { FieldResultsTable } from "./field-results-table";
import { documentKindLabel, modelLabel } from "./extraction-model";
import { usePromoteCorrection } from "./use-promote-correction";

/** One correction: every field as it was read beside what a person confirmed. */
export function CorrectionPanel({ open, onOpenChange, row }: DataTablePanelProps<AICorrectionRow>) {
  const t = useT();
  const id = row?.id;
  const { allowed: canCreate } = usePermission(Resource.AgentEvalSuite, Operation.Create);
  const promote = usePromoteCorrection();

  const detail = useQuery({
    queryKey: [AI_CORRECTION_DETAIL_KEY, id],
    queryFn: ({ signal }) => fetchAICorrection(id ?? "", { signal }),
    enabled: open && id !== undefined,
  });
  const correction = detail.data;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={correction ? documentKindLabel(correction.documentKind, t) : t("Correction")}
      size="xl"
      headerActions={
        correction && canCreate ? (
          <div className="flex items-center gap-2">
            <Button
              type="button"
              size="sm"
              variant="secondary"
              isLoading={promote.isPending && promote.variables?.activate === true}
              onClick={() => promote.mutate({ correctionId: correction.id, activate: true })}
            >
              {t("Add as active case")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              isLoading={promote.isPending && promote.variables?.activate === false}
              onClick={() => promote.mutate({ correctionId: correction.id, activate: false })}
            >
              {t("Add as candidate")}
            </Button>
          </div>
        ) : undefined
      }
    >
      {detail.isLoading || !correction ? (
        <ComponentLoader />
      ) : (
        <div className="flex flex-col gap-4">
          <DescriptionList columns={3}>
            <DescriptionItem label={t("Captured")}>
              {formatUnixDateTime(correction.capturedAt)}
            </DescriptionItem>
            <DescriptionItem label={t("Model")}>
              {modelLabel(correction.extractionModel, t)}
            </DescriptionItem>
            <DescriptionItem label={t("Issuer")}>
              {correction.documentFingerprint || "—"}
            </DescriptionItem>
            <DescriptionItem label={t("Accuracy")} numeric>
              {correction.scoredCount > 0 ? formatShare(correction.accuracy) : "—"}
            </DescriptionItem>
            <DescriptionItem label={t("Draft confidence")} numeric>
              {formatShare(correction.predictedConfidence)}
            </DescriptionItem>
            <DescriptionItem label={t("Fields scored")} numeric>
              {correction.scoredCount}
            </DescriptionItem>
          </DescriptionList>
          <SectionPanel title={t("Fields")} count={correction.fieldResults.length}>
            {correction.fieldResults.length > 0 ? (
              <FieldResultsTable results={correction.fieldResults} />
            ) : (
              <SectionPanelQuiet>{t("Nothing was read or confirmed.")}</SectionPanelQuiet>
            )}
          </SectionPanel>
        </div>
      )}
    </DataTablePanelContainer>
  );
}
