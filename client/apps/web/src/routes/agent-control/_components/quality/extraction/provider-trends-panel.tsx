import { Sparkline } from "@/components/kpi/sparkline";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import {
  EXTRACTION_PROVIDER_TRENDS_KEY,
  fetchExtractionProviderTrends,
  type ExtractionProviderTrend,
} from "@/lib/graphql/extraction-eval";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { formatDelta, formatShare } from "../quality-model";
import { EXTRACTION_STALE_MS } from "./extraction-model";
import { TREND_STATE, trendChange, trendPoints, trendState } from "./provider-trend-model";

const TREND_WEEKS = 12;
const BASELINE_WEEKS = 4;

function ProviderName({ trend }: { trend: ExtractionProviderTrend }) {
  const t = useT();
  if (trend.providerRemoved) {
    return <span className="text-muted-foreground">{t("Removed provider")}</span>;
  }

  return <span>{t("{0} · {1}", trend.providerName, trend.model)}</span>;
}

function TrendRow({ trend }: { trend: ExtractionProviderTrend }) {
  const t = useT();
  const state = TREND_STATE[trendState(trend)];
  const change = formatDelta(trendChange(trend));
  const points = trendPoints(trend);

  return (
    <TableRow>
      <TableCell className="whitespace-nowrap">
        <ProviderName trend={trend} />
      </TableCell>
      <TableCell>
        {points.length > 1 ? (
          <Sparkline data={points} fill={false} />
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        <div className="flex flex-col items-end">
          <span>{trend.checked.scored > 0 ? formatShare(trend.checked.accuracy) : "—"}</span>
          <span className="text-muted-foreground text-xs">
            {t("{0, plural, one {# field} other {# fields}}", trend.checked.scored)}
          </span>
        </div>
      </TableCell>
      <TableCell className="text-right tabular-nums">
        <div className="flex flex-col items-end">
          <span>{trend.baseline.scored > 0 ? formatShare(trend.baseline.accuracy) : "—"}</span>
          <span className="text-muted-foreground text-xs">
            {t("{0, plural, one {# field} other {# fields}}", trend.baseline.scored)}
          </span>
        </div>
      </TableCell>
      <TableCell
        className={cn(
          "text-right tabular-nums",
          change.tone === "success" && "text-success",
          change.tone === "danger" && "text-danger",
          change.tone === "muted" && "text-muted-foreground",
        )}
      >
        {change.text}
      </TableCell>
      <TableCell>
        <Badge variant={phaseTone(state.phase)} title={t(state.description ?? "")}>
          {t(state.text)}
        </Badge>
      </TableCell>
    </TableRow>
  );
}

/**
 * Each extraction provider's accuracy week by week, read from the corrections
 * people made, and whether last week fell below that provider's own recent
 * weeks. A model that has served well for months can still drift as new
 * customers' documents arrive; this is where that shows.
 */
export function ProviderTrendsPanel() {
  const t = useT();
  const trends = useQuery({
    queryKey: [EXTRACTION_PROVIDER_TRENDS_KEY],
    queryFn: ({ signal }) => fetchExtractionProviderTrends({ signal }),
    staleTime: EXTRACTION_STALE_MS,
  });

  return (
    <SectionPanel
      title={t("Accuracy by provider over time")}
      hint={t("last {0} weeks", TREND_WEEKS)}
      help={
        trends.data
          ? t(
              "Weeks run Monday to Sunday in UTC. Last week is compared with the {0} weeks before it, once it has {1} confirmed fields and they have {2}; a drop of more than {3} points is drift, and the people who choose providers are told each Monday.",
              BASELINE_WEEKS,
              trends.data.minWeekFields,
              trends.data.minBaselineFields,
              trends.data.driftPoints,
            )
          : undefined
      }
    >
      {trends.isError ? (
        <SectionPanelQuiet>
          {t("Accuracy by provider could not be loaded. Try again shortly.")}
        </SectionPanelQuiet>
      ) : !trends.data ? (
        <Skeleton className="m-3 h-24" aria-busy />
      ) : trends.data.providers.length === 0 ? (
        <SectionPanelQuiet>
          {t(
            "No corrections in the last {0} weeks came from an AI provider. They are recorded when someone creates a shipment from a document's draft.",
            TREND_WEEKS,
          )}
        </SectionPanelQuiet>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Provider")}</TableHead>
              <TableHead>{t("Weekly accuracy")}</TableHead>
              <TableHead className="text-right">{t("Last week")}</TableHead>
              <TableHead className="text-right">{t("{0} weeks before", BASELINE_WEEKS)}</TableHead>
              <TableHead className="text-right">{t("Change")}</TableHead>
              <TableHead>{t("Status")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {trends.data.providers.map((trend) => (
              <TrendRow key={trend.providerId} trend={trend} />
            ))}
          </TableBody>
        </Table>
      )}
    </SectionPanel>
  );
}
