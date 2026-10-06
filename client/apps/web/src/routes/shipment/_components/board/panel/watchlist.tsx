import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatMinutesSpan, userWallClockNow } from "@trenova/shared/lib/date";
import { cn, formatCompactCurrency, formatCurrency } from "@trenova/shared/lib/utils";
import type { UncoveredPickupWindow } from "@trenova/graphql/generated/graphql";
import { HoldButton } from "@/components/hold-button";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import type { ShipmentWatchlist } from "@/lib/graphql/shipment-board";
import { queries } from "@/lib/queries";
import type { QuickFilterToken } from "@/lib/shipment-board/quick-filters";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";
import { useBoardActions } from "../use-board-actions";
import { useShipmentBoardUrl } from "../url-state";

const WATCHLIST_STALE_MS = 30_000;
const TICK_MS = 1_000;
const URGENT_SECONDS = 3_600;

function useNow(enabled = true) {
  const [now, setNow] = useState(() => Date.now() / 1000);
  useEffect(() => {
    if (!enabled) return;
    const timer = window.setInterval(() => setNow(Date.now() / 1000), TICK_MS);
    return () => window.clearInterval(timer);
  }, [enabled]);
  return now;
}

function useBoardFilter() {
  const [, setUrl] = useShipmentBoardUrl();
  return (tokens: QuickFilterToken[], expanded: string | null = null) =>
    void setUrl({ qf: tokens, view: "table", expanded });
}

function Section({
  title,
  figure,
  children,
}: {
  title: string;
  figure: ReactNode;
  children: ReactNode;
}) {
  return (
    <section
      className="border-border flex flex-col gap-2.5 border-t pt-3 first:border-t-0 first:pt-0"
      aria-label={title}
    >
      <header className="flex items-baseline justify-between gap-2">
        <span className="text-muted-foreground text-xs font-medium">{title}</span>
        <span className="font-mono text-sm tabular-nums">{figure}</span>
      </header>
      {children}
    </section>
  );
}

function Deliveries({ data }: { data: ShipmentWatchlist["deliveries"] }) {
  const t = useT();
  const filter = useBoardFilter();
  const [hover, setHover] = useState<number | null>(null);
  const timezone = useUserTimezone();
  const wallClock = userWallClockNow(timezone);
  const hourNow = wallClock.getHours() + wallClock.getMinutes() / 60;
  const max = Math.max(1, ...data.buckets.map((b) => b.delivered + b.scheduled + b.late));
  const hovered = data.buckets.find((bucket) => bucket.hour === hover);

  return (
    <Section
      title={t("Today's deliveries")}
      figure={t("{0} / {1} on time", data.onTime.toLocaleString(), data.total.toLocaleString())}
    >
      <div className="flex flex-col gap-1">
        <div className="relative flex h-14 items-end gap-0.5" onMouseLeave={() => setHover(null)}>
          {data.buckets.map((bucket) => {
            const total = bucket.delivered + bucket.scheduled + bucket.late;
            return (
              <button
                key={bucket.hour}
                type="button"
                aria-label={t(
                  "{0}:00 · {1} deliveries",
                  String(bucket.hour).padStart(2, "0"),
                  total,
                )}
                onMouseEnter={() => setHover(bucket.hour)}
                onFocus={() => setHover(bucket.hour)}
                onClick={() => filter([{ filter: "DeliveryHour", hour: bucket.hour }])}
                className={cn(
                  "ui-focus-ring flex h-full min-w-0 flex-1 flex-col-reverse rounded-xs",
                  hover === bucket.hour && "bg-surface-hover",
                )}
              >
                <span
                  className="bg-success w-full"
                  style={{ height: `${(bucket.delivered / max) * 100}%` }}
                />
                <span
                  className="bg-brand/60 w-full"
                  style={{ height: `${(bucket.scheduled / max) * 100}%` }}
                />
                <span
                  className="bg-danger w-full rounded-t-[2px]"
                  style={{ height: `${(bucket.late / max) * 100}%` }}
                />
              </button>
            );
          })}
          {hourNow >= 6 && hourNow < 24 ? (
            <span
              aria-hidden
              className="bg-foreground pointer-events-none absolute inset-y-0 w-px"
              style={{ left: `${((hourNow - 6) / 18) * 100}%` }}
            />
          ) : null}
        </div>
        <div
          className="text-muted-foreground flex justify-between font-mono text-2xs tabular-nums"
          aria-hidden
        >
          {[6, 12, 18, 24].map((hour) => (
            <span key={hour}>{String(hour).padStart(2, "0")}</span>
          ))}
        </div>
        <span className="text-muted-foreground min-h-4 text-xs">
          {hovered
            ? t(
                "{0}:00 · {1} delivered · {2} scheduled · {3} late",
                String(hovered.hour).padStart(2, "0"),
                hovered.delivered,
                hovered.scheduled,
                hovered.late,
              )
            : t("Click an hour to see its loads")}
        </span>
      </div>
      {data.worstLate.length > 0 ? (
        <div className="flex flex-col">
          {data.worstLate.map((late) => (
            <button
              key={late.shipmentId}
              type="button"
              onClick={() => filter([{ filter: "Late" }], late.shipmentId)}
              className="ui-focus-ring hover:bg-surface-hover -mx-2 flex h-7 items-center gap-2 rounded-md px-2 text-left text-sm"
            >
              <span className="text-danger w-16 shrink-0 font-mono text-xs tabular-nums">
                +{formatMinutesSpan(late.deltaMinutes)}
              </span>
              <span className="min-w-0 flex-1 truncate">{late.city}</span>
              <span className="text-muted-foreground max-w-28 truncate text-xs">
                {late.customerName}
              </span>
            </button>
          ))}
          <Button
            variant="link"
            size="xs"
            className="self-start px-0"
            onClick={() => filter([{ filter: "Late" }])}
          >
            {t("Review all {0} late", data.lateCount)}
          </Button>
        </div>
      ) : null}
    </Section>
  );
}

const WINDOW_LABEL: Record<UncoveredPickupWindow, string> = {
  UnderTwoHours: "< 2h",
  TwoToSixHours: "2–6h",
  LaterToday: "Later today",
  TomorrowOrLater: "Tomorrow+",
};

function Countdown({ until }: { until: number }) {
  const now = useNow();
  const left = Math.max(0, Math.round(until - now));
  const hours = Math.floor(left / 3600);
  const minutes = Math.floor((left % 3600) / 60);
  const seconds = left % 60;
  return (
    <span className={cn("font-mono tabular-nums", left < URGENT_SECONDS && "text-danger")}>
      {hours}:{String(minutes).padStart(2, "0")}:{String(seconds).padStart(2, "0")}
    </span>
  );
}

function Uncovered({ data }: { data: ShipmentWatchlist["uncovered"] }) {
  const t = useT();
  const filter = useBoardFilter();
  return (
    <Section
      title={t("Uncovered pickups")}
      figure={t("{0} · {1} loads", formatCurrency(Number(data.revenue)), data.count)}
    >
      <div className="grid grid-cols-4 gap-1">
        {data.windows.map((window) => (
          <button
            key={window.window}
            type="button"
            onClick={() =>
              filter([
                {
                  filter: "PickupWindow",
                  windowStartMinutes: window.startMinutes,
                  windowEndMinutes: window.endMinutes ?? undefined,
                },
              ])
            }
            className={cn(
              "ui-focus-ring border-border hover:border-border-strong flex flex-col items-start gap-0.5 rounded-md border p-1.5 text-left",
              window.window === "UnderTwoHours" &&
                window.count > 0 &&
                "border-danger-border bg-danger-subtle",
            )}
          >
            <span className="text-muted-foreground text-2xs">{t(WINDOW_LABEL[window.window])}</span>
            <span className="font-mono text-base font-semibold tabular-nums">{window.count}</span>
            <span className="text-muted-foreground font-mono text-2xs tabular-nums">
              {formatCompactCurrency(Number(window.revenue))}
            </span>
          </button>
        ))}
      </div>
      {data.next ? (
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground text-xs">{t("Next pickup in")}</span>
          <Countdown until={data.next.pickupAt} />
          <span className="min-w-0 flex-1 truncate text-xs">
            {data.next.originCity} → {data.next.destinationCity}
          </span>
          <Button
            size="xs"
            variant="outline"
            onClick={() => filter([{ filter: "Uncovered" }], data.next?.shipmentId ?? null)}
          >
            {t("Cover")}
          </Button>
        </div>
      ) : null}
    </Section>
  );
}

function AccruingAmount({
  base,
  ratePerHour,
  since,
}: {
  base: number;
  ratePerHour: number;
  since: number;
}) {
  const now = useNow();
  const amount = base + (Math.max(0, now - since) / 3600) * ratePerHour;
  return <>{formatCurrency(amount)}</>;
}

function Detention({ data }: { data: ShipmentWatchlist["detention"] }) {
  const t = useT();
  const filter = useBoardFilter();
  const actions = useBoardActions();
  const [billed, setBilled] = useState<Set<string>>(new Set());
  const longest = Math.max(1, ...data.top.map((row) => data.snapshotAt - row.billableSince));

  return (
    <Section
      title={t("Detention accruing")}
      figure={
        <>
          <AccruingAmount
            base={Number(data.amount)}
            ratePerHour={Number(data.ratePerHour)}
            since={data.snapshotAt}
          />
          <span className="text-muted-foreground"> · {t("{0} stops", data.stopCount)}</span>
        </>
      }
    >
      {data.top.length === 0 ? (
        <span className="text-muted-foreground text-sm">{t("No detention is accruing.")}</span>
      ) : (
        <div className="flex flex-col gap-1.5">
          {data.top.map((row) => {
            const isBilled = billed.has(row.stopId);
            const elapsedMinutes = Math.round((data.snapshotAt - row.billableSince) / 60);
            return (
              <div key={row.stopId} className="flex flex-col gap-1">
                <div className="flex items-center gap-2 text-sm">
                  <span className="min-w-0 flex-1 truncate">
                    {row.facilityName}
                    {row.coverageName ? (
                      <span className="text-muted-foreground"> · {row.coverageName}</span>
                    ) : null}
                  </span>
                  <span className="text-muted-foreground font-mono text-xs tabular-nums">
                    {formatMinutesSpan(elapsedMinutes)}
                  </span>
                  <span className="font-mono text-xs tabular-nums">
                    {isBilled ? (
                      formatCurrency(Number(row.amount))
                    ) : (
                      <AccruingAmount
                        base={Number(row.amount)}
                        ratePerHour={Number(row.ratePerHour)}
                        since={data.snapshotAt}
                      />
                    )}
                  </span>
                  <Button
                    size="xs"
                    variant="outline"
                    disabled={isBilled || !row.occurrenceId || actions.approveDetention.isPending}
                    onClick={() =>
                      row.occurrenceId &&
                      actions.approveDetention.mutate(row.occurrenceId, {
                        onSuccess: () => setBilled((current) => new Set(current).add(row.stopId)),
                      })
                    }
                  >
                    {isBilled ? t("Billed") : t("Bill")}
                  </Button>
                </div>
                <span className="bg-muted h-1 overflow-hidden rounded-full">
                  <span
                    className="bg-warning block h-full rounded-full"
                    style={{ width: `${((data.snapshotAt - row.billableSince) / longest) * 100}%` }}
                  />
                </span>
              </div>
            );
          })}
          <Button
            variant="link"
            size="xs"
            className="self-start px-0"
            onClick={() => filter([{ filter: "Detention" }])}
          >
            {t("Review all {0}", data.stopCount)}
          </Button>
        </div>
      )}
    </Section>
  );
}

function Billing({ data }: { data: ShipmentWatchlist["billing"] }) {
  const t = useT();
  const filter = useBoardFilter();
  const actions = useBoardActions();
  const [transferred, setTransferred] = useState(false);
  const top = Math.max(1, ...data.customers.map((customer) => Number(customer.total)));

  return (
    <Section
      title={t("Ready to bill")}
      figure={t("{0} · {1} loads", formatCurrency(Number(data.total)), data.count)}
    >
      {data.customers.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          {data.customers.map((customer) => (
            <div key={customer.customerId} className="flex items-center gap-2 text-sm">
              <span className="min-w-0 flex-1 truncate">{customer.name}</span>
              <span className="bg-muted h-1.5 w-20 overflow-hidden rounded-full">
                <span
                  className="bg-success block h-full rounded-full"
                  style={{ width: `${(Number(customer.total) / top) * 100}%` }}
                />
              </span>
              <span className="text-muted-foreground w-6 text-right font-mono text-xs tabular-nums">
                {customer.count}
              </span>
              <span className="w-16 text-right font-mono text-xs tabular-nums">
                {formatCompactCurrency(Number(customer.total))}
              </span>
            </div>
          ))}
          {data.moreCustomers > 0 ? (
            <span className="text-muted-foreground text-xs">
              {t("+{0} more customers", data.moreCustomers)}
            </span>
          ) : null}
        </div>
      ) : null}
      <div className="flex items-center gap-2">
        <div className="flex-1">
          <HoldButton
            label={t("Transfer {0} to billing", data.count)}
            doneLabel={t("Transferred to billing")}
            done={transferred}
            disabled={data.count === 0 || actions.transferReadyToBill.isPending}
            onConfirm={() =>
              actions.transferReadyToBill.mutate(undefined, {
                onSuccess: () => setTransferred(true),
              })
            }
          />
        </div>
        <Button size="sm" variant="ghost" onClick={() => filter([{ filter: "ReadyToBill" }])}>
          {t("Review first")}
        </Button>
      </div>
    </Section>
  );
}

/**
 * The day at a glance, counted on the server: deliveries by hour, uncovered
 * pickups by window, detention as it accrues, and what is ready to bill. Each
 * figure filters the table to the loads behind it.
 */
export function Watchlist() {
  const timezone = useUserTimezone();
  const { data, isLoading } = useQuery({
    ...queries.shipmentBoard.watchlist(timezone),
    staleTime: WATCHLIST_STALE_MS,
  });

  if (isLoading || !data) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-20 w-full" />
        <Skeleton className="h-20 w-full" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <Deliveries data={data.deliveries} />
      <Uncovered data={data.uncovered} />
      <Detention data={data.detention} />
      <Billing data={data.billing} />
    </div>
  );
}
