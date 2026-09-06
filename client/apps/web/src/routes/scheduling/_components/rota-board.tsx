import { type RotaBoard, type RotaBoardDay, type RotaBoardRow } from "@/lib/graphql/scheduling";
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  AVAILABILITY_TONES,
  DAY_LABELS,
  formatShiftDate,
  formatShiftWindow,
  minutesToClock,
  rotaStateTone,
  summariseRotaRow,
} from "@trenova/shared/lib/scheduling";
import { formatHours } from "@trenova/shared/lib/timesheet";
import { cn, initials } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon } from "lucide-react";

const SECONDS_IN_DAY = 86400;

export type RotaBoardProps = {
  rota: RotaBoard;
  onSelectDay?: (row: RotaBoardRow, day: RotaBoardDay) => void;
};

/**
 * The board. Columns are days and rows are people, because that is the shape a
 * dispatcher already reads a week in — the alternative, a column per person,
 * stops fitting at about eight drivers.
 */
export function RotaBoard({ rota, onSelectDay }: RotaBoardProps) {
  const columns = rota.rows[0]?.days ?? [];
  const today = getTodayDate();
  const todayIndex = columns.findIndex(
    (day) => day.date <= today && today < day.date + SECONDS_IN_DAY,
  );

  return (
    <div
      key={rota.weekStart}
      className="border-border/80 animate-in fade-in-0 overflow-x-auto rounded-xl border duration-200"
    >
      <table className="w-full min-w-[56rem] border-separate border-spacing-0 text-xs">
        <thead>
          <tr>
            <th className="bg-muted/40 sticky left-0 z-10 w-56 rounded-tl-xl px-3 py-2 text-left font-medium">
              Worker
            </th>
            {columns.map((day, index) => (
              <th
                key={day.date}
                className={cn(
                  "bg-muted/40 px-1 py-2 text-center font-medium",
                  index === todayIndex && "bg-primary/10",
                )}
              >
                <DayHeading date={day.date} today={index === todayIndex} />
              </th>
            ))}
            <th className="bg-muted/40 rounded-tr-xl px-3 py-2 text-right font-medium">Week</th>
          </tr>
        </thead>
        <tbody>
          {rota.rows.map((row) => {
            const summary = summariseRotaRow(row);
            return (
              <tr key={row.workerId} className="group transition-colors">
                <td className="bg-background group-hover:bg-muted/30 sticky left-0 z-10 border-t px-3 py-1.5 align-middle transition-colors">
                  <div className="flex items-center gap-2.5">
                    <Avatar className="size-7">
                      <AvatarFallback
                        className="text-[10px] font-medium"
                        style={
                          row.fleetColor
                            ? { backgroundColor: `${row.fleetColor}22`, color: row.fleetColor }
                            : undefined
                        }
                      >
                        {initials(...splitName(row.name))}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex min-w-0 flex-col leading-tight">
                      <span className="truncate font-medium">{row.name}</span>
                      <span className="text-muted-foreground flex items-center gap-1 truncate text-[11px]">
                        {row.shiftColor ? (
                          <span
                            className="size-1.5 shrink-0 rounded-full"
                            style={{ backgroundColor: row.shiftColor }}
                            aria-hidden
                          />
                        ) : null}
                        {row.shiftName ?? "No shift"}
                        {row.fleetCode ? ` · ${row.fleetCode}` : ""}
                      </span>
                    </div>
                  </div>
                </td>
                {row.days.map((day, index) => (
                  <td
                    key={day.date}
                    className={cn(
                      "group-hover:bg-muted/30 border-t px-0.5 py-1 align-middle transition-colors",
                      index === todayIndex && "bg-primary/5",
                    )}
                  >
                    <RotaCell row={row} day={day} onSelect={onSelectDay} />
                  </td>
                ))}
                <td className="group-hover:bg-muted/30 border-t px-3 py-1.5 text-right align-middle tabular-nums transition-colors">
                  <div className="flex flex-col items-end leading-tight">
                    <span className="font-medium">
                      {summary.scheduledDays}d
                      <span className="text-muted-foreground font-normal">
                        {" "}
                        · {formatHours(summary.hours * 60)}
                      </span>
                    </span>
                    {summary.conflicts > 0 ? (
                      <Badge variant="inactive" className="mt-0.5 gap-1">
                        <AlertTriangleIcon className="size-3" />
                        {summary.conflicts}
                      </Badge>
                    ) : null}
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function splitName(name: string): [string, string] {
  const [first = "", ...rest] = name.trim().split(/\s+/);
  return [first, rest.join(" ")];
}

function DayHeading({ date, today }: { date: number; today: boolean }) {
  const day = new Date(date * 1000);
  return (
    <span className="flex flex-col items-center leading-tight">
      <span className={cn(today && "text-primary")}>{DAY_LABELS[day.getUTCDay()]}</span>
      <span
        className={cn(
          "mt-0.5 grid size-5 place-items-center rounded-full font-normal tabular-nums",
          today ? "bg-primary text-primary-foreground" : "text-muted-foreground",
        )}
      >
        {day.getUTCDate()}
      </span>
    </span>
  );
}

function RotaCell({
  row,
  day,
  onSelect,
}: {
  row: RotaBoardRow;
  day: RotaBoardDay;
  onSelect?: (row: RotaBoardRow, day: RotaBoardDay) => void;
}) {
  const tone = rotaStateTone(day.state);
  const interactive = Boolean(onSelect);

  const cell = (
    <div
      className={cn(
        "relative flex h-12 w-full flex-col items-center justify-center gap-0.5 rounded-lg border px-1 transition-[transform,box-shadow] duration-150",
        tone.cell,
        day.isConflict && "ring-destructive/60 ring-offset-background ring-2 ring-offset-1",
        interactive && "cursor-pointer hover:-translate-y-px hover:shadow-sm",
      )}
    >
      {day.scheduled ? (
        <>
          <span className="leading-none font-medium tabular-nums">
            {minutesToClock(day.startMinute)}
          </span>
          <span className="text-[10px] leading-none opacity-75">
            {formatHours(day.durationMinutes)}
          </span>
        </>
      ) : (
        <span className={cn("size-1.5 rounded-full", tone.dot)} aria-hidden />
      )}
      {day.assignmentCount > 0 ? (
        <span className="bg-background/80 absolute top-1 right-1 rounded-full px-1 text-[9px] leading-none font-semibold tabular-nums">
          {day.assignmentCount}
        </span>
      ) : null}
    </div>
  );

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          interactive ? (
            <button
              type="button"
              className="w-full"
              onClick={() => onSelect?.(row, day)}
              aria-label={`${row.name}, ${formatShiftDate(day.date)}: ${tone.label}`}
            />
          ) : (
            <div className="w-full" />
          )
        }
      >
        {cell}
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-56">
        <p className="font-medium">
          {tone.label} · {formatShiftDate(day.date)}
        </p>
        {day.scheduled ? (
          <p className="text-muted-foreground">
            {formatShiftWindow(day.startMinute, day.durationMinutes)}
          </p>
        ) : null}
        {day.preference ? (
          <p className="text-muted-foreground">
            Stated: {AVAILABILITY_TONES[day.preference]?.label ?? day.preference}
          </p>
        ) : null}
        {day.assignmentCount > 0 ? (
          <p className="text-muted-foreground">
            {day.assignmentCount} load{day.assignmentCount === 1 ? "" : "s"} already assigned
          </p>
        ) : null}
        {day.isConflict ? (
          <p className="text-destructive">Rostered on a day they cannot work</p>
        ) : null}
      </TooltipContent>
    </Tooltip>
  );
}

export function rotaWeekLabel(weekStart: number, weeks: number): string {
  const start = new Date(weekStart * 1000);
  const end = new Date((weekStart + weeks * 7 * SECONDS_IN_DAY - SECONDS_IN_DAY) * 1000);
  const format = (date: Date) =>
    `${date.toLocaleString("en-US", { month: "short", timeZone: "UTC" })} ${date.getUTCDate()}`;
  return `${format(start)} – ${format(end)}, ${end.getUTCFullYear()}`;
}
