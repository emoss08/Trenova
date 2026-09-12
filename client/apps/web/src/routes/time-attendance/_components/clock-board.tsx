import { useT } from "@trenova/shared/i18n/use-t";
import type { OpenTimeEntryRow } from "@/lib/graphql/timesheet";
import { isOverlong, rankRunning } from "@/lib/time-attendance";
import {
  Avatar,
  AvatarBadge,
  AvatarFallback,
  AvatarImage,
} from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { formatHours } from "@trenova/shared/lib/timesheet";
import { cn, initials } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, SquareIcon, TimerIcon, UsersIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";

const STAGGER_LIMIT = 10;
/** The length of the meter: a working day, the point past which a punch reads as forgotten. */
const METER_MINUTES = 12 * 60;

function formatPunchTime(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, { hour: "numeric", minute: "2-digit" });
}

export function entryWorkerName(entry: OpenTimeEntryRow): string {
  return entry.worker ? `${entry.worker.firstName} ${entry.worker.lastName}` : entry.workerId;
}

type ClockBoardProps = {
  entries: readonly OpenTimeEntryRow[] | undefined;
  isLoading: boolean;
  now: number;
  teamOnly: boolean;
  onTeamOnlyChange: (value: boolean) => void;
  /** Whether the viewer may punch somebody out from here. */
  canPunch: boolean;
  /** Which worker a punch-out is in flight for, so only that row shows busy. */
  punchingOut: string | null;
  onPunchOut: (workerId: string) => void;
  onPick: (workerId: string) => void;
  selectedWorkerId: string;
};

/**
 * Everyone on the clock this minute, longest-running first. It answers the
 * question a manager walks in with, before they have picked anybody: who is
 * still here, and has anybody forgotten to punch out. Each row carries a
 * meter the length of a working day, so a shift that has run too long is
 * seen before it is read.
 */
export function ClockBoard({
  entries,
  isLoading,
  now,
  teamOnly,
  onTeamOnlyChange,
  canPunch,
  punchingOut,
  onPunchOut,
  onPick,
  selectedWorkerId,
}: ClockBoardProps) {
  const t = useT();

  const ranked = useMemo(() => rankRunning(entries ?? [], now), [entries, now]);
  const overlong = ranked.filter((row) => isOverlong(row.runningMinutes)).length;
  const runningMinutes = ranked.reduce((sum, row) => sum + row.runningMinutes, 0);

  return (
    <section
      aria-labelledby="clock-board-heading"
      className="bg-card overflow-hidden rounded-lg border"
    >
      <header className="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex flex-wrap items-center gap-2">
          <TimerIcon className="text-muted-foreground size-3.5" aria-hidden />
          <h3 id="clock-board-heading" className="text-sm font-medium">
            {t("On the clock now")}
          </h3>
          {ranked.length > 0 ? (
            <Badge variant="secondary" className="text-2xs h-4 px-1 tabular-nums">
              {ranked.length}
            </Badge>
          ) : null}
          {ranked.length > 0 ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0} running between them", formatHours(runningMinutes))}
            </span>
          ) : null}
          {overlong > 0 ? (
            <span className="text-warning-foreground flex items-center gap-1 text-xs">
              <AlertTriangleIcon className="size-3" aria-hidden />
              {t("{0} past 12h", overlong)}
            </span>
          ) : null}
        </div>
        <Button
          size="xs"
          variant={teamOnly ? "default" : "outline"}
          aria-pressed={teamOnly}
          onClick={() => onTeamOnlyChange(!teamOnly)}
        >
          <UsersIcon className="size-3" />
          {t("My team")}
        </Button>
      </header>

      {isLoading && !entries ? (
        <div className="flex flex-col gap-2 p-3">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-2/3" />
        </div>
      ) : ranked.length === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-sm">
          {t("Nobody is punched in{0} right now.", teamOnly ? ` ${t("on your team")}` : "")}
        </p>
      ) : (
        <BoardList
          ranked={ranked}
          canPunch={canPunch}
          punchingOut={punchingOut}
          onPunchOut={onPunchOut}
          onPick={onPick}
          selectedWorkerId={selectedWorkerId}
        />
      )}
    </section>
  );
}

type BoardListProps = Pick<
  ClockBoardProps,
  "canPunch" | "punchingOut" | "onPunchOut" | "onPick" | "selectedWorkerId"
> & {
  ranked: ReturnType<typeof rankRunning<OpenTimeEntryRow>>;
};

function BoardList({
  ranked,
  canPunch,
  punchingOut,
  onPunchOut,
  onPick,
  selectedWorkerId,
}: BoardListProps) {
  const t = useT();

  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);

  // The entrance belongs to the first paint. The board refetches every minute
  // and a row that is still there must not re-announce itself.
  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <ul className="divide-y">
      {ranked.map(({ entry, runningMinutes }, index) => {
        const name = entryWorkerName(entry);
        const selected = entry.workerId === selectedWorkerId;
        const fleet = entry.worker?.fleetCode ?? null;
        const overlong = isOverlong(runningMinutes);
        const share = Math.min(1, runningMinutes / METER_MINUTES);
        return (
          <m.li
            key={entry.id}
            initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25, delay: Math.min(index, STAGGER_LIMIT) * 0.03 }}
            className={cn(
              "grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 transition-colors",
              selected && "bg-accent/60",
            )}
          >
            <button
              type="button"
              onClick={() => onPick(entry.workerId)}
              aria-pressed={selected}
              aria-label={`Open the clock for ${name}`}
              className="focus-visible:ring-ring/60 flex min-w-0 items-center gap-2.5 rounded-md text-left outline-none focus-visible:ring-2"
            >
              <Avatar size="sm">
                {entry.worker?.profilePicUrl ? (
                  <AvatarImage src={entry.worker.profilePicUrl} alt="" />
                ) : null}
                <AvatarFallback className="text-2xs font-medium">
                  {entry.worker ? initials(entry.worker.firstName, entry.worker.lastName) : "?"}
                </AvatarFallback>
                <AvatarBadge aria-hidden className={overlong ? "bg-warning" : "bg-success"} />
              </Avatar>
              <span className="flex min-w-0 flex-1 flex-col gap-1 leading-tight">
                <span className="flex min-w-0 items-center gap-1.5">
                  <span className="truncate text-sm font-medium">{name}</span>
                  {fleet ? (
                    <span className="text-muted-foreground text-2xs flex items-center gap-1">
                      <span
                        aria-hidden
                        className="size-1.5 rounded-full"
                        style={{ backgroundColor: fleet.color }}
                      />
                      {fleet.code}
                    </span>
                  ) : null}
                  <span className="text-muted-foreground text-xs tabular-nums">
                    {t(
                      "since {0}{1}",
                      formatPunchTime(entry.clockedInAt),
                      entry.source !== "Clock" ? ` · ${entry.source}` : "",
                    )}
                  </span>
                </span>
                <span className="flex items-center gap-2">
                  <span
                    role="meter"
                    aria-label={`${name} shift meter`}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-valuenow={Math.round(share * 100)}
                    aria-valuetext={`${formatHours(runningMinutes)} of a 12 hour day`}
                    className="bg-muted relative h-1 w-40 max-w-full overflow-hidden rounded-full"
                  >
                    <span
                      className={cn(
                        "absolute inset-y-0 left-0 rounded-full transition-[width] duration-700 ease-out motion-reduce:transition-none",
                        overlong ? "bg-warning" : "bg-brand/70",
                      )}
                      style={{ width: `${share * 100}%` }}
                    />
                  </span>
                  {entry.note ? (
                    <span className="text-muted-foreground truncate text-2xs">{entry.note}</span>
                  ) : null}
                </span>
              </span>
            </button>
            <span className="flex items-center gap-3">
              <span
                className={cn(
                  "font-mono text-sm font-semibold tabular-nums",
                  overlong && "text-warning-foreground",
                )}
                aria-label={`${name} running time`}
              >
                {formatHours(runningMinutes)}
              </span>
              {canPunch ? (
                <Button
                  size="xs"
                  variant="outline"
                  isLoading={punchingOut === entry.workerId}
                  disabled={punchingOut !== null && punchingOut !== entry.workerId}
                  onClick={() => onPunchOut(entry.workerId)}
                  aria-label={`Clock out ${name}`}
                >
                  <SquareIcon className="size-3" />
                  {t("Clock out")}
                </Button>
              ) : null}
            </span>
          </m.li>
        );
      })}
    </ul>
  );
}
