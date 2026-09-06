import { formatUnixWeekday } from "@trenova/shared/lib/date";
import { ptoTypeMeta } from "@trenova/shared/lib/pto";
import { cn } from "@trenova/shared/lib/utils";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useMemo } from "react";
import { DAY_SECONDS } from "./calendar-layout";

export const WHOS_OUT_DAYS = 7;

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

function workerName(pto: WorkerPTO): string {
  const first = pto.worker?.firstName ?? "";
  const last = pto.worker?.lastName ?? "";
  return `${first} ${last}`.trim() || "Unknown worker";
}

export function WhosOutStrip({
  items,
  todayUnix,
}: {
  items: readonly WorkerPTO[];
  todayUnix: number;
}) {
  const days = useMemo(() => buildWhosOut(items, todayUnix), [items, todayUnix]);
  const totalOut = days[0]?.out.length ?? 0;

  return (
    <div className="flex shrink-0 flex-col gap-1" data-testid="whos-out-strip">
      <p className="text-muted-foreground text-[11px] font-medium uppercase">
        Who&apos;s out · {totalOut} today
      </p>
      <div className="flex gap-1 overflow-x-auto pb-1">
        {days.map((day) => (
          <div
            key={day.unix}
            className={cn(
              "bg-muted/30 min-w-[104px] flex-1 rounded-md border px-2 py-1.5",
              day.out.length > 0 && "border-primary/30",
            )}
          >
            <p className="text-[11px] font-medium">
              {day.label}
              <span className="text-muted-foreground ml-1 tabular-nums">{day.out.length}</span>
            </p>
            <ul className="mt-0.5 flex flex-col gap-0.5">
              {day.out.slice(0, 3).map((pto) => (
                <li key={pto.id} className="flex items-center gap-1 truncate text-[11px]">
                  <span
                    className={cn("size-1.5 shrink-0 rounded-full", ptoTypeMeta(pto.type).barClass)}
                  />
                  <span className="truncate">{workerName(pto)}</span>
                </li>
              ))}
              {day.out.length > 3 ? (
                <li className="text-muted-foreground text-[10px]">+{day.out.length - 3} more</li>
              ) : null}
            </ul>
          </div>
        ))}
      </div>
    </div>
  );
}
