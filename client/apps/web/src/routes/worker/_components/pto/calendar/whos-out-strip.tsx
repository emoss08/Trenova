import { useT } from "@trenova/shared/i18n/use-t";
import {
  Avatar,
  AvatarFallback,
  AvatarGroup,
  AvatarGroupCount,
  AvatarImage,
} from "@trenova/shared/components/ui/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixWeekday } from "@trenova/shared/lib/date";
import { ptoTypeMeta } from "@trenova/shared/lib/pto";
import { cn } from "@trenova/shared/lib/utils";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useMemo } from "react";
import { ptoWorkerInitials, ptoWorkerName } from "../pto-worker";
import { DAY_SECONDS } from "./calendar-layout";

export const WHOS_OUT_DAYS = 7;
const MAX_FACES = 3;

export type WhosOutDay = {
  unix: number;
  label: string;
  out: WorkerPTO[];
};

export function buildWhosOut(
  items: readonly WorkerPTO[],
  todayUnix: number,
  days: number = WHOS_OUT_DAYS,
): WhosOutDay[] {
  const result: WhosOutDay[] = [];
  for (let i = 0; i < days; i += 1) {
    const dayStart = todayUnix + i * DAY_SECONDS;
    const dayEnd = dayStart + DAY_SECONDS - 1;
    const out = items.filter(
      (pto) => pto.status === "Approved" && pto.startDate <= dayEnd && pto.endDate >= dayStart,
    );
    result.push({
      unix: dayStart,
      label: i === 0 ? "Today" : formatUnixWeekday(dayStart),
      out,
    });
  }
  return result;
}

export type WhosOutStripProps = {
  items: readonly WorkerPTO[];
  todayUnix: number;
  highlightedDay?: number | null;
  onHighlightDay?: (unix: number | null) => void;
  onSelectDay?: (unix: number) => void;
};

/**
 * The next seven days as a row of tiles, each carrying the faces of whoever is
 * approved to be away. Hovering a tile lights the matching day on the grid and
 * clicking it takes the calendar there, so the strip doubles as a jump bar.
 */
export function WhosOutStrip({
  items,
  todayUnix,
  highlightedDay,
  onHighlightDay,
  onSelectDay,
}: WhosOutStripProps) {
  const t = useT();

  const days = useMemo(() => buildWhosOut(items, todayUnix), [items, todayUnix]);
  const totalToday = days[0]?.out.length ?? 0;
  const distinctThisWeek = useMemo(() => {
    const ids = new Set<string>();
    for (const day of days) for (const pto of day.out) ids.add(pto.id ?? `${pto.workerId}`);
    return ids.size;
  }, [days]);

  return (
    <section className="flex shrink-0 flex-col gap-1.5" data-testid="whos-out-strip">
      <div className="flex items-baseline justify-between">
        <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
          {t("Who's out")}
        </p>
        <p className="text-muted-foreground text-[11px] tabular-nums">
          {t("{0} today · {1} this week", totalToday, distinctThisWeek)}
        </p>
      </div>
      <div className="grid grid-cols-7 gap-1">
        {days.map((day, index) => (
          <DayTile
            key={day.unix}
            day={day}
            isToday={index === 0}
            highlighted={highlightedDay === day.unix}
            onHighlightDay={onHighlightDay}
            onSelectDay={onSelectDay}
          />
        ))}
      </div>
    </section>
  );
}

function DayTile({
  day,
  isToday,
  highlighted,
  onHighlightDay,
  onSelectDay,
}: {
  day: WhosOutDay;
  isToday: boolean;
  highlighted: boolean;
  onHighlightDay?: (unix: number | null) => void;
  onSelectDay?: (unix: number) => void;
}) {
  const t = useT();

  const count = day.out.length;
  const dateNumber = new Date(day.unix * 1000).getDate();
  const tile = (
    <button
      type="button"
      aria-label={`${day.label}: ${count} out`}
      onClick={() => onSelectDay?.(day.unix)}
      onMouseEnter={() => onHighlightDay?.(day.unix)}
      onMouseLeave={() => onHighlightDay?.(null)}
      onFocus={() => onHighlightDay?.(day.unix)}
      onBlur={() => onHighlightDay?.(null)}
      className={cn(
        "focus-visible:ring-ring/50 flex min-w-0 flex-col gap-1 rounded-lg border px-2 py-1.5 text-left transition-colors outline-none focus-visible:ring-[3px]",
        isToday ? "border-primary/40 bg-primary/5" : "bg-accent/40 border-transparent",
        highlighted && "bg-accent border-border",
        count > 0 ? "hover:bg-accent" : "hover:bg-accent/70",
      )}
    >
      <span className="flex items-baseline justify-between gap-1 leading-none">
        <span className={cn("truncate text-[11px] font-medium", isToday && "text-primary")}>
          {t(day.label)}
        </span>
        <span className="text-muted-foreground text-[10px] tabular-nums">{dateNumber}</span>
      </span>
      {count > 0 ? (
        <AvatarGroup className="-space-x-1.5">
          {day.out.slice(0, MAX_FACES).map((pto) => (
            <Avatar key={pto.id} className="size-5 rounded-full after:rounded-full">
              <AvatarImage
                src={pto.worker?.profilePicUrl ?? undefined}
                alt=""
                className="rounded-full"
              />
              <AvatarFallback className="rounded-full text-[9px] font-medium">
                {ptoWorkerInitials(pto)}
              </AvatarFallback>
            </Avatar>
          ))}
          {count > MAX_FACES ? (
            <AvatarGroupCount className="size-5 text-[9px] font-medium">
              +{count - MAX_FACES}
            </AvatarGroupCount>
          ) : null}
        </AvatarGroup>
      ) : (
        <span className="text-muted-foreground/50 h-5 text-[11px] leading-5">{t("Nobody")}</span>
      )}
    </button>
  );

  if (count === 0) return tile;

  return (
    <Tooltip>
      <TooltipTrigger render={tile} />
      <TooltipContent side="bottom" className="max-w-56">
        <ul className="flex flex-col gap-0.5">
          {day.out.map((pto) => (
            <li key={pto.id} className="flex items-center gap-1.5">
              <span
                className={cn("size-1.5 shrink-0 rounded-full", ptoTypeMeta(pto.type).dotClass)}
                aria-hidden
              />
              <span className="truncate">{ptoWorkerName(pto)}</span>
              <span className="opacity-70">· {t(ptoTypeMeta(pto.type).label)}</span>
            </li>
          ))}
        </ul>
      </TooltipContent>
    </Tooltip>
  );
}
