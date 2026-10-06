import { useT } from "@trenova/shared/i18n/use-t";
import {
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  PlugIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn, formatPerMile } from "@trenova/shared/lib/utils";
import type { CapacityGroup, CapacityUnitKind } from "@trenova/graphql/generated/graphql";
import type { CapacityMatch, CapacityUnit, ShipmentCapacity } from "@/lib/graphql/shipment-board";
import { shipmentBoardTableGraphQLConfig } from "@/lib/graphql/shipment";
import { fetchAllRows } from "@/lib/data-table-export";
import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import {
  CAPACITY_PROVIDERS,
  capacityProvidersFor,
  type CapacityProvider,
} from "@/lib/shipment-board/capacity-providers";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useNavigate } from "react-router";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { useBoardActions } from "../use-board-actions";
import { useShipmentBoardUrl } from "../url-state";
import { CapacityAvatar } from "./capacity-avatar";
import { CapacityPopover } from "./capacity-popover";

const UNIT_WIDTH = 68;
const LABEL_WIDTH = 28;
const SEPARATOR_WIDTH = 17;
const SCROLL_STEP = 320;
const LEAVE_MS = 320;
const OVERFLOW_FETCH_LIMIT = 500;
const INTEGRATIONS_PATH = "/admin/integrations";

type DockItem =
  | { type: "label"; key: string; group: CapacityGroup }
  | { type: "separator"; key: string }
  | { type: "unit"; key: string; unit: CapacityUnit; dim: boolean };

function dockItems(units: CapacityUnit[], provider: CapacityProvider): DockItem[] {
  const items: DockItem[] = [];
  provider.groups.forEach((group, index) => {
    const members = units.filter((unit) => unit.group === group);
    if (members.length === 0) return;
    if (items.length > 0) items.push({ type: "separator", key: `sep-${group}` });
    items.push({ type: "label", key: `label-${group}`, group });
    for (const unit of members) {
      items.push({ type: "unit", key: unit.id, unit, dim: index > 0 });
    }
  });
  return items;
}

function itemWidth(item: DockItem) {
  if (item.type === "unit") return UNIT_WIDTH;
  return item.type === "label" ? LABEL_WIDTH : SEPARATOR_WIDTH;
}

/** Uncovered loads, soonest pickup first, for the strip's bulk tender actions. */
function useUncoveredLoads() {
  const timezone = useUserTimezone();
  return useCallback(async () => {
    const graphql = shipmentBoardTableGraphQLConfig({
      quickFilters: [{ filter: "Uncovered" }],
      timezone,
    });
    return fetchAllRows<Shipment>({
      graphql,
      options: { sort: [{ field: "pickupAppointment.scheduledWindowStart", direction: "asc" }] },
      maxRows: OVERFLOW_FETCH_LIMIT,
    });
  }, [timezone]);
}

function Bar({ parts }: { parts: Array<{ value: number; swatchClassName: string }> }) {
  const total = parts.reduce((sum, part) => sum + part.value, 0);
  return (
    <div className="bg-muted flex h-[5px] w-full overflow-hidden rounded-full" aria-hidden>
      {total > 0
        ? parts.map((part, index) => (
            <span key={index} className={part.swatchClassName} style={{ flexGrow: part.value }} />
          ))
        : null}
    </div>
  );
}

type LegendItem = { label: string; swatchClassName?: string };

function Legend({ items }: { items: Array<LegendItem | null> }) {
  return (
    <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
      {items
        .filter((item): item is LegendItem => item !== null)
        .map((item) => (
          <span key={item.label} className="inline-flex items-center gap-1.5">
            {item.swatchClassName ? (
              <span aria-hidden className={cn("size-2 rounded-xs", item.swatchClassName)} />
            ) : null}
            {item.label}
          </span>
        ))}
    </div>
  );
}

const HATCH = "bg-[repeating-linear-gradient(135deg,var(--warning)_0_2px,transparent_2px_4px)]";

function DriverSummary({
  capacity,
  canTender,
}: {
  capacity: ShipmentCapacity;
  canTender: boolean;
}) {
  const t = useT();
  const summary = capacity.drivers;
  const [, setUrl] = useShipmentBoardUrl();
  const actions = useBoardActions();
  const loadUncovered = useUncoveredLoads();
  const [tendered, setTendered] = useState<number | null>(null);
  if (!summary) return null;

  const tenderOverflow = async () => {
    const loads = (await loadUncovered()).filter((load) => load.tenderStatus !== "Tendered");
    const overflow = loads.slice(-summary.short);
    const result = await actions.tender.mutateAsync(
      overflow.map((load) => ({ shipmentId: load.id ?? "" })).filter((item) => item.shipmentId),
    );
    setTendered(result.tendered.length);
    void setUrl({ capacity: "Carrier" });
    toast.success(t("Tendered {0} loads to your carrier network", result.tendered.length));
  };

  return (
    <>
      <div className="flex items-baseline gap-2">
        <b className="font-mono text-3xl font-semibold tracking-tight tabular-nums">
          {summary.ready}
        </b>
        <span className="text-muted-foreground text-sm">
          {t("drivers ready for {0} uncovered loads", summary.uncovered)}
        </span>
      </div>
      <Bar
        parts={[
          { value: summary.ready, swatchClassName: "bg-success" },
          { value: summary.withinTwoHours, swatchClassName: "bg-success/40" },
          { value: summary.short, swatchClassName: HATCH },
        ]}
      />
      <Legend
        items={[
          { label: t("{0} ready", summary.ready), swatchClassName: "bg-success" },
          { label: t("{0} within 2h", summary.withinTwoHours), swatchClassName: "bg-success/40" },
          summary.short > 0
            ? { label: t("{0} short", summary.short), swatchClassName: HATCH }
            : null,
        ]}
      />
      {summary.short > 0 ? (
        canTender ? (
          tendered != null ? (
            <span className="text-success inline-flex items-center gap-1 text-sm">
              <CheckIcon className="size-3.5" />
              {t("{0} loads sent to carriers", tendered)}
            </span>
          ) : (
            <Button
              size="sm"
              variant="outline"
              className="self-start"
              isLoading={actions.tender.isPending}
              onClick={() => void tenderOverflow()}
            >
              {t("Tender {0} to carriers", summary.short)}
            </Button>
          )
        ) : (
          <Button
            size="sm"
            variant="outline"
            className="self-start"
            onClick={() => void setUrl({ qf: [{ filter: "Uncovered" }], view: "table" })}
          >
            {t("Review {0} you can't cover", summary.short)}
          </Button>
        )
      ) : null}
    </>
  );
}

function CarrierSummary({ capacity }: { capacity: ShipmentCapacity }) {
  const t = useT();
  const summary = capacity.carriers;
  const actions = useBoardActions();
  const loadUncovered = useUncoveredLoads();
  if (!summary) return null;

  const tenderAll = async () => {
    const loads = (await loadUncovered()).filter((load) => load.tenderStatus !== "Tendered");
    const result = await actions.tender.mutateAsync(
      loads.map((load) => ({ shipmentId: load.id ?? "" })).filter((item) => item.shipmentId),
    );
    toast.success(t("Tendered {0} loads to best-match carriers", result.tendered.length));
  };

  return (
    <>
      <div className="flex items-baseline gap-2">
        <b className="font-mono text-3xl font-semibold tracking-tight tabular-nums">
          {summary.posting}
        </b>
        <span className="text-muted-foreground text-sm">
          {t("carriers posting trucks for {0} untendered loads", summary.untendered)}
        </span>
      </div>
      <Bar
        parts={[
          { value: summary.awaitingAcceptance, swatchClassName: "bg-success/40" },
          { value: summary.untendered, swatchClassName: HATCH },
        ]}
      />
      <Legend
        items={[
          summary.awaitingAcceptance > 0
            ? {
                label: t("{0} awaiting acceptance", summary.awaitingAcceptance),
                swatchClassName: "bg-success/40",
              }
            : null,
          summary.untendered > 0
            ? { label: t("{0} not tendered", summary.untendered), swatchClassName: HATCH }
            : null,
          summary.avgRatePerMile
            ? { label: t("avg {0}", formatPerMile(Number(summary.avgRatePerMile))) }
            : null,
        ]}
      />
      {summary.untendered > 0 ? (
        <Button
          size="sm"
          variant="outline"
          className="self-start"
          isLoading={actions.tender.isPending}
          onClick={() => void tenderAll()}
        >
          {t("Tender all {0} to best matches", summary.untendered)}
        </Button>
      ) : summary.awaitingAcceptance > 0 ? (
        <span className="text-success inline-flex items-center gap-1 text-sm">
          <CheckIcon className="size-3.5" />
          {t("Everything is tendered")}
        </span>
      ) : null}
    </>
  );
}

function CapacityDock({
  capacity,
  provider,
}: {
  capacity: ShipmentCapacity;
  provider: CapacityProvider;
}) {
  const t = useT();
  const navigate = useNavigate();
  const { hos } = useShipmentCapabilities();
  const actions = useBoardActions();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [openId, setOpenId] = useState<string | null>(null);
  const [leaving, setLeaving] = useState<Set<string>>(new Set());
  const [edges, setEdges] = useState({ start: true, end: false });
  const items = useMemo(() => dockItems(capacity.units, provider), [capacity.units, provider]);

  const virtualizer = useVirtualizer({
    horizontal: true,
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => itemWidth(items[index]),
    overscan: 8,
  });

  const updateEdges = useCallback(() => {
    const element = scrollRef.current;
    if (!element) return;
    setEdges({
      start: element.scrollLeft <= 2,
      end: element.scrollLeft + element.clientWidth >= element.scrollWidth - 2,
    });
  }, []);

  useEffect(() => {
    updateEdges();
    const element = scrollRef.current;
    if (!element || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(updateEdges);
    observer.observe(element);
    return () => observer.disconnect();
  }, [updateEdges, items.length]);

  const commit = (unit: CapacityUnit, match: CapacityMatch) => {
    setOpenId(null);
    setLeaving((current) => new Set(current).add(unit.id));
    const done = () =>
      window.setTimeout(
        () =>
          setLeaving((current) => {
            const next = new Set(current);
            next.delete(unit.id);
            return next;
          }),
        LEAVE_MS,
      );
    if (unit.kind === "Driver") {
      actions.assign.mutate(
        [{ moveId: match.moveId, workerId: unit.id, tractorId: unit.tractorId }],
        {
          onSuccess: () =>
            toast.success(t("{0} assigned to {1}", unit.name, match.proNumber ?? "")),
          onSettled: done,
        },
      );
    } else {
      actions.tender.mutate([{ shipmentId: match.shipmentId, carrierId: unit.id }], {
        onSuccess: () => toast.success(t("Tendered {0} to {1}", match.proNumber ?? "", unit.name)),
        onSettled: done,
      });
    }
  };

  const formatTime = (unix: number) =>
    formatToUserTimezone(unix, { showTimeZone: false, showSeconds: false, showDate: false });
  const scrollBy = (delta: number) =>
    scrollRef.current?.scrollBy({ left: delta, behavior: "smooth" });

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1.5">
      <div className="relative min-w-0">
        <div
          ref={scrollRef}
          onScroll={() => {
            updateEdges();
            if (openId) setOpenId(null);
          }}
          className="overflow-x-auto overflow-y-hidden pr-12 pl-6 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
          role="list"
          aria-label={provider.kind === "Driver" ? t("Available drivers") : t("Available carriers")}
        >
          {items.length === 0 ? (
            <p className="text-muted-foreground py-6 text-sm">
              {provider.kind === "Driver"
                ? t("No drivers are free right now.")
                : t("No carriers have capacity on your lanes.")}
            </p>
          ) : (
            <div className="relative h-[76px]" style={{ width: virtualizer.getTotalSize() }}>
              {virtualizer.getVirtualItems().map((virtual) => {
                const item = items[virtual.index];
                return (
                  <div
                    key={item.key}
                    role="listitem"
                    className="absolute top-0 flex h-full items-center"
                    style={{ left: virtual.start, width: virtual.size }}
                  >
                    {item.type === "separator" ? (
                      <span aria-hidden className="bg-border mx-2 h-12 w-px" />
                    ) : item.type === "label" ? (
                      <span className="text-muted-foreground mx-auto text-2xs font-medium [writing-mode:vertical-rl] rotate-180">
                        {provider.groupLabel(item.group, t)}
                      </span>
                    ) : (
                      <CapacityPopover
                        unit={item.unit}
                        provider={provider}
                        hos={hos}
                        open={openId === item.unit.id}
                        onOpenChange={(open) => setOpenId(open ? item.unit.id : null)}
                        onCommit={commit}
                        pending={actions.assign.isPending || actions.tender.isPending}
                        trigger={
                          <CapacityAvatar
                            unit={item.unit}
                            provider={provider}
                            caption={provider.caption(item.unit, t, formatTime)}
                            showRing={provider.kind === "Carrier" || hos}
                            dim={item.dim}
                            leaving={leaving.has(item.unit.id)}
                            active={openId === item.unit.id}
                          />
                        }
                      />
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
        <span
          aria-hidden
          className={cn(
            "from-canvas pointer-events-none absolute inset-y-0 left-0 w-6 bg-gradient-to-r to-transparent transition-opacity",
            edges.start ? "opacity-0" : "opacity-100",
          )}
        />
        <span
          aria-hidden
          className={cn(
            "from-canvas pointer-events-none absolute inset-y-0 right-0 w-12 bg-gradient-to-l to-transparent transition-opacity",
            edges.end ? "opacity-0" : "opacity-100",
          )}
        />
      </div>
      <div className="text-muted-foreground flex items-center gap-2 pl-6 text-xs">
        <span className="truncate">{provider.ringHint(t, hos)}</span>
        {provider.kind === "Driver" && !hos ? (
          <button
            type="button"
            onClick={() => void navigate(INTEGRATIONS_PATH)}
            className="ui-focus-ring text-brand inline-flex shrink-0 items-center gap-1 rounded-sm hover:underline"
          >
            <PlugIcon className="size-3" />
            {t("Connect ELD for hours of service")}
          </button>
        ) : null}
        <span className="flex-1" />
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Scroll left")}
          disabled={edges.start}
          onClick={() => scrollBy(-SCROLL_STEP)}
        >
          <ChevronLeftIcon className="size-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Scroll right")}
          disabled={edges.end}
          onClick={() => scrollBy(SCROLL_STEP)}
        >
          <ChevronRightIcon className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}

/**
 * Who can take the uncovered loads right now: a summary of driver or carrier
 * capacity against the loads waiting, and a dock of the drivers or carriers
 * themselves. An organization that does both switches between the two.
 */
export function CapacityStrip() {
  const t = useT();
  const { operationType, runsBrokerage } = useShipmentCapabilities();
  const providers = capacityProvidersFor(operationType);
  const [{ capacity: requested }, setUrl] = useShipmentBoardUrl();
  const kind: CapacityUnitKind =
    providers.find((provider) => provider.kind === requested)?.kind ?? providers[0].kind;
  const provider = CAPACITY_PROVIDERS[kind];
  const driverCount = useQuery({
    ...queries.shipmentBoard.capacity("Driver"),
    enabled: operationType === "both",
    staleTime: 15_000,
  });
  const carrierCount = useQuery({
    ...queries.shipmentBoard.capacity("Carrier"),
    enabled: operationType === "both",
    staleTime: 15_000,
  });
  const { data, isLoading } = useQuery({
    ...queries.shipmentBoard.capacity(kind),
    staleTime: 15_000,
  });

  const tabCount = (target: CapacityUnitKind) =>
    target === "Driver" ? driverCount.data?.drivers?.ready : carrierCount.data?.carriers?.posting;

  return (
    <section
      aria-label={t("Capacity")}
      className="flex min-w-0 flex-col gap-3 @[760px]/board:flex-row @[760px]/board:items-stretch"
    >
      <div className="flex shrink-0 flex-col gap-2 @[760px]/board:w-[260px] @[760px]/board:border-r @[760px]/board:border-border @[760px]/board:pr-4">
        {providers.length > 1 ? (
          <div
            role="tablist"
            aria-label={t("Capacity")}
            className="border-input bg-card flex h-7 self-start rounded-md border p-0.5"
          >
            {providers.map((entry) => (
              <button
                key={entry.kind}
                type="button"
                role="tab"
                aria-selected={entry.kind === kind}
                onClick={() => void setUrl({ capacity: entry.kind })}
                className={cn(
                  "ui-focus-ring text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 rounded-sm px-2 text-sm font-medium",
                  entry.kind === kind && "bg-surface-active text-foreground",
                )}
              >
                {t(entry.tabLabel)}
                <span className="text-muted-foreground font-mono text-xs tabular-nums">
                  {tabCount(entry.kind) ?? ""}
                </span>
              </button>
            ))}
          </div>
        ) : (
          <span className="text-muted-foreground text-xs font-medium">
            {kind === "Driver" ? t("Driver capacity") : t("Carrier capacity")}
          </span>
        )}
        {isLoading || !data ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-8 w-40" />
            <Skeleton className="h-1.5 w-full" />
            <Skeleton className="h-4 w-48" />
          </div>
        ) : kind === "Driver" ? (
          <DriverSummary capacity={data} canTender={runsBrokerage} />
        ) : (
          <CarrierSummary capacity={data} />
        )}
      </div>
      {isLoading || !data ? (
        <div className="flex flex-1 items-center gap-3 pl-6">
          {Array.from({ length: 8 }, (_, i) => (
            <Skeleton key={i} className="size-10 rounded-full" />
          ))}
        </div>
      ) : (
        <CapacityDock key={kind} capacity={data} provider={provider} />
      )}
    </section>
  );
}
