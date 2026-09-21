import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { SafetyScorecard } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { safetyRatingMeta, summariseInspections } from "@trenova/shared/lib/safety";
import { cn } from "@trenova/shared/lib/utils";
import {
  DISCIPLINARY_LEVEL_LABELS,
  type DisciplinaryLevel,
  type SafetyRating,
} from "@trenova/shared/types/worker-safety";
import { AwardIcon, PlusIcon } from "lucide-react";

type SafetyScorecardCardProps = {
  scorecard: SafetyScorecard;
  canRecord: boolean;
  canRecognise: boolean;
  onRecordEvent: () => void;
  onRecognise: () => void;
};

/**
 * The scorecard as a row of KPI tiles rather than a badge strip: a score, the
 * points against the thresholds that move the rating, the clean-inspection
 * record, and how long it has been quiet. Below them, the raw counts as plain
 * facts. Colour lives in the rating badge and the goal bar and nowhere else.
 */
export function SafetyScorecardCard({
  scorecard,
  canRecord,
  canRecognise,
  onRecordEvent,
  onRecognise,
}: SafetyScorecardCardProps) {
  const t = useT();

  const meta = safetyRatingMeta(scorecard.rating as SafetyRating);
  const quiet = scorecard.daysSinceLastEvent;

  return (
    <div data-testid="safety-scorecard" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold">{t("Safety")}</h3>
          <Badge variant={meta.badgeVariant}>{t(meta.label)}</Badge>
          <InfoPopover title={t("Safety score")}>
            <p>
              {t(
                "The score starts at 100 and loses 5 for each active point, 10 for each preventable accident, 15 for each out-of-service order and 5 for each disciplinary action still active. Event counts cover the last twelve months.",
              )}
            </p>
            <p>
              {t(
                "Points stay active for two years from the date of the event, then roll off. The rating turns to Watch at 6 active points or a score under 75, and to At risk at 10 points or a score under 50.",
              )}
            </p>
          </InfoPopover>
        </div>
        <div className="flex items-center gap-2">
          {canRecognise ? (
            <Button size="sm" variant="outline" onClick={onRecognise}>
              <AwardIcon className="size-3.5" />
              {t("Add recognition")}
            </Button>
          ) : null}
          {canRecord ? (
            <Button size="sm" onClick={onRecordEvent}>
              <PlusIcon className="size-3.5" />
              {t("Record event")}
            </Button>
          ) : null}
        </div>
      </div>

      <KpiStrip minItemWidth="14rem">
        <KpiStripItem
          label={t("Score")}
          value={<MetricValue value={String(scorecard.score)} unit="of 100" />}
          sub={t("From the last two years of events")}
          hint={t("From the last two years of events")}
        />
        <KpiStripItem
          label={t("Active points")}
          value={
            <span className="flex flex-col gap-1.5 pb-0.5">
              <MetricValue value={String(scorecard.activePoints)} unit="pts" />
              <ThresholdBar
                value={scorecard.activePoints}
                watch={scorecard.pointsWatchThreshold}
                atRisk={scorecard.pointsAtRiskThreshold}
              />
            </span>
          }
          sub={`of ${scorecard.pointsAtRiskThreshold} before at-risk`}
          hint={`of ${scorecard.pointsAtRiskThreshold} before at-risk`}
        />
        <KpiStripItem
          label={t("Inspections")}
          value={
            <MetricValue
              value={
                scorecard.inspections > 0
                  ? `${scorecard.inspectionsPassed}/${scorecard.inspections}`
                  : "—"
              }
              unit={scorecard.inspections > 0 ? "clean" : undefined}
            />
          }
          sub={summariseInspections(scorecard)}
          hint={summariseInspections(scorecard)}
        />
        <KpiStripItem
          label={t("Quiet for")}
          value={quiet == null ? "—" : `${quiet} days`}
          sub={
            quiet == null
              ? "No accidents, incidents or citations on record"
              : "Since the last accident, incident or citation"
          }
          hint={
            quiet == null
              ? "No accidents, incidents or citations on record"
              : "Since the last accident, incident or citation"
          }
        />
      </KpiStrip>

      <DescriptionList className="flex flex-wrap gap-y-2 rounded-lg border px-4 py-3">
        <DescriptionItem label={t("Accidents")} numeric>
          {scorecard.accidents}
        </DescriptionItem>
        <DescriptionItem label={t("Preventable")} numeric>
          {scorecard.preventableAccidents}
        </DescriptionItem>
        <DescriptionItem label={t("Citations")} numeric>
          {scorecard.citations}
        </DescriptionItem>
        <DescriptionItem label={t("Out of service")} numeric>
          {scorecard.outOfServiceOrders}
        </DescriptionItem>
        <DescriptionItem label={t("Open")} numeric>
          {scorecard.openEvents}
        </DescriptionItem>
        <DescriptionItem label={t("Recognition")} numeric>
          {scorecard.recognitions}
        </DescriptionItem>
        {scorecard.highestDiscipline ? (
          <DescriptionItem label={t("Discipline")} numeric className="ml-auto">
            {t(
              "{0} active, highest {1}",
              scorecard.activeDiscipline,
              DISCIPLINARY_LEVEL_LABELS[
                scorecard.highestDiscipline as DisciplinaryLevel
              ]?.toLowerCase() ?? scorecard.highestDiscipline,
            )}
          </DescriptionItem>
        ) : null}
      </DescriptionList>
    </div>
  );
}

function MetricValue({ value, unit }: { value: string; unit?: string }) {
  return (
    <span className="flex min-w-0 items-baseline gap-1">
      <span className="truncate">{value}</span>
      {unit ? (
        <span className="text-muted-foreground shrink-0 text-xs font-normal">{unit}</span>
      ) : null}
    </span>
  );
}

/**
 * Points against the two thresholds that move the rating. The fill stays
 * neutral below the watch line and turns to the warning token above it; the
 * at-risk line is the end of the bar.
 */
function ThresholdBar({ value, watch, atRisk }: { value: number; watch: number; atRisk: number }) {
  const max = Math.max(atRisk, value, 1);
  const fill = Math.min(100, (value / max) * 100);
  const watchAt = Math.min(100, (watch / max) * 100);
  return (
    <span className="bg-muted relative block h-1.5 rounded-sm" aria-hidden>
      <span
        className={cn(
          "absolute inset-y-0 left-0 block rounded-sm",
          value >= atRisk ? "bg-destructive" : value >= watch ? "bg-warning" : "bg-primary/60",
        )}
        style={{ width: `${fill}%` }}
      />
      <span
        className="bg-foreground/50 absolute -top-0.5 -bottom-0.5 block w-0.5 rounded-full"
        style={{ left: `calc(${watchAt}% - 1px)` }}
        title={`Watch at ${watch}`}
      />
    </span>
  );
}
