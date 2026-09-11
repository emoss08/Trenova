import { useT } from "@trenova/shared/i18n/use-t";
import { type RotaBoard, type RotaBoardDay, type RotaBoardRow } from "@/lib/graphql/scheduling";
import {
  coverageByDay,
  coverageTone,
  peakCoverage,
  rotaCellMode,
  todayColumnIndex,
  type DayCoverage,
  type RotaCellMode,
  type RotaDensity,
} from "@/lib/scheduling-board";
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
import { useMemo } from "react";

const SECONDS_IN_DAY = 86400;

export type RotaBoardProps = {
  rota: RotaBoard;
  /** The rows to draw; defaults to every row on the rota. A search narrows it. */
  rows?: readonly RotaBoardRow[];
  density?: RotaDensity;
  onSelectDay?: (row: RotaBoardRow, day: RotaBoardDay) => void;
};

/**
 * The board. Columns are days and rows are people, because that is the shape a
 * dispatcher already reads a week in — the alternative, a column per person,
 * stops fitting at about eight drivers.
 *
 * The board scrolls inside its own box with the day headings and the cover
 * row pinned, so a forty-driver roster is read against the same columns from
 * top to bottom rather than against a heading that scrolled away.
 */
export function RotaBoard({
  rota,
  rows = rota.rows,
  density = "comfortable",
  onSelectDay,
}: RotaBoardProps) {
  const t = useT();

  const columns = rota.rows[0]?.days ?? [];
  const today = getTodayDate();
  const todayIndex = todayColumnIndex(columns, today);
  const compact = density === "compact";
  const cellMode = rotaCellMode(density, rota.weeks);
  // Cover is read across the whole rota, not the rows a search left: a day is
  // thin or not regardless of who the reader is looking for.
  const coverage = useMemo(() => coverageByDay(rota.rows), [rota.rows]);
  const peak = peakCoverage(coverage);

  return (
    <div
      key={rota.weekStart}
      data-density={density}
      data-cell-mode={cellMode}
      className="border-border animate-in fade-in-0 max-h-[75vh] overflow-auto rounded-lg border duration-200"
    >
      <table
        className={cn(
          "w-full border-separate border-spacing-0 text-xs",
          cellMode === "block" ? "min-w-[48rem]" : "min-w-[56rem]",
        )}
      >
        <thead className="sticky top-0 z-20">
          <tr>
            <th
              className={cn(
                "bg-muted sticky left-0 z-30 rounded-tl-lg px-3 text-left font-medium",
                compact ? "w-40 py-1" : "w-56 py-2",
              )}
            >
              {t("Worker")}
            </th>
            {columns.map((day, index) => (
              <th
                key={day.date}
                className={cn(
                  "bg-muted px-0.5 text-center font-medium",
                  compact ? "py-1" : "py-2",
                  index === todayIndex && "bg-primary/10",
                )}
              >
                <DayHeading date={day.date} today={index === todayIndex} compact={compact} />
              </th>
            ))}
            <th
              className={cn(
                "bg-muted rounded-tr-lg px-3 text-right font-medium",
                compact ? "py-1" : "py-2",
              )}
            >
              {t("Week")}
            </th>
          </tr>
          <tr>
            <th
              scope="row"
              className="bg-background sticky left-0 z-30 border-t border-b px-3 py-1 text-left align-middle text-[11px] font-medium"
            >
              <span className="text-muted-foreground">{t("Cover")}</span>
            </th>
            {columns.map((day, index) => (
              <td
                key={day.date}
                className={cn(
                  "bg-background border-t border-b px-0.5 py-1 align-bottom",
                  index === todayIndex && "bg-primary/5",
                )}
              >
                <CoverageCell coverage={coverage[index]} peak={peak} compact={compact} />
              </td>
            ))}
            <td className="bg-background text-muted-foreground border-t border-b px-3 py-1 text-right align-middle text-[11px] tabular-nums">
              {t("{0} of {1}", rows.length, rota.rows.length)}
            </td>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const summary = summariseRotaRow(row);
            return (
              <tr key={row.workerId} className="group transition-colors">
                <td
                  className={cn(
                    "bg-background group-hover:bg-muted/30 sticky left-0 z-10 border-t px-3 align-middle transition-colors",
                    compact ? "py-0.5" : "py-1.5",
                  )}
                >
                  <WorkerCell row={row} compact={compact} />
                </td>
                {row.days.map((day, index) => (
                  <td
                    key={day.date}
                    className={cn(
                      "group-hover:bg-muted/30 border-t px-0.5 align-middle transition-colors",
                      compact ? "py-0.5" : "py-1",
                      index === todayIndex && "bg-primary/5",
                    )}
                  >
                    <RotaCell
                      row={row}
                      day={day}
                      mode={cellMode}
                      compact={compact}
                      onSelect={onSelectDay}
                    />
                  </td>
                ))}
                <td
                  className={cn(
                    "group-hover:bg-muted/30 border-t px-3 text-right align-middle tabular-nums transition-colors",
                    compact ? "py-0.5" : "py-1.5",
                  )}
                >
                  <WeekTotal summary={summary} compact={compact} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function WorkerCell({ row, compact }: { row: RotaBoardRow; compact: boolean }) {
  const shift = (
    <span className="text-muted-foreground flex min-w-0 items-center gap-1 truncate text-[11px]">
      {row.shiftColor ? (
        <span
          className="size-1.5 shrink-0 rounded-full"
          style={{ backgroundColor: row.shiftColor }}
          aria-hidden
        />
      ) : null}
      <span className="truncate">
        {row.shiftName ?? "No shift"}
        {row.fleetCode ? ` · ${row.fleetCode}` : ""}
      </span>
    </span>
  );

  if (compact) {
    return (
      <div className="flex min-w-0 items-center gap-2 leading-tight">
        <span className="truncate font-medium">{row.name}</span>
        {shift}
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2.5">
      <Avatar className="size-7">
        <AvatarFallback className="text-[10px] font-medium">
          {initials(...splitName(row.name))}
        </AvatarFallback>
      </Avatar>
      <div className="flex min-w-0 flex-col leading-tight">
        <span className="truncate font-medium">{row.name}</span>
        {shift}
      </div>
    </div>
  );
}

function WeekTotal({
  summary,
  compact,
}: {
  summary: ReturnType<typeof summariseRotaRow>;
  compact: boolean;
}) {
  const t = useT();

  const conflicts =
    summary.conflicts > 0 ? (
      <Badge variant="inactive" className={cn("gap-1", !compact && "mt-0.5")}>
        <AlertTriangleIcon className="size-3" />
        {summary.conflicts}
      </Badge>
    ) : null;
  const total = (
    <span className="font-medium">
      {t("{0}d", summary.scheduledDays)}
      <span className="text-muted-foreground font-normal">
        {" "}
        · {formatHours(summary.hours * 60)}
      </span>
    </span>
  );

  if (compact) {
    return (
      <div className="flex items-center justify-end gap-1.5 leading-tight">
        {conflicts}
        {total}
      </div>
    );
  }
  return (
    <div className="flex flex-col items-end leading-tight">
      {total}
      {conflicts}
    </div>
  );
}

/**
 * One day's headcount as a bar scaled against the busiest day on the board.
 * A thin day is set in the foreground weight rather than a colour, so the
 * column reads before the number does.
 */
function CoverageCell({
  coverage,
  peak,
  compact,
}: {
  coverage: DayCoverage | undefined;
  peak: number;
  compact: boolean;
}) {
  const covered = coverage?.covered ?? 0;
  const expected = coverage?.expected ?? 0;
  const tone = coverageTone(covered, peak);
  const share = peak > 0 ? Math.round((covered / peak) * 100) : 0;
  return (
    <div
      role="img"
      aria-label={
        coverage
          ? `${formatShiftDate(coverage.date)}: ${covered} of ${expected} rostered can work`
          : "No cover"
      }
      className={cn("flex flex-col items-center", compact ? "gap-0.5" : "gap-1")}
    >
      <span
        className={cn(
          "font-mono leading-none tabular-nums",
          compact ? "text-[10px]" : "text-[11px]",
          tone === "strong" ? "text-foreground" : "text-muted-foreground",
          tone === "thin" && "text-warning font-semibold",
        )}
      >
        {covered}
        {expected > covered ? (
          <span className="text-muted-foreground font-normal">/{expected}</span>
        ) : null}
      </span>
      <span className="bg-muted flex h-1 w-full max-w-16 overflow-hidden rounded-full">
        <span
          aria-hidden
          className={cn(
            "h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none",
            tone === "thin" ? "bg-warning" : "bg-brand/60",
          )}
          style={{ width: `${share}%` }}
        />
      </span>
    </div>
  );
}

function splitName(name: string): [string, string] {
  const [first = "", ...rest] = name.trim().split(/\s+/);
  return [first, rest.join(" ")];
}

function DayHeading({ date, today, compact }: { date: number; today: boolean; compact: boolean }) {
  const day = new Date(date * 1000);
  if (compact) {
    return (
      <span
        className={cn(
          "inline-flex items-center gap-1 rounded-full px-1.5 py-px leading-tight tabular-nums",
          today ? "bg-primary text-primary-foreground" : "text-foreground",
        )}
      >
        <span>{DAY_LABELS[day.getUTCDay()]}</span>
        <span className={cn("font-normal", !today && "text-muted-foreground")}>
          {day.getUTCDate()}
        </span>
      </span>
    );
  }
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

const CELL_HEIGHT: Record<RotaCellMode, string> = {
  detail: "h-12",
  time: "h-7",
  block: "h-6",
};

function RotaCell({
  row,
  day,
  mode,
  compact,
  onSelect,
}: {
  row: RotaBoardRow;
  day: RotaBoardDay;
  mode: RotaCellMode;
  compact: boolean;
  onSelect?: (row: RotaBoardRow, day: RotaBoardDay) => void;
}) {
  const t = useT();

  const tone = rotaStateTone(day.state);
  const interactive = Boolean(onSelect);
  const height = mode === "block" && !compact ? "h-8" : CELL_HEIGHT[mode];

  const cell = (
    <div
      data-cell={mode}
      className={cn(
        "relative flex w-full flex-col items-center justify-center gap-0.5 rounded-md border px-1 transition-colors",
        height,
        tone.cell,
        day.isConflict && "ring-destructive/60 ring-offset-background ring-2 ring-offset-1",
        interactive && "cursor-pointer hover:brightness-95",
      )}
    >
      {mode === "block" ? null : day.scheduled ? (
        <>
          <span
            className={cn(
              "leading-none font-medium tabular-nums",
              mode === "time" && "text-[11px]",
            )}
          >
            {minutesToClock(day.startMinute)}
          </span>
          {mode === "detail" ? (
            <span className="text-[10px] leading-none opacity-75">
              {formatHours(day.durationMinutes)}
            </span>
          ) : null}
        </>
      ) : (
        <span className={cn("size-1.5 rounded-full", tone.dot)} aria-hidden />
      )}
      {day.assignmentCount > 0 ? (
        <span
          className={cn(
            "bg-background/80 absolute rounded-full px-1 text-[9px] leading-none font-semibold tabular-nums",
            mode === "detail" ? "top-1 right-1" : "top-0.5 right-0.5",
          )}
        >
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
            {t("Stated: {0}", AVAILABILITY_TONES[day.preference]?.label ?? day.preference)}
          </p>
        ) : null}
        {day.assignmentCount > 0 ? (
          <p className="text-muted-foreground">
            {t("{0} load{1} already assigned", day.assignmentCount, day.assignmentCount === 1 ? "" : "s")}
          </p>
        ) : null}
        {day.isConflict ? (
          <p className="text-destructive">{t("Rostered on a day they cannot work")}</p>
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
