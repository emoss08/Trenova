import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { getTodayDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type {
  DataTableGraphQLSource,
  DataTableQueryOptions,
} from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { fetchAllRows } from "@/lib/data-table-export";
import { STAGE_META } from "@/lib/shipment-board/stage";
import { resolveCoverage } from "@/lib/shipment-board/coverage";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import { buildTimelineRows } from "@/lib/shipment-board/timeline";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useShipmentBoardUrl } from "../url-state";

const AXIS_START = 4;
const AXIS_END = 24;
const TICKS = [4, 8, 12, 16, 20];
const TIMELINE_ROW_LIMIT = 300;
const LABEL_WIDTH = 220;

const pct = (hour: number) =>
  ((Math.max(AXIS_START, Math.min(AXIS_END, hour)) - AXIS_START) / (AXIS_END - AXIS_START)) * 100;

const clock = (hour: number) => {
  const wrapped = ((hour % 24) + 24) % 24;
  return `${String(Math.floor(wrapped)).padStart(2, "0")}:${String(Math.round((wrapped % 1) * 60) % 60).padStart(2, "0")}`;
};

type ShipmentTimelineProps = {
  graphql: DataTableGraphQLSource<Shipment>;
  queryOptions: Omit<DataTableQueryOptions, "cursor">;
};

/**
 * Today's loads as bars across the day, from pickup to delivery, with the
 * stretch a late load will overrun hatched in red and a line at the current
 * hour. Clicking a bar opens that shipment in the table.
 */
export default function ShipmentTimeline({ graphql, queryOptions }: ShipmentTimelineProps) {
  const t = useT();
  const timezone = useUserTimezone();
  const [, setUrl] = useShipmentBoardUrl();
  const { data, isLoading } = useQuery({
    queryKey: ["shipment-list", "timeline", graphql.operationName, queryOptions],
    queryFn: () =>
      fetchAllRows<Shipment>({ graphql, options: queryOptions, maxRows: TIMELINE_ROW_LIMIT }),
    staleTime: 15_000,
  });

  const dayStart = getTodayDate(timezone);
  const nowHour = (Date.now() / 1000 - dayStart) / 3600;
  const rows = useMemo(() => buildTimelineRows(data ?? [], dayStart), [data, dayStart]);

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2 p-4">
        {Array.from({ length: 8 }, (_, i) => (
          <Skeleton key={i} className="h-7 w-full" />
        ))}
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="text-muted-foreground flex h-48 items-center justify-center text-sm">
        {t("No shipments with appointments match")}
      </div>
    );
  }

  return (
    <div className="relative min-w-0 overflow-auto" role="list" aria-label={t("Shipment timeline")}>
      <div
        className="bg-canvas border-border sticky top-0 z-10 grid h-(--row-head-h) border-b text-xs"
        style={{ gridTemplateColumns: `${LABEL_WIDTH}px 1fr` }}
      >
        <div className="text-muted-foreground flex items-center px-3 font-medium">
          {t("Shipment")}
        </div>
        <div className="relative">
          {TICKS.map((tick) => (
            <span
              key={tick}
              className="text-muted-foreground absolute top-1/2 -translate-x-1/2 -translate-y-1/2 font-mono tabular-nums"
              style={{ left: `${pct(tick)}%` }}
            >
              {clock(tick)}
            </span>
          ))}
        </div>
      </div>
      {rows.map(({ shipment, pickup, delivery, slipped }, index) => {
        const stage = shipment.stage ?? "NeedsCoverage";
        const meta = STAGE_META[stage];
        const coverage = resolveCoverage(shipment);
        const uncovered = coverage.kind === "uncovered" || coverage.kind === "tendered";
        const startsTomorrow = pickup >= AXIS_END;
        const open = () => void setUrl({ view: "table", expanded: shipment.id ?? null });
        return (
          <div
            key={shipment.id}
            role="listitem"
            className="border-border hover:bg-surface-hover animate-row-rise grid h-(--row-h) border-b"
            style={{
              gridTemplateColumns: `${LABEL_WIDTH}px 1fr`,
              animationDelay: `${Math.min(index, 14) * 18}ms`,
            }}
          >
            <button
              type="button"
              onClick={open}
              className="ui-focus-ring flex min-w-0 flex-col justify-center px-3 text-left"
            >
              <span className="truncate text-sm leading-tight font-medium">
                {getOriginStop(shipment)?.location?.city} →{" "}
                {getDestinationStop(shipment)?.location?.city}
              </span>
              <span className="text-muted-foreground in-data-[density=compact]:hidden truncate font-mono text-xs">
                {shipment.proNumber} · {"name" in coverage ? coverage.name : t("no coverage")}
              </span>
            </button>
            <div className="relative">
              {TICKS.map((tick) => (
                <span
                  key={tick}
                  aria-hidden
                  className="bg-border absolute inset-y-0 w-px"
                  style={{ left: `${pct(tick)}%` }}
                />
              ))}
              {startsTomorrow ? (
                <span className="text-muted-foreground absolute top-1/2 right-2 -translate-y-1/2 text-xs">
                  {t("Pickup tomorrow →")}
                </span>
              ) : (
                <>
                  <button
                    type="button"
                    onClick={open}
                    aria-label={t("Open {0}", shipment.proNumber ?? "")}
                    className={cn(
                      "ui-focus-ring absolute top-1/2 flex h-5 -translate-y-1/2 items-center overflow-hidden rounded-sm px-1.5 font-mono text-xs whitespace-nowrap",
                      uncovered
                        ? "border-warning text-warning-subtle-foreground border border-dashed"
                        : cn(meta.swatchClassName, "text-foreground-on-solid"),
                    )}
                    style={{
                      left: `${pct(pickup)}%`,
                      width: `calc(${Math.max(pct(delivery) - pct(pickup), 1.5)}% - 2px)`,
                    }}
                  >
                    {delivery > AXIS_END ? `→ ${clock(delivery)}` : clock(delivery)}
                  </button>
                  {slipped ? (
                    <span
                      aria-hidden
                      className="absolute top-1/2 h-3 -translate-y-1/2 rounded-r-sm bg-[repeating-linear-gradient(135deg,var(--danger)_0_3px,transparent_3px_6px)] opacity-70"
                      style={{
                        left: `${pct(delivery)}%`,
                        width: `${pct(slipped) - pct(delivery)}%`,
                      }}
                    />
                  ) : null}
                </>
              )}
            </div>
          </div>
        );
      })}
      {nowHour > AXIS_START && nowHour < AXIS_END ? (
        <div
          aria-hidden
          className="bg-danger pointer-events-none absolute top-0 bottom-0 w-px"
          style={{
            left: `calc(${LABEL_WIDTH}px + (100% - ${LABEL_WIDTH}px) * ${pct(nowHour) / 100})`,
          }}
        >
          <span className="bg-danger text-foreground-on-solid absolute top-1 left-1/2 -translate-x-1/2 rounded px-1 font-mono text-2xs">
            {clock(nowHour)}
          </span>
        </div>
      ) : null}
    </div>
  );
}
