import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatRange } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { fetchMyPeriodSummary, fetchMyRecentPayEvents } from "@trenova/shared/lib/graphql/driver-portal";
import { useQuery } from "@tanstack/react-query";
import { ChevronRightIcon, ReceiptTextIcon } from "lucide-react";
import { m } from "motion/react";
import { Link } from "react-router";
import { LoadCard } from "../_components/load-card";
import { useDashProfile } from "../_components/dash-layout";
import { useMyLoads } from "../_components/use-loads";

function greeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 17) return "Good afternoon";
  return "Good evening";
}

function daysUntil(unix: number): number {
  return Math.max(0, Math.ceil((unix - Date.now() / 1000) / 86400));
}

function Section({
  title,
  to,
  toLabel,
  delay,
  children,
}: {
  title: string;
  to?: string;
  toLabel?: string;
  delay: number;
  children: React.ReactNode;
}) {
  return (
    <m.section
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut", delay }}
      className="flex flex-col gap-3"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">{title}</h2>
        {to ? (
          <Link
            to={to}
            className="flex items-center text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            {toLabel} <ChevronRightIcon className="size-3.5" />
          </Link>
        ) : null}
      </div>
      {children}
    </m.section>
  );
}

export function DashHomePage() {
  const { data: profile } = useDashProfile();
  const period = useQuery({
    queryKey: ["dash-period-summary"],
    queryFn: fetchMyPeriodSummary,
  });
  const loads = useMyLoads("Active");
  const events = useQuery({
    queryKey: ["dash-recent-pay-events"],
    queryFn: () => fetchMyRecentPayEvents(5),
  });

  const currentLoad = loads.data?.[0];

  return (
    <div className="flex flex-col gap-6">
      <m.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.22, ease: "easeOut" }}
      >
        <h1 className="text-xl font-semibold tracking-tight">
          {greeting()}
          {profile ? `, ${profile.firstName}` : ""}
        </h1>
        <p className="text-sm text-muted-foreground">Here&apos;s where you stand.</p>
      </m.div>

      <m.section
        initial={{ opacity: 0, y: 10, scale: 0.99 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.25, ease: "easeOut", delay: 0.04 }}
        className="relative overflow-hidden rounded-2xl border border-zinc-800 bg-zinc-950 p-5 text-zinc-50"
      >
        <div
          aria-hidden
          className="pointer-events-none absolute -top-24 -right-16 size-56 rounded-full bg-teal-500/20 blur-3xl"
        />
        <div
          aria-hidden
          className="pointer-events-none absolute -bottom-28 -left-10 size-56 rounded-full bg-indigo-500/15 blur-3xl"
        />
        <p className="text-2xs font-medium tracking-wide text-zinc-400 uppercase">
          Earned this period
        </p>
        {period.isPending ? (
          <Skeleton className="mt-2 h-10 w-40 bg-zinc-800" />
        ) : period.data ? (
          <>
            <p className="mt-1 text-4xl font-semibold tracking-tight tabular-nums">
              {formatCurrency(period.data.accruedGrossMinor / 100)}
            </p>
            <div className="mt-3 flex flex-wrap items-center gap-1.5">
              <span className="rounded-full bg-zinc-800/80 px-2.5 py-1 text-xs text-zinc-300">
                {period.data.eventCount} load{period.data.eventCount === 1 ? "" : "s"}
              </span>
              <span className="rounded-full bg-zinc-800/80 px-2.5 py-1 text-xs text-zinc-300">
                {formatRange(period.data.periodStart, period.data.periodEnd)}
              </span>
              <span className="rounded-full bg-teal-500/15 px-2.5 py-1 text-xs font-medium text-teal-300">
                {daysUntil(period.data.payDate) === 0
                  ? "Settles today"
                  : `Settles in ${daysUntil(period.data.payDate)}d`}
              </span>
            </div>
          </>
        ) : (
          <p className="mt-2 text-sm text-zinc-400">
            We couldn&apos;t load your pay period right now.
          </p>
        )}
      </m.section>

      <Section title="Current load" to="/dash/loads" toLabel="All loads" delay={0.08}>
        {loads.isPending ? (
          <Skeleton className="h-36 w-full rounded-2xl" />
        ) : currentLoad ? (
          <LoadCard load={currentLoad} />
        ) : (
          <div className="rounded-2xl border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            No active loads right now.
          </div>
        )}
      </Section>

      <Section title="Recent pay activity" to="/dash/pay" toLabel="Settlements" delay={0.12}>
        {events.isPending ? (
          <Skeleton className="h-28 w-full rounded-2xl" />
        ) : events.data && events.data.length > 0 ? (
          <ul className="divide-y divide-border rounded-2xl border border-border bg-card">
            {events.data.map((event) => (
              <li key={event.id} className="flex items-center justify-between gap-3 px-4 py-3">
                <div className="min-w-0">
                  <p className="truncate font-mono text-sm font-medium">
                    {event.proNumber || "Pay event"}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {formatRange(event.eventDate, event.eventDate)}
                    {Number(event.totalMiles) > 0 ? ` · ${event.totalMiles} mi` : ""}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  {event.onHold ? <Badge variant="warning">Held</Badge> : null}
                  <span className="text-sm font-semibold tabular-nums">
                    <AmountDisplay value={event.grossAmountMinor} currency={event.currencyCode} />
                  </span>
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-8 text-center">
            <ReceiptTextIcon className="size-6 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Pay from your loads will show up here as you run them.
            </p>
          </div>
        )}
      </Section>
    </div>
  );
}
