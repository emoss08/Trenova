import { DataTable } from "@/components/data-table/data-table";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import {
  EXTRACTION_SHADOW_REPORT_KEY,
  EXTRACTION_SHADOW_RESULT_LIST_KEY,
  EXTRACTION_SHADOW_SETTINGS_KEY,
  extractionShadowResultTableGraphQLConfig,
  fetchExtractionShadowReport,
  fetchExtractionShadowSettings,
  type ExtractionShadowResultRow,
  type ExtractionShadowSettings,
} from "@/lib/graphql/extraction-shadow";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AlertCircleIcon, Sliders01Icon } from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { EXTRACTION_STALE_MS } from "./extraction-model";
import { ShadowFieldsTable } from "./shadow-fields-table";
import { ShadowFigures } from "./shadow-figures";
import { getShadowResultColumns } from "./shadow-result-columns";
import { ShadowResultPanel } from "./shadow-result-panel";
import { SHADOW_WINDOW_DAYS } from "./shadow-model";
import { ShadowSettingsDialog } from "./shadow-settings-dialog";

function CandidateSummary({ settings }: { settings: ExtractionShadowSettings }) {
  const t = useT();
  const providers = useQuery({ ...queries.aiProvider.list() });
  const provider = providers.data?.find((item) => item.id === settings.providerId);

  return (
    <DescriptionList columns={4} className="p-3">
      <DescriptionItem label={t("Status")}>
        <Badge variant={settings.enabled ? "success" : "neutral"}>
          {settings.enabled ? t("On") : t("Off")}
        </Badge>
      </DescriptionItem>
      <DescriptionItem label={t("Candidate provider")}>
        {provider ? t("{0} · {1}", provider.name, provider.model) : <DescriptionEmpty />}
      </DescriptionItem>
      <DescriptionItem label={t("Share of extractions")} numeric>
        {t("{0}%", settings.samplePercent)}
      </DescriptionItem>
      <DescriptionItem label={t("Most per 24 hours")} numeric>
        {settings.dailyLimit}
      </DescriptionItem>
    </DescriptionList>
  );
}

/**
 * A candidate provider shadowing production: a share of real extractions is
 * run again on it, kept, and scored beside production against the shipment a
 * person confirms. Nothing it answers ever reaches a draft.
 */
export function ShadowView() {
  const t = useT();
  const [editing, setEditing] = useState(false);
  const { allowed: canUpdate } = usePermission(Resource.AgentEvalSuite, Operation.Update);
  const columns = useMemo(() => getShadowResultColumns(t), [t]);

  const settings = useQuery({
    queryKey: [EXTRACTION_SHADOW_SETTINGS_KEY],
    queryFn: ({ signal }) => fetchExtractionShadowSettings({ signal }),
    staleTime: EXTRACTION_STALE_MS,
  });
  const report = useQuery({
    queryKey: [EXTRACTION_SHADOW_REPORT_KEY, SHADOW_WINDOW_DAYS],
    queryFn: ({ signal }) => fetchExtractionShadowReport(SHADOW_WINDOW_DAYS, null, { signal }),
    staleTime: EXTRACTION_STALE_MS,
  });

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <SectionPanel
        title={t("Shadow traffic")}
        help={t(
          "Shadow calls spend from the evaluation budget and stop when it is reached. Accuracy counts only documents a person confirmed, and both sides are scored on the same documents.",
        )}
        action={
          canUpdate && settings.data ? (
            <Button type="button" size="sm" variant="outline" onClick={() => setEditing(true)}>
              <Sliders01Icon />
              {t("Edit settings")}
            </Button>
          ) : undefined
        }
      >
        {settings.isError ? (
          <SectionPanelQuiet>{t("The shadow settings could not be loaded.")}</SectionPanelQuiet>
        ) : settings.data ? (
          <CandidateSummary settings={settings.data} />
        ) : (
          <Skeleton className="m-3 h-10" aria-busy />
        )}
      </SectionPanel>

      {report.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertCircleIcon />
          <AlertDescription>
            {t("The shadow comparison could not be loaded. Try again shortly.")}
          </AlertDescription>
        </Alert>
      ) : report.data ? (
        report.data.providerId ? (
          <>
            <ShadowFigures report={report.data} />
            {report.data.truncated ? (
              <Alert variant="info" size="sm">
                <AlertDescription>
                  {t("Only the most recent scored shadows are counted in the comparison.")}
                </AlertDescription>
              </Alert>
            ) : null}
            <SectionPanel
              title={t("Accuracy by field")}
              hint={t("biggest shortfall first")}
              help={t(
                "Correct fields divided by fields a person confirmed, on the documents both sides were scored on.",
              )}
            >
              {report.data.fields.length > 0 ? (
                <ShadowFieldsTable fields={report.data.fields} />
              ) : (
                <SectionPanelQuiet>
                  {t(
                    "No shadowed document has been confirmed yet. Scores appear once someone creates a shipment from one.",
                  )}
                </SectionPanelQuiet>
              )}
            </SectionPanel>
          </>
        ) : (
          <SectionPanel title={t("Comparison")}>
            <SectionPanelQuiet>
              {t("Choose a candidate provider to start shadowing production extraction.")}
            </SectionPanelQuiet>
          </SectionPanel>
        )
      ) : (
        <Skeleton className="h-16" aria-busy />
      )}

      <DataTable<ExtractionShadowResultRow>
        name="Extraction Shadow Result"
        emptyTitle={t("No extraction shadow results yet")}
        queryKey={EXTRACTION_SHADOW_RESULT_LIST_KEY}
        graphql={extractionShadowResultTableGraphQLConfig}
        resource={Resource.AgentEvalSuite}
        columns={columns}
        TablePanel={ShadowResultPanel}
        enableCreateAction={false}
        initialColumnVisibility={{ latencyMs: false }}
      />

      {settings.data ? (
        <ShadowSettingsDialog open={editing} onOpenChange={setEditing} settings={settings.data} />
      ) : null}
    </div>
  );
}
