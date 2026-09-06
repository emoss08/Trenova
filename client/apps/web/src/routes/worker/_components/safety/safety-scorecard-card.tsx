import type { SafetyScorecard } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { safetyRatingMeta, scoreRingValue, summariseInspections } from "@trenova/shared/lib/safety";
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

export function SafetyScorecardCard({
  scorecard,
  canRecord,
  canRecognise,
  onRecordEvent,
  onRecognise,
}: SafetyScorecardCardProps) {
  const meta = safetyRatingMeta(scorecard.rating as SafetyRating);
  const pointsTone =
    scorecard.activePoints >= scorecard.pointsAtRiskThreshold
      ? "critical"
      : scorecard.activePoints >= scorecard.pointsWatchThreshold
        ? "warning"
        : "muted";

  return (
    <div
      data-testid="safety-scorecard"
      className="bg-card border-border flex flex-wrap items-center gap-4 rounded-xl border p-4"
    >
      <RingGauge
        value={scoreRingValue(scorecard.score)}
        size={72}
        strokeWidth={7}
        tone={meta.ringTone}
        aria-label="Safety score"
      >
        <span className="text-sm font-semibold tabular-nums">{scorecard.score}</span>
      </RingGauge>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex justify-between">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold">Safety scorecard</h3>
            <Badge variant={meta.badgeVariant}>{meta.label}</Badge>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            {canRecognise ? (
              <Button size="sm" variant="outline" onClick={onRecognise}>
                <AwardIcon className="size-3.5" />
                Add recognition
              </Button>
            ) : null}
            {canRecord ? (
              <Button size="sm" onClick={onRecordEvent}>
                <PlusIcon className="size-3.5" />
                Record event
              </Button>
            ) : null}
          </div>
        </div>
        <p className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-xs">
          <span>{summariseInspections(scorecard)}</span>
          <span aria-hidden>·</span>
          <span>
            {scorecard.daysSinceLastEvent == null
              ? "No accidents, incidents or citations on record"
              : `${scorecard.daysSinceLastEvent} days since the last accident, incident or citation`}
          </span>
        </p>
        <div className="mt-1 flex flex-wrap gap-2">
          <Tile
            label={`of ${scorecard.pointsAtRiskThreshold} before at-risk`}
            value={scorecard.activePoints}
            tone={pointsTone}
          />
          <Tile label="Accidents" value={scorecard.accidents} tone="critical" />
          <Tile label="Preventable" value={scorecard.preventableAccidents} tone="critical" />
          <Tile label="Citations" value={scorecard.citations} tone="warning" />
          <Tile label="Out of service" value={scorecard.outOfServiceOrders} tone="critical" />
          <Tile label="Open" value={scorecard.openEvents} tone="warning" />
          <Tile label="Recognition" value={scorecard.recognitions} tone="success" />
        </div>
        {scorecard.highestDiscipline ? (
          <p className="text-muted-foreground mt-1 text-[11px]">
            {scorecard.activeDiscipline} active disciplinary action
            {scorecard.activeDiscipline === 1 ? "" : "s"}, highest{" "}
            {DISCIPLINARY_LEVEL_LABELS[
              scorecard.highestDiscipline as DisciplinaryLevel
            ]?.toLowerCase() ?? scorecard.highestDiscipline}
            .
          </p>
        ) : null}
      </div>
    </div>
  );
}

function Tile({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "success" | "warning" | "critical" | "muted";
}) {
  const active = value > 0;
  return (
    <div
      className={cn(
        "flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs",
        !active && "text-muted-foreground border-dashed",
        active &&
          tone === "success" &&
          "border-green-500/40 bg-green-500/10 text-green-700 dark:text-green-400",
        active &&
          tone === "warning" &&
          "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-400",
        active &&
          tone === "critical" &&
          "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-400",
        active && tone === "muted" && "border-border bg-muted/50",
      )}
    >
      <span className="font-semibold tabular-nums">{value}</span>
      <span>{label}</span>
    </div>
  );
}
