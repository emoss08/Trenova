import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@trenova/shared/components/ui/hover-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan, formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

/** Within this many minutes of the window an arrival reads as on time. */
export const ON_TIME_GRACE_MINUTES = 15;

type ArrivalWindowCellProps = {
  /** When the truck is expected, or got there, in Unix seconds. */
  arrival: number | null | undefined;
  /** Whether `arrival` already happened rather than being an estimate. */
  arrived?: boolean;
  windowStart: number | null | undefined;
  windowEnd: number | null | undefined;
};

export type ArrivalVerdict = {
  /** Minutes past the window's end (positive) or before its start (negative); 0 inside it. */
  minutesOff: number;
  tone: "early" | "on-time" | "late";
};

/** How an arrival sits against its window, with the grace an on-time arrival is given. */
export function arrivalVerdict(
  arrival: number,
  windowStart: number,
  windowEnd: number | null | undefined,
): ArrivalVerdict {
  const end = windowEnd ?? windowStart;
  if (arrival > end) {
    const minutesOff = (arrival - end) / 60;
    return { minutesOff, tone: minutesOff <= ON_TIME_GRACE_MINUTES ? "on-time" : "late" };
  }
  if (arrival < windowStart) {
    const minutesOff = (arrival - windowStart) / 60;
    return { minutesOff, tone: -minutesOff <= ON_TIME_GRACE_MINUTES ? "on-time" : "early" };
  }
  return { minutesOff: 0, tone: "on-time" };
}

const MOMENT = { showTimeZone: false, showSeconds: false } as const;
const DAY_ONLY = { showTime: false } as const;
const TIME_ONLY = { showDate: false, showTimeZone: false, showSeconds: false } as const;

function formatWindow(start: number, end: number | null | undefined): string {
  if (!end || end === start) return formatToUserTimezone(start, MOMENT);
  const sameDay =
    formatToUserTimezone(start, DAY_ONLY) === formatToUserTimezone(end, DAY_ONLY);
  return `${formatToUserTimezone(start, MOMENT)} – ${formatToUserTimezone(
    end,
    sameDay ? TIME_ONLY : MOMENT,
  )}`;
}

const TONE_BADGE = {
  late: "danger",
  "on-time": "success",
  early: "info",
} as const;

/**
 * An arrival against its appointment, read as a verdict: a badge that says late,
 * on time or early and by how much, over the arrival and the window it was due in.
 * Hovering lays out the window, the arrival, the difference and the grace allowed.
 */
export function ArrivalWindowCell({
  arrival,
  arrived = false,
  windowStart,
  windowEnd,
}: ArrivalWindowCellProps) {
  const t = useT();
  if (!windowStart || !arrival) {
    return <span className="text-muted-foreground font-mono text-sm">—</span>;
  }

  const verdict = arrivalVerdict(arrival, windowStart, windowEnd);
  const off = formatMinutesSpan(Math.abs(verdict.minutesOff));
  const badge =
    verdict.tone === "late"
      ? t("Late {0}", off)
      : verdict.tone === "early"
        ? t("Early {0}", off)
        : t("On time");
  const difference =
    verdict.minutesOff === 0
      ? t("Inside the window")
      : verdict.minutesOff > 0
        ? t("{0} after the window closes", off)
        : t("{0} before the window opens", off);

  return (
    <HoverCard>
      <HoverCardTrigger
        delay={300}
        render={<span className="flex w-full min-w-0 cursor-default flex-col justify-center gap-1" />}
      >
        <span className="flex min-w-0 items-center gap-2">
          <Badge variant={TONE_BADGE[verdict.tone]} className="tabular-nums">
            {badge}
          </Badge>
          {!arrived ? (
            <span className="text-muted-foreground truncate text-xs">{t("Projected")}</span>
          ) : null}
        </span>
        <span className="text-muted-foreground truncate font-mono text-xs tabular-nums in-data-[density=compact]:hidden">
          {arrived ? t("Arrived") : t("ETA")}{" "}
          <span className="text-foreground">{formatToUserTimezone(arrival, MOMENT)}</span>
        </span>
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-80 p-0">
        <div className="border-border flex items-center justify-between gap-3 border-b px-3 py-2">
          <span className="text-sm font-medium">{t("Delivery appointment")}</span>
          <Badge variant={TONE_BADGE[verdict.tone]}>{badge}</Badge>
        </div>
        <DescriptionList layout="split" className="px-3 py-1">
          <DescriptionItem label={t("Window")} numeric>
            {formatWindow(windowStart, windowEnd)}
          </DescriptionItem>
          <DescriptionItem label={arrived ? t("Arrived") : t("Expected")} numeric>
            {formatToUserTimezone(arrival, MOMENT)}
          </DescriptionItem>
          <DescriptionItem
            label={t("Difference")}
            valueClassName={cn(verdict.tone === "late" && "text-danger")}
          >
            {difference}
          </DescriptionItem>
          <DescriptionItem label={t("Grace")} numeric>
            {t("{0} either side", formatMinutesSpan(ON_TIME_GRACE_MINUTES))}
          </DescriptionItem>
        </DescriptionList>
      </HoverCardContent>
    </HoverCard>
  );
}
