import { HoverGlideGroup } from "@/components/hover-glide";
import type { DispatchBoardDriver, DispatchDriverMoveMatch } from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { formatTimeAgo } from "@/lib/time-utils";
import type { TimeRange } from "@/lib/timeline/time-scale";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixTime } from "@trenova/shared/lib/date";
import { cn, formatCompactCurrency, formatCentsWhole } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { availabilityMeta, formatMiles, pickupState } from "./dispatch-vocabulary";
import { hoursLeft, whenFree } from "./driver-queue";
import { FindingList } from "./finding-list";
import { formatWhen } from "./format-when";
import { InspectorScroll } from "./inspector-scroll";

const MATCH_STALE_MS = 30_000;
const MATCH_LIMIT = 20;

/** A titled card in the inspector, as "Drivers for this move" is for a move. */
function Panel({
  title,
  count,
  children,
}: {
  title: string;
  count?: number | null;
  children: ReactNode;
}) {
  return (
    <section className="bg-card border-border overflow-hidden rounded-md border" aria-label={title}>
      <header className="border-border flex h-8.5 items-center gap-1.5 border-b px-3 text-xs font-semibold">
        {title}
        {count != null ? (
          <span className="text-muted-foreground ml-auto text-2xs font-normal tabular-nums">
            {count}
          </span>
        ) : null}
      </header>
      {children}
    </section>
  );
}

function MatchRow({
  match,
  nowSeconds,
  onSelectMove,
}: {
  match: DispatchDriverMoveMatch;
  nowSeconds: number;
  onSelectMove: (moveId: string) => void;
}) {
  const t = useT();
  const { move, candidate } = match;
  const pickup = pickupState(
    move,
    nowSeconds,
    (seconds) => formatWhen(seconds, nowSeconds),
    formatUnixTime,
  );

  return (
    <button
      type="button"
      data-glide-target
      onClick={() => onSelectMove(move.moveId)}
      className="ui-focus-ring relative mx-1 flex w-[calc(100%-0.5rem)] flex-col gap-0.5 rounded-md p-2 text-left text-xs"
    >
      <span className="flex min-w-0 items-center gap-2">
        <span className="shrink-0 font-mono font-medium">{move.proNumber}</span>
        <span className="truncate">
          {t(
            "{0}, {1} → {2}, {3}",
            move.originCity,
            move.originState,
            move.destinationCity,
            move.destinationState,
          )}
        </span>
        <span className="ml-auto shrink-0 font-mono tabular-nums">
          {formatCentsWhole(candidate.costCents)}
        </span>
      </span>
      <span className="text-muted-foreground truncate text-2xs">
        {candidate.deadheadMiles != null
          ? t("{0} · {1} empty", pickup.headline, formatMiles(candidate.deadheadMiles))
          : pickup.headline}
      </span>
    </button>
  );
}

/**
 * A driver, read the way a dispatcher sizes them up: when they are free, where, on what
 * hours, what they already hold, and the open moves in the window that suit them best.
 */
export function DriverInspector({
  driver,
  range,
  onSelectMove,
  onClose,
}: {
  driver: DispatchBoardDriver;
  range: TimeRange;
  onSelectMove: (moveId: string) => void;
  onClose: () => void;
}) {
  const t = useT();
  const nowSeconds = useNow("minute");

  // The board's window, so "best fit" answers for the moves on screen.
  const matchesQuery = useQuery({
    ...dispatchConsoleQueries.driverMoves({
      workerId: driver.workerId,
      limit: MATCH_LIMIT,
      windowStart: range.start,
      windowEnd: range.end,
    }),
    staleTime: MATCH_STALE_MS,
  });
  const matches = matchesQuery.data ?? [];

  const availability = availabilityMeta(driver.availability);
  const hours = hoursLeft(driver, t);
  const nextTimeOff = driver.timeOff[0];
  const planned = driver.plannedMove ?? null;
  const where =
    driver.formattedLocation ||
    (driver.city ? t("{0}, {1}", driver.city, driver.stateAbbreviation) : t("Location unknown"));

  return (
    <div className="animate-panel-in flex min-h-0 flex-1 flex-col">
      <InspectorScroll className="flex flex-col gap-3 p-3.5">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <span className="font-mono text-xs font-semibold">
              {driver.tractorCode || t("No tractor")}
            </span>
            <span className="text-muted-foreground truncate text-2xs">{driver.fleetCodeName}</span>
            <Button
              variant="ghost"
              size="icon-xs"
              className="ml-auto"
              aria-label={t("Close inspector")}
              title={t("Close (Esc)")}
              onClick={onClose}
            >
              <XCloseIcon className="size-3.5" />
            </Button>
          </div>
          <h3 className="text-base leading-6 font-semibold">
            {t("{0} {1}", driver.firstName, driver.lastName)}
          </h3>
          <div className={cn("text-xs font-medium", availability.labelClass)}>
            {whenFree(driver, t)}
          </div>
          <DescriptionList layout="inline" className="mt-1.5 text-xs">
            <DescriptionItem label={t("Location")}>{where}</DescriptionItem>
            <DescriptionItem label={t("Hours left")}>
              <span className={cn(!hours.muted && "font-mono tabular-nums")}>{hours.line}</span>
              <span className="text-muted-foreground"> · {hours.detail}</span>
            </DescriptionItem>
            <DescriptionItem label={t("Holding")}>
              {driver.openAssignments > 0
                ? [
                    t("{0, plural, one {# load} other {# loads}}", driver.openAssignments),
                    driver.committedMiles > 0 ? formatMiles(driver.committedMiles) : null,
                    driver.committedRevenue > 0
                      ? formatCompactCurrency(driver.committedRevenue)
                      : null,
                  ]
                    .filter(Boolean)
                    .join(" · ")
                : t("No work yet")}
            </DescriptionItem>
            {planned ? (
              <DescriptionItem label={t("Plan")}>
                <span className="inline-flex flex-wrap items-center gap-x-1.5">
                  <AssistMark className="text-muted-foreground size-3 shrink-0" aria-hidden />
                  <Button
                    variant="link"
                    size="xxs"
                    className="px-0"
                    title={t(
                      "Priced {0}",
                      formatTimeAgo(planned.generatedAt * 1000, nowSeconds * 1000),
                    )}
                    onClick={() => onSelectMove(planned.moveId)}
                  >
                    {t(
                      "Planned on {0} for {1}",
                      planned.proNumber,
                      formatCentsWhole(planned.costCents),
                    )}
                  </Button>
                  {planned.valueCents > 0 ? (
                    <span className="text-muted-foreground tabular-nums">
                      {t("Worth {0} to the plan", formatCentsWhole(planned.valueCents))}
                    </span>
                  ) : null}
                </span>
              </DescriptionItem>
            ) : null}
            {driver.lastHomeAt ? (
              <DescriptionItem label={t("Last home")}>
                {formatUnixDateMedium(driver.lastHomeAt)}
              </DescriptionItem>
            ) : null}
            {nextTimeOff ? (
              <DescriptionItem label={t("Time off")}>
                {t(
                  "{0} – {1}",
                  formatUnixDateMedium(nextTimeOff.startDate),
                  formatUnixDateMedium(nextTimeOff.endDate),
                )}
              </DescriptionItem>
            ) : null}
          </DescriptionList>
        </div>

        {driver.findings.length > 0 ? (
          <Panel title={t("Needs attention")} count={driver.findings.length}>
            <FindingList findings={driver.findings} className="p-3" />
          </Panel>
        ) : null}

        {driver.commitments.length > 0 ? (
          <Panel title={t("Committed")} count={driver.commitments.length}>
            <HoverGlideGroup className="py-1">
              {driver.commitments.map((commitment) => (
                <button
                  key={commitment.moveId}
                  type="button"
                  data-glide-target
                  onClick={() => onSelectMove(commitment.moveId)}
                  className="ui-focus-ring relative mx-1 flex w-[calc(100%-0.5rem)] flex-col gap-0.5 rounded-md p-2 text-left text-xs"
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="shrink-0 font-mono font-medium">{commitment.proNumber}</span>
                    <span className="truncate">
                      {t("to {0}, {1}", commitment.destinationCity, commitment.destinationState)}
                    </span>
                  </span>
                  <span className="text-muted-foreground truncate text-2xs">
                    {t(
                      "{0} – {1}",
                      formatWhen(commitment.windowStart, nowSeconds),
                      formatWhen(commitment.windowEnd, nowSeconds),
                    )}
                  </span>
                </button>
              ))}
            </HoverGlideGroup>
          </Panel>
        ) : null}

        <Panel
          title={t("Best fit in this window")}
          count={matchesQuery.isPending || matchesQuery.isError ? null : matches.length}
        >
          {matchesQuery.isPending ? (
            <div
              className="flex flex-col gap-1 p-1"
              aria-busy="true"
              aria-label={t("Finding work")}
            >
              {Array.from({ length: 4 }, (_, index) => (
                <div key={index} className="ui-shimmer h-11 rounded-md" />
              ))}
            </div>
          ) : matchesQuery.isError ? (
            <div className="p-2">
              <ErrorState
                error={matchesQuery.error}
                layout="compact"
                title={t("Couldn't find work for this driver")}
                description={t("The matches didn't load. Open moves may still suit this driver.")}
                onRetry={() => void matchesQuery.refetch()}
              />
            </div>
          ) : matches.length === 0 ? (
            <p className="text-muted-foreground px-3 py-4 text-xs">
              {t("No open move in this window suits this driver.")}
            </p>
          ) : (
            <HoverGlideGroup className="py-1">
              {matches.map((match) => (
                <MatchRow
                  key={match.move.moveId}
                  match={match}
                  nowSeconds={nowSeconds}
                  onSelectMove={onSelectMove}
                />
              ))}
            </HoverGlideGroup>
          )}
        </Panel>
      </InspectorScroll>
      <footer className="border-border text-muted-foreground border-t px-3.5 py-2 text-2xs">
        {t("Select a move to open it and assign this driver from its pre-flight")}
      </footer>
    </div>
  );
}
