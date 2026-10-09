import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@trenova/shared/components/ui/hover-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan, formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

export type StopStripStop = {
  id: string;
  kind: "pickup" | "delivery";
  state: "done" | "current" | "upcoming" | "canceled";
  late: boolean;
  /** The city the stop is in. */
  place: string;
  /** The facility, when it has a name. */
  facility?: string;
  /** When it is due, in Unix seconds. */
  due: number | null;
  /** When the truck got there, in Unix seconds. */
  arrived: number | null;
};

export type StopProgress = {
  /** Stops that count: every stop that is not canceled. */
  total: number;
  done: number;
  /** The stop the load is headed for, or the first still to come. */
  next: StopStripStop | null;
  /** How much of the leg into `next` has gone by, 0 to 1; 0 before the load moves. */
  legShare: number;
  late: number;
  /** Minutes until `next` is due; negative once it is overdue; null without a time. */
  minutesToNext: number | null;
};

/**
 * Where a load is on its route at `now` (Unix seconds): how many stops are behind
 * it, which comes next and how soon, and how far through the current leg it is,
 * judged by the leg's time from the stop it left to the one it is headed for.
 */
export function stopProgress(stops: readonly StopStripStop[], now: number): StopProgress {
  const counted = stops.filter((stop) => stop.state !== "canceled");
  const done = counted.filter((stop) => stop.state === "done").length;
  const late = counted.filter((stop) => stop.late).length;
  const nextIndex = stops.findIndex((stop) => stop.state === "current");
  const fallbackIndex = stops.findIndex((stop) => stop.state === "upcoming");
  const index = nextIndex >= 0 ? nextIndex : fallbackIndex;
  const next = index >= 0 ? stops[index] : null;

  let legShare = 0;
  if (nextIndex > 0) {
    const from = stops[nextIndex - 1];
    const leftAt = from.arrived ?? from.due;
    const dueAt = stops[nextIndex].due;
    legShare =
      leftAt !== null && dueAt !== null && dueAt > leftAt
        ? Math.min(0.95, Math.max(0.05, (now - leftAt) / (dueAt - leftAt)))
        : 0.5;
  }

  const minutesToNext = next?.due ? Math.round((next.due - now) / 60) : null;
  return { total: counted.length, done, next, legShare, late, minutesToNext };
}

/** Past this many stops the bar is one track rather than a segment per stop. */
const MAX_SEGMENTS = 12;

/** Brings the next stop into view when a long itinerary opens part way down. */
function revealRow(row: HTMLTableRowElement | null) {
  row?.scrollIntoView({ block: "nearest" });
}

function formatMoment(value: number | null): string | null {
  return value ? formatToUserTimezone(value, { showTimeZone: false, showSeconds: false }) : null;
}

function StopStatus({ stop, isNext }: { stop: StopStripStop; isNext: boolean }) {
  const t = useT();
  if (stop.state === "canceled") return <Badge variant="neutral" appearance="outline">{t("Canceled")}</Badge>;
  if (stop.state === "done") {
    return stop.late ? (
      <Badge variant="danger">{t("Late")}</Badge>
    ) : (
      <Badge variant="success">{t("Done")}</Badge>
    );
  }
  if (stop.late) return <Badge variant="danger">{t("Overdue")}</Badge>;
  if (isNext) return <Badge variant="info">{t("Next")}</Badge>;
  return <Badge variant="neutral" appearance="outline">{t("Scheduled")}</Badge>;
}

/**
 * A load's route read as a status line: which stop it is on out of how many, the
 * next city and how soon (or how overdue), over a bar with one segment per stop that
 * fills as each is reached and part-fills through the leg under way. Hovering opens
 * the full itinerary.
 */
export function StopStripCell({ stops, now }: { stops: readonly StopStripStop[]; now: number }) {
  const t = useT();
  if (stops.length === 0) {
    return <span className="text-muted-foreground font-mono text-sm">—</span>;
  }

  const progress = stopProgress(stops, now);
  const finished = progress.total > 0 && progress.done === progress.total;
  const overdue = progress.minutesToNext !== null && progress.minutesToNext < 0;
  const nextLabel = progress.next
    ? progress.next.place ||
      progress.next.facility ||
      (progress.next.kind === "pickup" ? t("Pickup") : t("Delivery"))
    : null;
  const counted = stops.filter((stop) => stop.state !== "canceled");
  const lastPlace = counted.at(-1)?.place || counted.at(-1)?.facility || "";

  return (
    <HoverCard>
      <HoverCardTrigger
        delay={300}
        render={<span className="flex w-full min-w-0 cursor-default flex-col justify-center gap-1.5" />}
      >
        <span className="flex min-w-0 items-center gap-2">
          <Badge
            variant={finished ? "success" : progress.late > 0 ? "danger" : "neutral"}
            className="font-mono tabular-nums"
          >
            {progress.done}/{progress.total}
          </Badge>
          <span className="min-w-0 flex-1 truncate text-sm">
            {finished ? (
              <span className="text-muted-foreground">
                {t("Delivered")} · <span className="text-foreground">{lastPlace}</span>
              </span>
            ) : (
              <>
                <span className="text-muted-foreground">
                  {progress.next?.kind === "pickup" ? t("Pickup") : t("Next")}{" "}
                </span>
                <span className="font-medium">{nextLabel}</span>
              </>
            )}
          </span>
          {!finished && progress.minutesToNext !== null ? (
            <span
              className={cn(
                "shrink-0 font-mono text-xs tabular-nums",
                overdue ? "text-danger" : "text-muted-foreground",
              )}
            >
              {overdue
                ? t("{0} late", formatMinutesSpan(-progress.minutesToNext))
                : t("in {0}", formatMinutesSpan(progress.minutesToNext))}
            </span>
          ) : null}
        </span>
        {counted.length <= MAX_SEGMENTS ? (
          <span className="flex h-1 w-full gap-0.5 in-data-[density=compact]:hidden" aria-hidden>
            {counted.map((stop) => {
              const isNext = stop.id === progress.next?.id && stop.state === "current";
              const fill = stop.state === "done" ? 1 : isNext ? progress.legShare : 0;
              return (
                <span key={stop.id} className="bg-muted relative flex-1 overflow-hidden rounded-full">
                  <span
                    className={cn(
                      "absolute inset-y-0 left-0 rounded-full",
                      stop.late ? "bg-danger" : "bg-foreground",
                    )}
                    style={{ width: `${fill * 100}%` }}
                  />
                </span>
              );
            })}
          </span>
        ) : (
          <span
            className="bg-muted relative h-1 w-full overflow-hidden rounded-full in-data-[density=compact]:hidden"
            aria-hidden
          >
            <span
              className={cn(
                "absolute inset-y-0 left-0 rounded-full",
                progress.next?.late ? "bg-danger" : "bg-foreground",
              )}
              style={{
                width: `${progress.total > 0 ? ((progress.done + progress.legShare) / progress.total) * 100 : 0}%`,
              }}
            />
          </span>
        )}
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-96 p-0">
        <div className="border-border flex items-baseline justify-between gap-3 border-b px-3 py-2">
          <span className="text-sm font-medium">{t("Route")}</span>
          <span className="text-muted-foreground text-xs tabular-nums">
            {t("{0} of {1} stops done", progress.done, progress.total)}
            {progress.late > 0 ? ` · ${t("{0} late", progress.late)}` : ""}
          </span>
        </div>
        <ScrollArea viewportClassName="max-h-80" maskVariant="popover">
        <table className="w-full text-xs">
          <tbody>
            {stops.map((stop, index) => (
              <tr
                key={stop.id}
                ref={stop.id === progress.next?.id ? revealRow : undefined}
                className={cn(
                  "border-border border-b last:border-b-0",
                  stop.state === "canceled" && "opacity-60",
                )}
              >
                <td className="text-muted-foreground w-6 py-2 pl-3 align-top font-mono tabular-nums">
                  {index + 1}
                </td>
                <td className="py-2 pr-2 align-top">
                  <div className="flex flex-col">
                    <span className="font-medium">
                      {stop.place || stop.facility || "—"}
                    </span>
                    <span className="text-muted-foreground">
                      {stop.kind === "pickup" ? t("Pickup") : t("Delivery")}
                      {stop.facility && stop.place ? ` · ${stop.facility}` : ""}
                    </span>
                  </div>
                </td>
                <td className="text-muted-foreground py-2 pr-2 text-right align-top tabular-nums">
                  {stop.arrived ? (
                    <div className="flex flex-col">
                      <span className="text-foreground">{formatMoment(stop.arrived)}</span>
                      <span>{t("Arrived")}</span>
                    </div>
                  ) : (
                    <div className="flex flex-col">
                      <span className="text-foreground">{formatMoment(stop.due) ?? "—"}</span>
                      <span>{t("Due")}</span>
                    </div>
                  )}
                </td>
                <td className="py-2 pr-3 text-right align-top">
                  <StopStatus stop={stop} isNext={stop.id === progress.next?.id} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        </ScrollArea>
      </HoverCardContent>
    </HoverCard>
  );
}
