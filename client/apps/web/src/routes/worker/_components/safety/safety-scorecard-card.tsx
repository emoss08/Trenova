import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import type { SafetyScorecard } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { safetyRatingMeta, summariseInspections } from "@trenova/shared/lib/safety";
import { cn } from "@trenova/shared/lib/utils";
import {
  DISCIPLINARY_LEVEL_LABELS,
  type DisciplinaryLevel,
  type SafetyRating,
} from "@trenova/shared/types/worker-safety";
import {
  AwardIcon,
  CalendarCheckIcon,
  ClipboardCheckIcon,
  GaugeIcon,
  PlusIcon,
  TargetIcon,
  type LucideIcon,
} from "lucide-react";

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

      <div className="grid grid-cols-2 gap-3">
        <Metric
          label={t("Score")}
          icon={GaugeIcon}
          value={String(scorecard.score)}
          unit="of 100"
          sub={t("From the last two years of events")}
        />
        <Metric
          label={t("Active points")}
          icon={TargetIcon}
          value={String(scorecard.activePoints)}
          unit="pts"
          sub={`of ${scorecard.pointsAtRiskThreshold} before at-risk`}
        >
          <ThresholdBar
            value={scorecard.activePoints}
            watch={scorecard.pointsWatchThreshold}
            atRisk={scorecard.pointsAtRiskThreshold}
          />
        </Metric>
        <Metric
          label={t("Inspections")}
          icon={ClipboardCheckIcon}
          value={
            scorecard.inspections > 0
              ? `${scorecard.inspectionsPassed}/${scorecard.inspections}`
              : "—"
          }
          unit={scorecard.inspections > 0 ? "clean" : undefined}
          sub={summariseInspections(scorecard)}
        />
        <Metric
          label={t("Quiet for")}
          icon={CalendarCheckIcon}
          value={quiet == null ? "—" : `${quiet} days`}
          sub={
            quiet == null
              ? "No accidents, incidents or citations on record"
              : "Since the last accident, incident or citation"
          }
        />
      </div>

      <dl className="flex flex-wrap gap-x-6 gap-y-2 rounded-lg border px-4 py-3 text-xs">
        <Count label={t("Accidents")} value={scorecard.accidents} />
        <Count label={t("Preventable")} value={scorecard.preventableAccidents} />
        <Count label={t("Citations")} value={scorecard.citations} />
        <Count label={t("Out of service")} value={scorecard.outOfServiceOrders} />
        <Count label={t("Open")} value={scorecard.openEvents} />
        <Count label={t("Recognition")} value={scorecard.recognitions} />
        {scorecard.highestDiscipline ? (
          <div className="ml-auto flex flex-col">
            <dt className="text-2xs text-muted-foreground uppercase">{t("Discipline")}</dt>
            <dd className="font-medium tabular-nums">
              {t(
                "{0} active, highest {1}",
                scorecard.activeDiscipline,
                DISCIPLINARY_LEVEL_LABELS[
                  scorecard.highestDiscipline as DisciplinaryLevel
                ]?.toLowerCase() ?? scorecard.highestDiscipline,
              )}
            </dd>
          </div>
        ) : null}
      </dl>
    </div>
  );
}

/**
 * A metric tile sized for the panel: the dashboard KPI cards carry fixed
 * heights and a six-column rhythm that a 650px panel cannot honour.
 */
function Metric({
  label,
  icon: Icon,
  value,
  unit,
  sub,
  children,
}: {
  label: string;
  icon: LucideIcon;
  value: string;
  unit?: string;
  sub: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="border-border/80 flex min-w-0 flex-col gap-2 rounded-lg border p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground truncate text-[11px] font-semibold uppercase">
          {label}
        </span>
        <span className="bg-accent inline-flex size-6 shrink-0 items-center justify-center rounded-md">
          <Icon className="size-3.5" />
        </span>
      </div>
      <div className="flex min-w-0 items-baseline gap-1">
        <span className="truncate text-2xl leading-none font-semibold tracking-tight tabular-nums">
          {value}
        </span>
        {unit ? <span className="text-muted-foreground shrink-0 text-xs">{unit}</span> : null}
      </div>
      {children}
      <p className="text-muted-foreground truncate text-[11px]" title={sub}>
        {sub}
      </p>
    </div>
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
    <div className="bg-muted relative h-1.5 rounded-sm" aria-hidden>
      <div
        className={cn(
          "absolute inset-y-0 left-0 rounded-sm",
          value >= atRisk ? "bg-destructive" : value >= watch ? "bg-warning" : "bg-primary/60",
        )}
        style={{ width: `${fill}%` }}
      />
      <div
        className="bg-foreground/50 absolute -top-0.5 -bottom-0.5 w-0.5 rounded-[1px]"
        style={{ left: `calc(${watchAt}% - 1px)` }}
        title={`Watch at ${watch}`}
      />
    </div>
  );
}

function Count({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex flex-col">
      <dt className="text-2xs text-muted-foreground uppercase">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}
