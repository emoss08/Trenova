import { useCheckCallLabels } from "@/hooks/use-check-call-labels";
import { checkCallPlace, loggableMoves, sortCheckCalls } from "@/lib/check-calls";
import type { CheckCallRow } from "@/lib/graphql/check-calls";
import { shipment } from "@/lib/queries/shipment";
import { stopRouteLabel } from "@/lib/stop-labels";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormSection } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { LogCheckCallDialog, type CheckCallStopOption } from "./log-check-call-dialog";

export function CheckCalls() {
  const t = useT();
  const { control } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" });
  const moves = useWatch({ control, name: "moves" });
  const [dialogOpen, setDialogOpen] = useState(false);

  const { data, isPending, isError, error } = useQuery({
    ...shipment.checkCalls(shipmentId ?? ""),
    enabled: !!shipmentId,
  });

  const options = useMemo(() => {
    const allMoves = moves ?? [];
    const loggable = loggableMoves(allMoves, (_move, index) => t("Move {0}", index + 1));
    const stops: CheckCallStopOption[] = [];
    const stopLabels = new Map<string, string>();
    allMoves.forEach((move) => {
      (move.stops ?? []).forEach((stop: Stop, stopIndex) => {
        if (!move.id || !stop.id) {
          return;
        }
        const label = stopRouteLabel(stop, stopIndex);
        stopLabels.set(stop.id, label);
        stops.push({ id: stop.id, moveId: move.id, label });
      });
    });
    const moveLabels = new Map(loggable.map((move) => [move.id, move.label]));
    return { loggable, stops, stopLabels, moveLabels };
  }, [moves, t]);

  const calls = useMemo(() => sortCheckCalls(data ?? []), [data]);
  const showMove = options.loggable.length > 1;
  const defaultMoveId =
    options.loggable.find((move) => {
      const status = (moves ?? []).find((candidate) => candidate.id === move.id)?.status;
      return status === "InTransit";
    })?.id ??
    options.loggable[0]?.id ??
    "";

  if (!shipmentId) {
    return null;
  }

  return (
    <FormSection
      title={t("Check calls")}
      description={t(
        "Where the load was reported to be, by dispatchers, the carrier's tracking link and agents",
      )}
      action={
        options.loggable.length > 0 ? (
          <Button type="button" variant="outline" size="xxs" onClick={() => setDialogOpen(true)}>
            <PlusIcon className="size-3" />
            {t("Log check call")}
          </Button>
        ) : null
      }
    >
      {isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>
            {error.message || t("Check calls could not be loaded.")}
          </AlertDescription>
        </Alert>
      ) : calls.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("No check calls have been logged yet.")}</p>
      ) : (
        <ul className="divide-y divide-border rounded-md border border-border">
          {calls.map((call) => (
            <CheckCallItem
              key={call.id}
              call={call}
              moveLabel={showMove ? options.moveLabels.get(call.shipmentMoveId) : undefined}
              stopLabel={call.stopId ? options.stopLabels.get(call.stopId) : undefined}
            />
          ))}
        </ul>
      )}
      {options.loggable.length > 0 ? (
        <LogCheckCallDialog
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          shipmentId={shipmentId}
          moves={options.loggable}
          stops={options.stops}
          defaultMoveId={defaultMoveId}
        />
      ) : null}
    </FormSection>
  );
}

function CheckCallItem({
  call,
  moveLabel,
  stopLabel,
}: {
  call: CheckCallRow;
  moveLabel?: string;
  stopLabel?: string;
}) {
  const t = useT();
  const labels = useCheckCallLabels();
  const place = checkCallPlace(call);

  return (
    <li className="flex flex-col gap-1 px-3 py-2">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
        <span className="font-medium">{labels.kind(call.kind)}</span>
        <span>{formatUnixDateTime(call.reportedAt)}</span>
        {place ? <span>{place}</span> : null}
        {stopLabel ? <span className="text-muted-foreground">{stopLabel}</span> : null}
        {moveLabel ? <span className="text-muted-foreground">{moveLabel}</span> : null}
        <span className="ml-auto flex items-center gap-2">
          {call.delayed ? <Badge variant="warning">{t("Running late")}</Badge> : null}
          <span className="text-xs text-muted-foreground">{labels.source(call.source)}</span>
        </span>
      </div>
      {call.delayed && call.delayReason ? (
        <p className="text-xs text-muted-foreground">{call.delayReason}</p>
      ) : null}
      {call.estimatedArrival || call.contactName ? (
        <p className="text-xs text-muted-foreground">
          {[
            call.estimatedArrival
              ? t("Expected at next stop {0}", formatUnixDateTime(call.estimatedArrival))
              : null,
            call.contactName ? t("Spoke to {0}", call.contactName) : null,
          ]
            .filter(Boolean)
            .join(" · ")}
        </p>
      ) : null}
      {call.note ? <p className="text-xs whitespace-pre-line">{call.note}</p> : null}
    </li>
  );
}
