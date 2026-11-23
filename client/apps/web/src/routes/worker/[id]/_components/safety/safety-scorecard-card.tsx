import { InfoTooltip } from "@/components/info-tooltip";
import type { PreloadIntent } from "@/components/work-pane";
import type { SafetyScorecard, WorkerSafetyEventRow } from "@/lib/graphql/worker-safety";
import { Award01Icon, PlusIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { safetyRatingMeta } from "@trenova/shared/lib/safety";
import { cn } from "@trenova/shared/lib/utils";
import type { SafetyRating } from "@trenova/shared/types/worker-safety";
import type { CSSProperties } from "react";
import { useDrawn } from "@/hooks/use-drawn";
import {
  EARLIER_SEGMENT,
  pointsGauge,
  safetyEventHeadline,
  type GaugeSegment,
  type PointsGauge,
} from "./safety-model";

type SafetyScorecardCardProps = {
  scorecard: SafetyScorecard;
  /** The events loaded so far: they name the gauge's segments; the figures are the scorecard's. */
  events: readonly WorkerSafetyEventRow[];
  canRecord: boolean;
  canRecognise: boolean;
  onRecordEvent: () => void;
  onRecognise: () => void;
  recordIntent: PreloadIntent;
  recogniseIntent: PreloadIntent;
};

const FIGURE_TONE: Record<PointsGauge["tone"], string> = {
  calm: "text-foreground",
  watch: "text-warning-foreground",
  risk: "text-danger-foreground",
};

const FILL_TONE: Record<PointsGauge["tone"], string> = {
  calm: "bg-ink",
  watch: "bg-warning",
  risk: "bg-danger",
};

/**
 * The scorecard as one ruled band in three parts: the score with its rating and the two ways to
 * add to the record, the points gauge against the thresholds that move the rating, and the
 * record's counts as ruled facts.
 */
export function SafetyScorecardCard({
  scorecard,
  events,
  canRecord,
  canRecognise,
  onRecordEvent,
  onRecognise,
  recordIntent,
  recogniseIntent,
}: SafetyScorecardCardProps) {
  const t = useT();
  const meta = safetyRatingMeta(scorecard.rating as SafetyRating);
  const quiet = scorecard.daysSinceLastEvent;
  const gauge = pointsGauge(scorecard, events);

  return (
    <section
      data-testid="safety-scorecard"
      aria-label={t("Safety scorecard")}
      className="@container overflow-hidden rounded-lg border bg-card"
    >
      <div className="grid grid-cols-1 @xl:grid-cols-[17rem_minmax(0,1fr)] @5xl:grid-cols-[17rem_minmax(0,1fr)_minmax(0,25rem)]">
        <div className="flex min-w-0 flex-col border-b border-border-subtle px-4 pt-3.5 pb-4 @xl:border-r @xl:border-b-0">
          <div className="flex items-center gap-1.5 text-xs font-medium text-foreground-subtle">
            <span>{t("Safety score")}</span>
            <InfoTooltip label={t("Safety score")}>
              {t(
                "Starts at 100 and loses 5 for each active point, 10 for each preventable accident, 15 for each out-of-service order and 5 for each disciplinary action still active. The rating turns to Watch at {0} active points or a score under 75, and to At risk at {1} points or a score under 50.",
                scorecard.pointsWatchThreshold,
                scorecard.pointsAtRiskThreshold,
              )}
            </InfoTooltip>
          </div>
          <div className="mt-2 flex items-baseline gap-2">
            <span className="text-3xl font-semibold tabular-nums">
              {formatNumber(scorecard.score)}
            </span>
            <span className="text-sm text-muted-foreground">{t("of 100")}</span>
            <Badge variant={meta.badgeVariant} dot className="ml-1 self-center">
              {t(meta.label)}
            </Badge>
          </div>
          <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
            {quiet == null
              ? t("No accidents, incidents or citations on record")
              : t("{0, plural, one {Quiet for # day} other {Quiet for # days}}", quiet)}
          </p>
          {canRecognise || canRecord ? (
            <div className="mt-auto flex flex-wrap gap-2 pt-4">
              {canRecognise ? (
                <Button size="sm" variant="secondary" {...recogniseIntent} onClick={onRecognise}>
                  <Award01Icon className="size-3.5" />
                  {t("Add recognition")}
                </Button>
              ) : null}
              {canRecord ? (
                <Button size="sm" {...recordIntent} onClick={onRecordEvent}>
                  <PlusIcon className="size-3.5" />
                  {t("Record event")}
                </Button>
              ) : null}
            </div>
          ) : null}
        </div>

        <PointsGaugeView gauge={gauge} events={events} />

        <DescriptionList
          layout="ruled"
          columns={3}
          className="h-full rounded-none border-0 border-t border-border-subtle @xl:col-span-2 @5xl:col-span-1 @5xl:border-t-0 @5xl:border-l [&_dl]:h-full [&_dl]:auto-rows-fr"
        >
          <DescriptionItem label={t("Inspections")} numeric>
            {scorecard.inspections > 0
              ? t("{0} of {1} clean", scorecard.inspectionsPassed, scorecard.inspections)
              : t("None")}
          </DescriptionItem>
          <DescriptionItem label={t("Accidents")} numeric>
            {formatNumber(scorecard.accidents)}
          </DescriptionItem>
          <DescriptionItem label={t("Preventable")} numeric>
            {formatNumber(scorecard.preventableAccidents)}
          </DescriptionItem>
          <DescriptionItem label={t("Citations")} numeric>
            {formatNumber(scorecard.citations)}
          </DescriptionItem>
          <DescriptionItem label={t("Out of service")} numeric>
            {formatNumber(scorecard.outOfServiceOrders)}
          </DescriptionItem>
          <DescriptionItem label={t("Open events")} numeric>
            {formatNumber(scorecard.openEvents)}
          </DescriptionItem>
        </DescriptionList>
      </div>
    </section>
  );
}

function at(value: number, max: number): string {
  return `${Math.min(100, Math.max(0, (value / max) * 100)).toFixed(3)}%`;
}

/**
 * The ruled track from zero, one tick a point, with the watch and at-risk lines marked under it.
 * It fills to the active points once, the first time it is shown, one segment per event still
 * carrying points.
 */
function PointsGaugeView({
  gauge,
  events,
}: {
  gauge: PointsGauge;
  events: readonly WorkerSafetyEventRow[];
}) {
  const t = useT();
  const drawn = useDrawn();
  const byId = new Map(events.map((event) => [event.id, event]));

  return (
    <div className="flex min-w-0 flex-col px-5 pt-3.5 pb-4 @5xl:border-r-0">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-xs font-medium text-foreground-subtle">{t("Active points")}</span>
        <span className={cn("text-3xl font-semibold tabular-nums", FIGURE_TONE[gauge.tone])}>
          {formatNumber(gauge.points)}
        </span>
      </div>
      <div
        role="img"
        aria-label={t(
          "{0} active points on a scale of 0 to {1}. Watch from {2}, at risk from {3}.",
          gauge.points,
          gauge.max,
          gauge.watch,
          gauge.atRisk,
        )}
        className="relative mt-3.5 pt-1"
      >
        <div className="relative h-2.5 overflow-hidden rounded-xs bg-surface-active">
          <span
            data-testid="points-fill"
            className="absolute inset-y-0 left-0 flex origin-left transition-transform duration-420 ease-settle motion-reduce:transition-none"
            style={{ width: at(gauge.points, gauge.max), transform: `scaleX(${drawn ? 1 : 0})` }}
          >
            {gauge.segments.map((segment, index) => (
              <Tooltip key={segment.id}>
                <TooltipTrigger
                  render={
                    <span
                      className={cn(
                        "block h-full shrink-0",
                        FILL_TONE[gauge.tone],
                        index > 0 && "border-l border-card",
                      )}
                      style={{ flexBasis: `${(segment.share * 100).toFixed(3)}%` }}
                    />
                  }
                />
                <TooltipContent>
                  <SegmentNote segment={segment} event={byId.get(segment.id)} />
                </TooltipContent>
              </Tooltip>
            ))}
          </span>
          <span
            aria-hidden
            className="pointer-events-none absolute inset-0"
            style={
              {
                "--tick": `calc(100% / ${gauge.max})`,
                backgroundImage:
                  "repeating-linear-gradient(to right, transparent 0, transparent calc(var(--tick) - 1px), var(--color-card) calc(var(--tick) - 1px), var(--color-card) var(--tick))",
              } as CSSProperties
            }
          />
        </div>
        <span
          aria-hidden
          className="pointer-events-none absolute top-0 -ml-px h-4.5 w-px bg-warning-foreground"
          style={{ left: at(gauge.watch, gauge.max) }}
        />
        <span
          aria-hidden
          className="pointer-events-none absolute top-0 -ml-px h-4.5 w-px bg-danger-foreground"
          style={{ left: at(gauge.atRisk, gauge.max) }}
        />
        <div
          aria-hidden
          className="relative mt-1.5 h-4 text-2xs text-foreground-subtle tabular-nums"
        >
          <span className="absolute left-0">0</span>
          <span
            className="absolute -translate-x-1/2 whitespace-nowrap text-foreground"
            style={{ left: at(gauge.watch, gauge.max) }}
          >
            <span className="font-medium text-warning-foreground">{t("Watch")}</span>{" "}
            {formatNumber(gauge.watch)}
          </span>
          <span
            className="absolute -translate-x-1/2 whitespace-nowrap text-foreground"
            style={{ left: at(gauge.atRisk, gauge.max) }}
          >
            <span className="font-medium text-danger-foreground">{t("At risk")}</span>{" "}
            {formatNumber(gauge.atRisk)}
          </span>
          <span className="absolute right-0">{formatNumber(gauge.max)}</span>
        </div>
      </div>
      <p className="mt-auto pt-2.5 text-xs text-muted-foreground tabular-nums">
        {gauge.rollOffs.length > 0
          ? gauge.rollOffs
              .map((rollOff) =>
                t(
                  "{0, plural, one {# rolls off} other {# roll off}} {1}",
                  rollOff.points,
                  formatUnixDateMedium(rollOff.date),
                ),
              )
              .join(" · ")
          : t("No active points. The score recovers as events roll off.")}
      </p>
    </div>
  );
}

/** What a segment of the gauge stands for: its event and when its points roll off. */
function SegmentNote({
  segment,
  event,
}: {
  segment: GaugeSegment;
  event: WorkerSafetyEventRow | undefined;
}) {
  const t = useT();
  if (segment.id === EARLIER_SEGMENT || !event || segment.occurredAt === null) {
    return (
      <span className="block font-medium">
        {t(
          "{0, plural, one {# point from earlier events} other {# points from earlier events}}",
          segment.points,
        )}
      </span>
    );
  }
  return (
    <>
      <span className="block font-medium">
        {t(
          "{0, plural, one {# point} other {# points}} · {1}",
          segment.points,
          safetyEventHeadline(event),
        )}
      </span>
      <span className="block">
        {segment.rollsOffAt
          ? t(
              "{0}. Rolls off {1}.",
              formatUnixDateMedium(segment.occurredAt),
              formatUnixDateMedium(segment.rollsOffAt),
            )
          : formatUnixDateMedium(segment.occurredAt)}
      </span>
    </>
  );
}
