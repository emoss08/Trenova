import { useUserTimezone } from "@/hooks/use-user-timezone";
import { queries } from "@/lib/queries";
import { resolveCoverage } from "@/lib/shipment-board/coverage";
import { quickFilterLabel } from "@/lib/shipment-board/quick-filters";
import { getDestinationLocation, getDestinationStop, getOriginLocation } from "@/lib/shipment-utils";
import type { ShipmentQuickFilter } from "@trenova/graphql/generated/graphql";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useEtaDelta } from "../cells/eta-cell";
import { useOpenShipmentRecord } from "../use-open-shipment-record";

export const BRIEFING_PREVIEW_LOADS = 5;
const PREVIEW_STALE_MS = 30_000;

function coverageName(shipment: Shipment, t: ReturnType<typeof useT>): string {
  const coverage = resolveCoverage(shipment);
  switch (coverage.kind) {
    case "driver":
    case "carrier":
      return coverage.name;
    case "tendered":
      return t("Tendered · awaiting");
    case "uncovered":
      return t("Needs coverage");
  }
}

function PreviewLoad({ shipment, onOpen }: { shipment: Shipment; onOpen: () => void }) {
  const t = useT();
  const origin = getOriginLocation(shipment);
  const destination = getDestinationLocation(shipment);
  const stop = getDestinationStop(shipment);
  const arrival =
    shipment.eta?.estimatedArrival ?? stop?.scheduledWindowEnd ?? stop?.scheduledWindowStart ?? null;
  const delta = useEtaDelta(shipment);
  const uncovered = resolveCoverage(shipment).kind === "uncovered";

  return (
    <li>
      <button
        type="button"
        onClick={onOpen}
        className="ui-focus-ring hover:bg-muted flex w-full min-w-0 flex-col gap-0.5 rounded-sm px-2.5 py-1.5 text-left transition-colors"
      >
        <span className="flex min-w-0 items-center gap-1.5 text-sm">
          <span className="font-mono text-xs tabular-nums">{shipment.proNumber || "—"}</span>
          <span className="flex min-w-0 items-center gap-1 font-medium">
            <span className="truncate">{origin?.city ?? origin?.code ?? "—"}</span>
            <ArrowRightIcon className="text-muted-foreground size-3 shrink-0" aria-hidden />
            <span className="truncate">{destination?.city ?? destination?.code ?? "—"}</span>
          </span>
        </span>
        <span className="text-muted-foreground flex min-w-0 items-center gap-1.5 text-xs">
          <span className="truncate">{shipment.customer?.name ?? t("No customer")}</span>
          <span aria-hidden>·</span>
          <span className={cn("truncate", uncovered && "text-warning")}>
            {coverageName(shipment, t)}
          </span>
          {arrival ? (
            <>
              <span aria-hidden>·</span>
              <span className="shrink-0 font-mono tabular-nums">
                {formatToUserTimezone(arrival, { showTimeZone: false, showSeconds: false })}
              </span>
              {delta ? (
                <span className={cn("shrink-0 font-mono tabular-nums", delta.tone)}>
                  {delta.text}
                </span>
              ) : null}
            </>
          ) : null}
        </span>
      </button>
    </li>
  );
}

/**
 * The first few loads a phrase in the brief names, fetched with the same
 * query and filter the board uses, so the card and the table it opens agree.
 */
export function BriefingLoadsPreview({
  filter,
  onShowAll,
}: {
  filter: ShipmentQuickFilter;
  onShowAll: () => void;
}) {
  const t = useT();
  const timezone = useUserTimezone();
  const openRecord = useOpenShipmentRecord();
  const { data, isLoading, isError } = useQuery({
    ...queries.shipmentBoard.briefingLoads(filter, timezone, BRIEFING_PREVIEW_LOADS),
    staleTime: PREVIEW_STALE_MS,
  });

  const loads = data?.results ?? [];
  const total = data?.count ?? loads.length;
  const label = quickFilterLabel({ filter }, t);

  return (
    <div className="flex flex-col">
      <div className="border-border flex items-center justify-between gap-3 border-b px-3 py-2">
        <span className="text-sm font-medium">{label}</span>
        {data ? (
          <span className="text-muted-foreground font-mono text-xs tabular-nums">{total}</span>
        ) : null}
      </div>
      {isLoading ? (
        <div className="flex flex-col gap-2 p-3" aria-busy>
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-3/4" />
        </div>
      ) : isError ? (
        <p className="text-muted-foreground px-3 py-3 text-xs">
          {t("These loads could not be loaded.")}
        </p>
      ) : loads.length === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-xs">
          {t("None of these loads are on the board anymore.")}
        </p>
      ) : (
        <ul className="flex flex-col p-1">
          {loads.map((shipment) => (
            <PreviewLoad
              key={shipment.id}
              shipment={shipment}
              onOpen={() => shipment.id && openRecord(shipment.id)}
            />
          ))}
        </ul>
      )}
      <button
        type="button"
        onClick={onShowAll}
        className="ui-focus-ring border-border text-muted-foreground hover:text-foreground border-t px-3 py-2 text-left text-xs transition-colors"
      >
        {total > loads.length
          ? t("Show all {0} on the board", total)
          : t("Show on the board")}
      </button>
    </div>
  );
}
