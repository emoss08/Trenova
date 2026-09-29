import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import {
  EXTRACTION_SHADOW_RESULT_DETAIL_KEY,
  fetchExtractionShadowResult,
  type ExtractionShadowResultRow,
} from "@/lib/graphql/extraction-shadow";
import { useQuery } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { AICorrectionFieldResult } from "@/lib/graphql/extraction-eval";
import { formatShare, formatUsd } from "../quality-model";
import { OutcomeBadge, ShadowStatusBadge, ShadowVerdictBadge } from "./extraction-badges";
import { accuracyTone, fieldLabel } from "./extraction-model";
import { outcomesDiffer, pairFieldResults, type ShadowFieldRow } from "./shadow-model";

function Side({ result }: { result?: AICorrectionFieldResult }) {
  const t = useT();
  if (!result) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <div className="flex min-w-0 items-center gap-2">
      <span className="max-w-56 truncate" title={result.predicted}>
        {result.predicted || <span className="text-muted-foreground">—</span>}
      </span>
      <OutcomeBadge value={result.outcome} t={t} />
    </div>
  );
}

function ComparisonTable({ rows }: { rows: ShadowFieldRow[] }) {
  const t = useT();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("Field")}</TableHead>
          <TableHead>{t("Confirmed")}</TableHead>
          <TableHead>{t("Candidate read")}</TableHead>
          <TableHead>{t("Production read")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow key={row.key} data-state={outcomesDiffer(row) ? "selected" : undefined}>
            <TableCell className="whitespace-nowrap">{fieldLabel(row.key, t)}</TableCell>
            <TableCell className="max-w-56 truncate" title={row.confirmed}>
              {row.confirmed || <span className="text-muted-foreground">—</span>}
            </TableCell>
            <TableCell>
              <Side result={row.candidate} />
            </TableCell>
            <TableCell>
              <Side result={row.production} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

/**
 * One shadowed extraction: what the candidate would have drafted, and, once a
 * person confirmed the document, each field beside production's reading of it.
 */
export function ShadowResultPanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<ExtractionShadowResultRow>) {
  const t = useT();
  const id = row?.id;

  const detail = useQuery({
    queryKey: [EXTRACTION_SHADOW_RESULT_DETAIL_KEY, id],
    queryFn: ({ signal }) => fetchExtractionShadowResult(id ?? "", { signal }),
    enabled: open && id !== undefined,
  });
  const result = detail.data;
  const rows = result ? pairFieldResults(result.fieldResults, result.baselineFieldResults) : [];
  const differing = rows.filter(outcomesDiffer).length;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={result ? t("{0} shadow", result.providerName) : t("Shadow extraction")}
      subtitle={
        result ? (
          result.verdict ? (
            <ShadowVerdictBadge value={result.verdict} t={t} />
          ) : (
            <ShadowStatusBadge value={result.status} t={t} />
          )
        ) : undefined
      }
      size="xl"
    >
      {detail.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertDescription>{t("The shadow extraction could not be loaded.")}</AlertDescription>
        </Alert>
      ) : detail.isPending ? (
        <ComponentLoader />
      ) : !result ? (
        <SectionPanelQuiet>{t("This shadow extraction no longer exists.")}</SectionPanelQuiet>
      ) : (
        <div className="flex flex-col gap-4">
          {result.statusReason ? (
            <Alert variant={result.status === "Failed" ? "destructive" : "info"} size="sm">
              <AlertDescription>{result.statusReason}</AlertDescription>
            </Alert>
          ) : null}
          {result.status === "Completed" && !result.accepted ? (
            <Alert variant="warning" size="sm">
              <AlertDescription>
                {t(
                  "The candidate's answer failed the checks production's must pass ({0}), so its draft is the rule-based reading.",
                  result.rejectionReason,
                )}
              </AlertDescription>
            </Alert>
          ) : null}
          <KpiStrip aria-label={t("Shadow extraction figures")}>
            <KpiStripItem
              label={t("Candidate accuracy")}
              value={result.scoredCount > 0 ? formatShare(result.accuracy) : "—"}
              sub={t(
                "{0, plural, one {# field scored} other {# fields scored}}",
                result.scoredCount,
              )}
              tone={accuracyTone(result.accuracy, result.scoredCount)}
            />
            <KpiStripItem
              label={t("Production accuracy")}
              value={result.baselineScoredCount > 0 ? formatShare(result.baselineAccuracy) : "—"}
              sub={result.productionModel || t("Rules only")}
              tone={accuracyTone(result.baselineAccuracy, result.baselineScoredCount)}
            />
            <KpiStripItem
              label={t("Model")}
              value={result.servedModel || "—"}
              sub={result.providerName}
            />
            <KpiStripItem
              label={t("Latency")}
              value={result.latencyMs > 0 ? t("{0} ms", result.latencyMs) : "—"}
              sub={formatUsd(result.costUsd)}
            />
          </KpiStrip>
          <SectionPanel
            title={t("Field by field")}
            hint={
              rows.length > 0
                ? t("{0, plural, one {# field differs} other {# fields differ}}", differing)
                : undefined
            }
          >
            {rows.length > 0 ? (
              <ComparisonTable rows={rows} />
            ) : (
              <SectionPanelQuiet>
                {result.status === "Completed"
                  ? t("Scores appear once someone creates a shipment from this document's draft.")
                  : t("Nothing was scored.")}
              </SectionPanelQuiet>
            )}
          </SectionPanel>
        </div>
      )}
    </DataTablePanelContainer>
  );
}
