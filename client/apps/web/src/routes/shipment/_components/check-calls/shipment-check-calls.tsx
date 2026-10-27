import { useCheckCallLabels } from "@/hooks/use-check-call-labels";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import {
  checkCallPlace,
  defaultCheckCallMoveId,
  groupCheckCallsByDay,
  loggableMoves,
  relativeDay,
  sortCheckCalls,
  stopTimesFreeze,
  type StopTimesFreeze,
} from "@/lib/check-calls";
import type { CheckCallRow } from "@/lib/graphql/check-calls";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { stopRouteLabel } from "@/lib/stop-labels";
import type { CheckCallKind } from "@trenova/graphql/generated/graphql";
import {
  ArrowUpRightIcon,
  ClockAlertIcon,
  InfoCircleIcon,
  MarkerPin02Icon,
  NavigationPointer01Icon,
  PhoneCall01Icon,
  PlusIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatSecondsAgo,
  formatUnixDateMedium,
  formatUnixDateTimeShort,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { CheckCallsSkeleton } from "./check-calls-skeleton";
import { LogCheckCallDialog, type CheckCallStopOption } from "./log-check-call-dialog";

const ALL_MOVES = "all";
const DAY_SECONDS = 86_400;

const KIND_ICONS: Record<CheckCallKind, IconComponent> = {
  Update: NavigationPointer01Icon,
  Arrival: MarkerPin02Icon,
  Departure: ArrowUpRightIcon,
};

type CallContext = {
  moveLabel?: string;
  stopLabel?: string;
};

export default function ShipmentCheckCalls({ shipment }: { shipment?: Shipment | null }) {
  const t = useT();
  const shipmentId = shipment?.id ?? "";
  const moves = shipment?.moves ?? [];
  const [dialogOpen, setDialogOpen] = useState(false);
  const [moveFilter, setMoveFilter] = useState<string>(ALL_MOVES);

  const { data, isPending, isError, error } = useQuery({
    ...shipmentQueries.checkCalls(shipmentId),
    enabled: shipmentId !== "",
  });

  if (!shipmentId) {
    return null;
  }

  const moveLabels = new Map<string, string>();
  const stopLabels = new Map<string, string>();
  const stops: CheckCallStopOption[] = [];
  moves.forEach((move, moveIndex) => {
    const moveId = move.id;
    if (!moveId) {
      return;
    }
    moveLabels.set(moveId, t("Move {0}", moveIndex + 1));
    (move.stops ?? []).forEach((stop, stopIndex) => {
      if (!stop.id) {
        return;
      }
      const label = stopRouteLabel(stop, stopIndex);
      stopLabels.set(stop.id, label);
      stops.push({ id: stop.id, moveId, label });
    });
  });
  const loggable = loggableMoves(moves, (_move, index) => t("Move {0}", index + 1));
  const canLog = loggable.length > 0;
  const frozenNote = freezeNote(t, shipment ? stopTimesFreeze(shipment) : null);
  const showMoves = moveLabels.size > 1;

  const calls = sortCheckCalls(data ?? []).filter(
    (call) => moveFilter === ALL_MOVES || call.shipmentMoveId === moveFilter,
  );
  const context = (call: CheckCallRow): CallContext => ({
    moveLabel: showMoves ? moveLabels.get(call.shipmentMoveId) : undefined,
    stopLabel: call.stopId ? stopLabels.get(call.stopId) : undefined,
  });
  const openDialog = () => setDialogOpen(true);

  return (
    <div className="flex flex-col gap-6">
      {isPending ? (
        <CheckCallsSkeleton />
      ) : isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>
            {error.message || t("Check calls could not be loaded.")}
          </AlertDescription>
        </Alert>
      ) : (
        <>
          {canLog && frozenNote ? (
            <Alert size="sm" variant="info">
              <InfoCircleIcon />
              <AlertDescription>{frozenNote}</AlertDescription>
            </Alert>
          ) : null}
          {showMoves ? (
            <SegmentedControl
              aria-label={t("Show check calls for")}
              value={moveFilter}
              onValueChange={setMoveFilter}
              items={[
                { value: ALL_MOVES, label: t("All moves") },
                ...[...moveLabels].map(([value, label]) => ({ value, label })),
              ]}
            />
          ) : null}
          {calls.length === 0 ? (
            <EmptyCheckCalls
              filtered={moveFilter !== ALL_MOVES}
              canLog={canLog}
              stopTimesApply={!frozenNote}
              onLog={openDialog}
            />
          ) : (
            <>
              <LatestReport
                call={calls[0]}
                count={calls.length}
                context={context(calls[0])}
                onLog={canLog ? openDialog : undefined}
              />
              <CheckCallTimeline calls={calls} latestId={calls[0].id} context={context} />
            </>
          )}
        </>
      )}
      {canLog ? (
        <LogCheckCallDialog
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          shipmentId={shipmentId}
          moves={loggable}
          stops={stops}
          stopTimesNote={frozenNote}
          defaultMoveId={
            moveFilter !== ALL_MOVES && loggable.some((move) => move.id === moveFilter)
              ? moveFilter
              : defaultCheckCallMoveId(loggable, moves)
          }
        />
      ) : null}
    </div>
  );
}

// What a dispatcher needs to know before logging an arrival or departure the
// server will keep off the stop's times.
function freezeNote(t: ReturnType<typeof useT>, freeze: StopTimesFreeze | null): string | null {
  switch (freeze) {
    case "invoiced":
      return t(
        "Arrivals and departures logged now won't change stop times — this load has been invoiced.",
      );
    case "billing-queue":
      return t(
        "Arrivals and departures logged now won't change stop times — this load is in the billing queue.",
      );
    default:
      return null;
  }
}

function LogButton({ onLog }: { onLog: () => void }) {
  const t = useT();

  return (
    <Button type="button" size="sm" onClick={onLog}>
      <PlusIcon className="size-3.5" />
      {t("Log check call")}
    </Button>
  );
}

function reportedAgo(reportedAt: number, now: number): string {
  const age = Math.max(now - reportedAt, 0);
  return age < DAY_SECONDS ? formatSecondsAgo(age) : formatUnixDateTimeShort(reportedAt);
}

function LatestReport({
  call,
  count,
  context,
  onLog,
}: {
  call: CheckCallRow;
  count: number;
  context: CallContext;
  onLog?: () => void;
}) {
  const t = useT();
  const labels = useCheckCallLabels();
  const now = useNowSeconds();
  const kind = labels.kind(call.kind);
  const where = checkCallPlace(call) || context.stopLabel || "";

  return (
    <section
      aria-label={t("Last reported")}
      className="flex flex-col gap-4 rounded-lg border bg-card p-4"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <span className="text-xs text-foreground-muted">{t("Last reported")}</span>
          <h3 className="truncate text-xl font-semibold">{where || kind}</h3>
          <p className="text-xs text-foreground-muted">
            {[where ? kind : null, reportedAgo(call.reportedAt, now), labels.source(call.source)]
              .filter(Boolean)
              .join(" · ")}
          </p>
        </div>
        {onLog ? <LogButton onLog={onLog} /> : null}
      </div>
      {call.delayed ? (
        <Alert size="sm" variant="warning">
          <ClockAlertIcon />
          <AlertTitle>{t("Running late")}</AlertTitle>
          {call.delayReason ? <AlertDescription>{call.delayReason}</AlertDescription> : null}
        </Alert>
      ) : null}
      <dl className="grid grid-cols-2 gap-3 border-t border-border-subtle pt-3">
        <div className="flex flex-col gap-0.5">
          <dt className="text-2xs text-foreground-muted">{t("Expected at next stop")}</dt>
          <dd className="font-mono text-sm tabular-nums">
            {call.estimatedArrival
              ? formatUnixDateTimeShort(call.estimatedArrival)
              : t("Not given")}
          </dd>
        </div>
        <div className="flex flex-col gap-0.5">
          <dt className="text-2xs text-foreground-muted">{t("Logged")}</dt>
          <dd className="text-sm">
            {t("{0, plural, one {# check call} other {# check calls}}", count)}
          </dd>
        </div>
      </dl>
    </section>
  );
}

function dayLabel(day: number, now: Date, t: ReturnType<typeof useT>): string {
  switch (relativeDay(day, now)) {
    case "today":
      return t("Today");
    case "yesterday":
      return t("Yesterday");
    default:
      return formatUnixDateMedium(day);
  }
}

function CheckCallTimeline({
  calls,
  latestId,
  context,
}: {
  calls: CheckCallRow[];
  latestId: string;
  context: (call: CheckCallRow) => CallContext;
}) {
  const t = useT();
  const now = useNowSeconds();
  const days = groupCheckCallsByDay(calls);

  return (
    <ol aria-label={t("Check call history")} className="flex flex-col gap-5">
      {days.map(({ day, calls: dayCalls }) => (
        <li key={day} className="flex flex-col gap-3">
          <div className="flex items-center gap-3">
            <h4 className="text-xs font-medium text-foreground-muted">
              {dayLabel(day, new Date(now * 1000), t)}
            </h4>
            <span aria-hidden className="h-px flex-1 bg-border-subtle" />
          </div>
          <ol className="flex flex-col">
            {dayCalls.map((call) => (
              <CheckCallEntry
                key={call.id}
                call={call}
                latest={call.id === latestId}
                context={context(call)}
              />
            ))}
          </ol>
        </li>
      ))}
    </ol>
  );
}

function CheckCallEntry({
  call,
  latest,
  context,
}: {
  call: CheckCallRow;
  latest: boolean;
  context: CallContext;
}) {
  const t = useT();
  const labels = useCheckCallLabels();
  const Icon = KIND_ICONS[call.kind];
  const place = checkCallPlace(call);
  const meta = [
    context.stopLabel,
    context.moveLabel,
    labels.source(call.source),
    call.contactName ? t("Spoke to {0}", call.contactName) : null,
  ].filter(Boolean);

  return (
    <li className="group/entry grid grid-cols-[3.5rem_1.5rem_minmax(0,1fr)] gap-x-3">
      <time
        dateTime={new Date(call.reportedAt * 1000).toISOString()}
        className="pt-1 text-right font-mono text-2xs text-foreground-muted tabular-nums"
      >
        {formatUnixTime(call.reportedAt)}
      </time>
      <div className="relative flex justify-center">
        <span
          aria-hidden
          className="absolute top-7 -bottom-1 w-px bg-border-subtle group-last/entry:hidden"
        />
        <span
          data-latest={latest || undefined}
          className={cn(
            "relative flex size-6 items-center justify-center rounded-full border",
            latest
              ? "border-brand bg-brand text-foreground-on-solid"
              : call.delayed
                ? "border-warning-border bg-warning-subtle text-warning-subtle-foreground"
                : "bg-card text-foreground-muted",
          )}
        >
          <Icon className="size-3" />
        </span>
      </div>
      <div className="flex min-w-0 flex-col gap-1 pb-5 group-last/entry:pb-0">
        <div className="flex items-start justify-between gap-2 pt-0.5">
          <p className="min-w-0 text-sm">
            <span className="font-medium">{labels.kind(call.kind)}</span>
            {place ? <span className="text-foreground-muted"> · {place}</span> : null}
          </p>
          {call.delayed ? (
            <Badge variant="warning" className="shrink-0">
              {t("Running late")}
            </Badge>
          ) : null}
        </div>
        {meta.length > 0 ? (
          <p className="text-xs text-foreground-muted">{meta.join(" · ")}</p>
        ) : null}
        {call.delayed && call.delayReason ? (
          <p className="text-xs text-warning">{call.delayReason}</p>
        ) : null}
        {call.estimatedArrival ? (
          <p className="text-xs text-foreground-muted">
            {t("Expected at next stop {0}", formatUnixDateTimeShort(call.estimatedArrival))}
          </p>
        ) : null}
        {call.note ? (
          <p className="mt-1 rounded-md bg-sunken px-2.5 py-1.5 text-xs whitespace-pre-line">
            {call.note}
          </p>
        ) : null}
      </div>
    </li>
  );
}

function EmptyCheckCalls({
  filtered,
  canLog,
  stopTimesApply,
  onLog,
}: {
  filtered: boolean;
  canLog: boolean;
  stopTimesApply: boolean;
  onLog: () => void;
}) {
  const t = useT();

  return (
    <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed px-6 py-12 text-center">
      <span className="flex size-10 items-center justify-center rounded-full bg-sunken">
        <PhoneCall01Icon className="size-4 text-foreground-muted" />
      </span>
      <div className="flex max-w-xs flex-col gap-1">
        <p className="text-sm font-medium">
          {filtered ? t("No check calls on this move yet") : t("No check calls yet")}
        </p>
        <p className="text-xs text-foreground-muted">
          {!canLog
            ? t("This shipment has no active move to log a check call against.")
            : stopTimesApply
              ? t(
                  "Log where the load is each time the driver or carrier reports in. An arrival or departure also records the stop's actual time.",
                )
              : t("Log where the load is each time the driver or carrier reports in.")}
        </p>
      </div>
      {canLog ? <LogButton onLog={onLog} /> : null}
    </div>
  );
}
