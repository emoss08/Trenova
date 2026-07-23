import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { getShipmentPageAnalyticsGraphQL } from "@/lib/graphql/shipment";
import { cn } from "@trenova/shared/lib/utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import type React from "react";
import { useEffect, useMemo, useRef } from "react";
import type { ShipmentAnalyticsData } from "../../analytics/mock-data";

type CustomerMixEntry = ShipmentAnalyticsData["customerMix"]["entries"][number];
type TomorrowPickup = ShipmentAnalyticsData["tomorrowsPickups"]["pickups"][number];
type TomorrowsPickupsCard = ShipmentAnalyticsData["tomorrowsPickups"];
type PickupStatus = TomorrowPickup["status"];

type CustomerMixProps = {
  customerMix: ShipmentAnalyticsData["customerMix"];
  tomorrowsPickups: ShipmentAnalyticsData["tomorrowsPickups"];
  enabled?: boolean;
};

const PICKUP_STATUS: Record<PickupStatus, { label: string; variant: BadgeVariant }> = {
  scheduled: { label: "Scheduled", variant: "outline" },
  confirmed: { label: "Confirmed", variant: "active" },
  tentative: { label: "Tentative", variant: "warning" },
  unassigned: { label: "Needs driver", variant: "warning" },
};

const SHARE_BAR_COLORS = [
  "var(--color-brand)",
  "oklch(0.6 0.18 200)",
  "oklch(0.65 0.16 80)",
  "oklch(0.6 0.16 320)",
  "var(--color-muted-foreground)",
];

const PICKUPS_PAGE_SIZE = 20;

export function CustomerMix({ customerMix, tomorrowsPickups, enabled = true }: CustomerMixProps) {
  return (
    <CustomerMixSection>
      <Tabs defaultValue="customers" className="flex min-h-0 flex-1 flex-col gap-0">
        <header className="flex items-center justify-between border-b border-border px-2 py-1">
          <TabsList
            variant="underline"
            className="h-6 bg-transparent p-0 hover:bg-transparent *:data-[slot=tabs-tab]:hover:bg-transparent"
          >
            <TabsTab value="customers" className="h-6 px-2 text-[11px] hover:bg-transparent">
              Customers
            </TabsTab>
            <TabsTab value="pickups" className="h-7 px-2 text-[11px] hover:bg-transparent">
              Tomorrow&apos;s pickups
            </TabsTab>
          </TabsList>
          <span className="font-mono text-[10px] text-muted-foreground">
            {customerMix.windowDays}d
          </span>
        </header>
        <TabsPanel value="customers" className="min-h-0 flex-1 overflow-y-auto">
          <CustomersList entries={customerMix.entries} />
        </TabsPanel>
        <TabsPanel value="pickups" className="min-h-0 flex-1 overflow-y-auto">
          <PickupsList initialData={tomorrowsPickups} enabled={enabled} />
        </TabsPanel>
      </Tabs>
    </CustomerMixSection>
  );
}

function CustomerMixSection({ children }: { children: React.ReactNode }) {
  return <section className="cc-module-card flex min-h-65 flex-col">{children}</section>;
}

function CustomersList({ entries }: { entries: CustomerMixEntry[] }) {
  if (entries.length === 0) {
    return <EmptyState label="No customer revenue in this window" />;
  }

  return (
    <ul className="flex flex-col gap-2 px-3 py-2">
      {entries.map((entry, index) => (
        <li key={entry.customerId} className="flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between gap-2">
              <span className="truncate text-[11.5px] font-medium">{entry.name}</span>
              <span className="font-mono text-[10.5px] text-muted-foreground tabular-nums">
                ${(entry.revenue / 1000).toFixed(1)}K · {entry.loads}
              </span>
            </div>
            <div aria-hidden className="mt-1 h-1 w-full overflow-hidden rounded-full bg-muted">
              <span
                className="block h-full rounded-full"
                style={{
                  width: `${Math.min(100, entry.share)}%`,
                  background: SHARE_BAR_COLORS[index] ?? SHARE_BAR_COLORS.at(-1),
                }}
              />
            </div>
          </div>
          <span
            className={cn(
              "w-10 text-right font-mono text-[10px] tabular-nums",
              entry.trend > 0 && "text-success",
              entry.trend < 0 && "text-destructive",
              entry.trend === 0 && "text-muted-foreground",
            )}
          >
            {entry.trend > 0 ? "▲" : entry.trend < 0 ? "▼" : "–"}
            {formatPercent(Math.abs(entry.trend))}
          </span>
        </li>
      ))}
    </ul>
  );
}

function PickupsList({
  initialData,
  enabled,
}: {
  initialData: TomorrowsPickupsCard;
  enabled: boolean;
}) {
  const [, setSearchParams] = useQueryStates(panelSearchParamsParser);
  const observerTarget = useRef<HTMLLIElement>(null);

  const query = useInfiniteQuery({
    queryKey: [
      "analytics",
      "shipment-management",
      "tomorrowsPickups",
      Intl.DateTimeFormat().resolvedOptions().timeZone,
    ],
    queryFn: async ({ pageParam }) =>
      getShipmentPageAnalyticsGraphQL({
        include: "tomorrowsPickups",
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        limit: PICKUPS_PAGE_SIZE,
        offset: pageParam,
      }).then((response) => ({
        page: response.page,
        tomorrowsPickups: response.tomorrowsPickups as TomorrowsPickupsCard | undefined,
      })),
    initialPageParam: 0,
    initialData: {
      pageParams: [0],
      pages: [
        {
          page: "shipment-management",
          tomorrowsPickups: initialData,
        },
      ],
    },
    getNextPageParam: (lastPage, _, lastPageParam) => {
      const pickups = (lastPage.tomorrowsPickups as TomorrowsPickupsCard | undefined)?.pickups;
      if (pickups && pickups.length === PICKUPS_PAGE_SIZE) {
        return lastPageParam + PICKUPS_PAGE_SIZE;
      }

      return undefined;
    },
    staleTime: 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });

  const pickups = useMemo(
    () =>
      query.data.pages.flatMap((page) => {
        const card = page.tomorrowsPickups as TomorrowsPickupsCard | undefined;
        return card?.pickups ?? [];
      }),
    [query.data.pages],
  );

  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) observer.observe(currentTarget);

    return () => {
      if (currentTarget) observer.unobserve(currentTarget);
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  if (pickups.length === 0) {
    return <EmptyState label="No pickups scheduled for tomorrow" />;
  }

  return (
    <ul className="flex flex-col">
      {pickups.map((pickup, index) => (
        <li
          key={`${pickup.shipmentId}-${pickup.pickupWindowStart}`}
          className={cn(
            "transition-colors hover:bg-muted",
            index < pickups.length - 1 && "border-b border-border/60",
          )}
        >
          <button
            type="button"
            className="flex w-full cursor-pointer items-center gap-2 px-3 py-1.5 text-left"
            onClick={() =>
              void setSearchParams({
                panelType: "edit",
                panelEntityId: pickup.shipmentId,
              })
            }
          >
            <span className="w-10 font-mono text-[11px] font-semibold tabular-nums">
              {formatPickupTime(pickup.pickupWindowStart)}
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-[11px] font-medium">{pickup.customer}</p>
              <p className="truncate font-mono text-[9.5px] text-muted-foreground">
                {pickup.origin} → {pickup.destination}
              </p>
            </div>
            {pickup.status === "unassigned" || pickup.status === "tentative" ? (
              <Badge variant={PICKUP_STATUS[pickup.status].variant}>
                {PICKUP_STATUS[pickup.status].label}
              </Badge>
            ) : (
              <span className="max-w-20 truncate font-mono text-[10px] text-muted-foreground">
                {pickup.driver || PICKUP_STATUS[pickup.status].label}
              </span>
            )}
          </button>
        </li>
      ))}
      {isFetchingNextPage && (
        <li className="flex items-center justify-center py-3">
          <TextShimmer className="font-mono text-[11px]" duration={1}>
            Loading more...
          </TextShimmer>
        </li>
      )}
      {hasNextPage && <li ref={observerTarget} className="h-px" aria-hidden />}
    </ul>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="flex h-full min-h-32 items-center justify-center px-3 py-6 text-center text-[11px] text-muted-foreground">
      {label}
    </div>
  );
}

function formatPickupTime(unixSeconds: number) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(unixSeconds * 1000));
}

function formatPercent(value: number) {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1,
  }).format(value);
}
