import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { WorkerHosDailyLog } from "@/lib/graphql/telematics";
import { telematics } from "@/lib/queries/telematics";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { cn, metersToMiles } from "@trenova/shared/lib/utils";
import {
  buildLogSegments,
  dayLaneTotals,
  formatLogDay,
  HOS_LIVE_REFRESH_MS,
  type HosDaySlot,
} from "./hos-model";
import { formatHoursMinutes } from "@/lib/hours-format";
import { shortWeekday } from "../schedule/schedule-model";
import { LogGrid } from "./log-grid";

export type DailyLogPanelProps = {
  workerId: string;
  slots: readonly HosDaySlot[];
  selected: HosDaySlot;
  onSelect: (key: string) => void;
  /** The view's minute clock: today's open segment runs to it. */
  nowSec: number;
};

/**
 * The last seven days, one at a time: day buttons on a gliding plate (a dot under each certified
 * day), the paper log grid for the selected day and the day's ruled summary under it.
 */
export function DailyLogPanel({ workerId, slots, selected, onSelect, nowSec }: DailyLogPanelProps) {
  const t = useT();
  // Today's key is the view's own live read, refreshed there; a finished day does not change.
  const logsQuery = useQuery({
    ...telematics.workerHosLogs(workerId, selected.startAt, selected.endAt),
    staleTime: HOS_LIVE_REFRESH_MS,
  });
  const segments = buildLogSegments(logsQuery.data ?? [], selected.startAt, selected.endAt, nowSec);
  const totals = dayLaneTotals(segments, selected.dailyLog);
  const dayLabel = formatLogDay(selected.date);

  return (
    <SectionPanel title={t("Daily log")} hint={t("In your time zone")}>
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2.5 px-3 pt-3 pb-1">
        <SegmentedControl
          aria-label={t("Day")}
          value={selected.key}
          onValueChange={onSelect}
          items={slots.map((slot) => ({
            value: slot.key,
            label: shortWeekday(slot.date.getDay()),
            caption: (
              <span className="flex flex-col items-center">
                <span className="font-medium tabular-nums">{slot.date.getDate()}</span>
                <span
                  aria-hidden
                  className={cn(
                    "mt-0.5 size-1 rounded-full",
                    slot.dailyLog?.isCertified ? "bg-success" : "bg-transparent",
                  )}
                />
                {slot.dailyLog?.isCertified ? (
                  <span className="sr-only">{t("certified")}</span>
                ) : null}
              </span>
            ),
          }))}
        />
        <p className="flex items-baseline gap-2">
          <span className="font-medium">{dayLabel}</span>
          <span className="text-muted-foreground tabular-nums">
            {selected.isToday
              ? t("today, in progress")
              : selected.dailyLog?.isCertified && selected.dailyLog.certifiedAt
                ? t("certified {0}", formatUnixMonthDay(selected.dailyLog.certifiedAt))
                : t("not certified")}
          </span>
        </p>
      </div>
      {logsQuery.isPending ? (
        <div aria-busy aria-label={t("Loading the day's log")} className="px-3 pt-1 pb-3.5">
          <div aria-hidden className="h-[162px] w-full ui-shimmer rounded-md" />
        </div>
      ) : logsQuery.isError ? (
        <div className="px-3 pb-3">
          <ErrorState
            layout="compact"
            error={logsQuery.error}
            onRetry={() => void logsQuery.refetch()}
            title={t("The day's log could not be loaded")}
          />
        </div>
      ) : segments.length === 0 && !selected.dailyLog ? (
        <SectionPanelQuiet>{t("Nothing was logged on this day.")}</SectionPanelQuiet>
      ) : (
        <div className="px-3 pt-1 pb-3.5">
          <LogGrid
            segments={segments}
            dayStart={selected.startAt}
            dayEnd={selected.endAt}
            totals={totals}
            nowSec={selected.isToday ? nowSec : null}
            dayLabel={dayLabel}
            drawKey={selected.key}
          />
        </div>
      )}
      <DaySummary key={selected.key} dailyLog={selected.dailyLog} totals={totals} />
    </SectionPanel>
  );
}

function DaySummary({
  dailyLog,
  totals,
}: {
  dailyLog: WorkerHosDailyLog | null;
  totals: readonly [number, number, number, number];
}) {
  const t = useT();
  const vehicles = dailyLog?.vehicleNames ?? [];
  return (
    <DescriptionList
      layout="ruled"
      columns={4}
      className="animate-rise rounded-none border-0 border-t motion-reduce:animate-none"
    >
      <DescriptionItem label={t("Drive")} numeric valueClassName="font-mono text-xs">
        {formatHoursMinutes(t, totals[2])}
      </DescriptionItem>
      <DescriptionItem label={t("On duty")} numeric valueClassName="font-mono text-xs">
        {formatHoursMinutes(t, totals[3])}
      </DescriptionItem>
      <DescriptionItem label={t("Off duty")} numeric valueClassName="font-mono text-xs">
        {formatHoursMinutes(t, totals[0])}
      </DescriptionItem>
      <DescriptionItem label={t("Sleeper")} numeric valueClassName="font-mono text-xs">
        {formatHoursMinutes(t, totals[1])}
      </DescriptionItem>
      <DescriptionItem label={t("Distance")} numeric valueClassName="font-mono text-xs">
        {dailyLog
          ? t("{0} mi", Math.round(metersToMiles(dailyLog.driveDistanceMeters)))
          : t("Not reported")}
      </DescriptionItem>
      <DescriptionItem
        label={t("{0, plural, one {Vehicle} other {Vehicles}}", Math.max(vehicles.length, 1))}
        valueClassName="font-mono text-xs"
      >
        {vehicles.length > 0 ? vehicles.join(", ") : t("None reported")}
      </DescriptionItem>
      <DescriptionItem label={t("Certified")} valueClassName="font-mono text-xs">
        {dailyLog?.isCertified
          ? dailyLog.certifiedAt
            ? t("Yes, {0}", formatUnixMonthDay(dailyLog.certifiedAt))
            : t("Yes")
          : t("Not yet")}
      </DescriptionItem>
      {dailyLog?.shippingDocs ? (
        <DescriptionItem label={t("Shipping documents")} valueClassName="font-mono text-xs">
          {dailyLog.shippingDocs}
        </DescriptionItem>
      ) : null}
    </DescriptionList>
  );
}
