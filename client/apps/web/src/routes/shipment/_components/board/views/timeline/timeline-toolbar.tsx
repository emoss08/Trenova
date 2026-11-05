import { ACTIVE_STAGES, STAGE_META, type ActiveStage } from "@/lib/shipment-board/stage";
import type { TimelineRange, TimelineRowMode } from "@/lib/shipment-board/timeline";
import { ChevronLeftIcon, ChevronRightIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, formatUnixWeekday } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { TimelineScale } from "@/components/timeline/timeline-scale";

type TimelineToolbarProps = {
  scale: TimelineScale;
  range: TimelineRange;
  mode: TimelineRowMode;
  onModeChange: (mode: TimelineRowMode) => void;
  onRangeChange: (range: TimelineRange) => void;
  onEarlier: () => void;
  onLater: () => void;
  onNow: () => void;
  counts: Record<ActiveStage, number>;
  hiddenStages: ReadonlySet<ActiveStage>;
  onToggleStage: (stage: ActiveStage) => void;
};

/** The timeline's one row of controls: rows, range and stages. */
export function TimelineToolbar({
  scale,
  range,
  mode,
  onModeChange,
  onRangeChange,
  onEarlier,
  onLater,
  onNow,
  counts,
  hiddenStages,
  onToggleStage,
}: TimelineToolbarProps) {
  const t = useT();
  const first = scale.days[0];
  const last = scale.days[scale.days.length - 2];
  const dayLabel = (day: number) => t("{0}, {1}", formatUnixWeekday(day), formatUnixMonthDay(day));
  const rangeLabel =
    first === last ? dayLabel(first) : t("{0} to {1}", dayLabel(first), dayLabel(last));

  return (
    <div className="border-subtle bg-card flex min-h-10 shrink-0 flex-wrap items-center gap-x-3 gap-y-1.5 border-b px-2.5 py-1.5">
      <SegmentedControl<TimelineRowMode>
        aria-label={t("Rows")}
        value={mode}
        onValueChange={onModeChange}
        items={[
          { value: "shipments", label: t("Shipments") },
          { value: "coverage", label: t("Drivers & carriers") },
        ]}
      />
      <div className="flex items-center gap-1">
        <Button variant="outline" size="icon-sm" aria-label={t("Earlier")} onClick={onEarlier}>
          <ChevronLeftIcon className="size-3.5" />
        </Button>
        <span className="min-w-36 text-center text-sm font-medium whitespace-nowrap">
          {rangeLabel}
        </span>
        <Button variant="outline" size="icon-sm" aria-label={t("Later")} onClick={onLater}>
          <ChevronRightIcon className="size-3.5" />
        </Button>
        <Button variant="outline" size="sm" onClick={onNow}>
          {t("Now")}
        </Button>
      </div>
      <SegmentedControl<TimelineRange>
        aria-label={t("Range")}
        value={range}
        onValueChange={onRangeChange}
        items={[
          { value: "day", label: t("Day") },
          { value: "3d", label: t("3 days") },
          { value: "week", label: t("Week") },
        ]}
      />
      <div
        role="group"
        aria-label={t("Show stages")}
        className="ml-auto flex flex-wrap items-center gap-1"
      >
        {ACTIVE_STAGES.filter((stage) => counts[stage] > 0 || hiddenStages.has(stage)).map(
          (stage) => {
            const on = !hiddenStages.has(stage);
            return (
              <button
                key={stage}
                type="button"
                aria-pressed={on}
                onClick={() => onToggleStage(stage)}
                className={cn(
                  "ui-focus-ring inline-flex h-6 items-center gap-1.5 rounded-md border px-2 text-xs whitespace-nowrap",
                  on
                    ? "bg-card text-foreground hover:bg-surface-hover"
                    : "text-muted-foreground border-dashed",
                )}
              >
                <span
                  aria-hidden
                  className={cn(
                    "size-2 rounded-full",
                    STAGE_META[stage].swatchClassName,
                    !on && "opacity-40",
                  )}
                />
                {t(STAGE_META[stage].label)}
                <span className="text-muted-foreground font-mono tabular-nums">
                  {counts[stage]}
                </span>
              </button>
            );
          },
        )}
      </div>
    </div>
  );
}
