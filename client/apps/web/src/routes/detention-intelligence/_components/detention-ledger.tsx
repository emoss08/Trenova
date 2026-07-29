import { ShareBreakdown, type ShareSegment } from "@/components/detention/detention-charts";
import { RingGauge, type RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { deltaToneClass } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import NumberFlow from "@number-flow/react";
import { m } from "motion/react";
import type { DetentionRollup } from "./use-detention-intelligence";

const CURRENCY_FORMAT = {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 0,
} as const;

function retentionTone(rate: number): RingGaugeTone {
  if (rate >= 0.5) return "success";
  if (rate >= 0.25) return "warning";
  return "critical";
}

function LedgerStat({
  label,
  value,
  detail,
  valueClassName,
}: {
  label: string;
  value: string;
  detail: string;
  valueClassName?: string;
}) {
  return (
    <div className="min-w-0 px-3.5 py-2.5">
      <p className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </p>
      <p className={cn("mt-1 truncate text-sm font-medium tabular-nums", valueClassName)}>
        {value}
      </p>
      <p className="mt-0.5 truncate text-2xs text-muted-foreground tabular-nums">
        {detail}
      </p>
    </div>
  );
}

export function DetentionLedgerSkeleton() {
  return <Skeleton className="h-[232px] w-full rounded-lg" />;
}

/**
 * The page's opening argument: of every dollar detention put in play, how much
 * was kept, how much went straight back out as driver pay, and how much was
 * handed back at someone's discretion.
 */
export function DetentionLedger({ rollup }: { rollup: DetentionRollup }) {
  const { billed, driverPay, netMargin, waived, stopCount, breachCount } = rollup;

  const kept = Math.max(netMargin, 0);
  const overrun = netMargin < 0;
  const exposure = billed + waived;
  const retention = billed > 0 ? netMargin / billed : 0;
  const breachRate = stopCount > 0 ? breachCount / stopCount : 0;
  const marginPerStop = stopCount > 0 ? netMargin / stopCount : 0;

  const segments: ShareSegment[] = [
    {
      key: "kept",
      label: "Kept",
      value: kept,
      className: "bg-emerald-500 dark:bg-emerald-400",
      caption: formatCurrency(kept),
    },
    {
      key: "driverPay",
      label: "Driver pay",
      value: driverPay,
      className: "bg-blue-500 dark:bg-blue-400",
      caption: formatCurrency(driverPay),
    },
    {
      key: "waived",
      label: "Forgiven",
      value: waived,
      className: "bg-amber-500 dark:bg-amber-400",
      caption: formatCurrency(waived),
    },
  ];

  return (
    <m.section
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
      className="relative overflow-hidden rounded-lg border border-border bg-card"
    >
      <span
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-foreground/20 to-transparent"
      />
      <span
        aria-hidden
        className="pointer-events-none absolute -top-28 -right-20 size-64 rounded-full bg-brand/10 blur-3xl"
      />

      <div className="relative flex flex-col gap-5 px-4 py-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1">
          <p className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
            Net detention margin
          </p>
          <p
            className={cn(
              "mt-1.5 text-4xl leading-none font-semibold tracking-tight tabular-nums",
              deltaToneClass(netMargin),
            )}
          >
            <NumberFlow value={netMargin} format={CURRENCY_FORMAT} />
          </p>
          <p className="mt-2 text-2xs text-muted-foreground tabular-nums">
            {stopCount.toLocaleString()} settled{" "}
            {stopCount === 1 ? "stop" : "stops"} across {rollup.facilityCount}{" "}
            {rollup.facilityCount === 1 ? "facility" : "facilities"} ·{" "}
            {Math.round(breachRate * 100)}% ran past free time
            {rollup.truncated ? " · top facilities only" : ""}
          </p>

          <div className="mt-4 max-w-xl">
            <ShareBreakdown segments={segments} />
          </div>

          {overrun ? (
            <p className="mt-3 text-2xs text-red-600 dark:text-red-400">
              Driver detention pay exceeded what was billed — the free-time
              concessions granted to customers are wider than the driver contract
              allows for.
            </p>
          ) : null}
        </div>

        <div className="flex shrink-0 items-center gap-4 sm:flex-col sm:gap-2">
          <RingGauge
            value={Math.max(retention, 0)}
            size={96}
            strokeWidth={7}
            tone={retentionTone(retention)}
            aria-label="Share of billed detention retained after driver pay"
          >
            <div className="text-center">
              <p className="text-lg leading-none font-semibold tabular-nums">
                {Math.round(retention * 100)}%
              </p>
              <p className="mt-0.5 text-2xs text-muted-foreground">retained</p>
            </div>
          </RingGauge>
          <p className="max-w-[9rem] text-2xs leading-snug text-muted-foreground sm:text-center">
            of every billed detention dollar survives driver pay
          </p>
        </div>
      </div>

      <div className="relative grid grid-cols-2 divide-x divide-y divide-border border-t border-border sm:grid-cols-4 sm:divide-y-0">
        <LedgerStat
          label="Billed"
          value={formatCurrency(billed)}
          detail={`${formatCurrency(exposure)} put in play`}
        />
        <LedgerStat
          label="Driver pay"
          value={formatCurrency(driverPay)}
          detail={
            billed > 0
              ? `${Math.round((driverPay / billed) * 100)}% of billed`
              : "no billed detention"
          }
        />
        <LedgerStat
          label="Forgiven"
          value={formatCurrency(waived)}
          detail={
            exposure > 0
              ? `${Math.round((waived / exposure) * 100)}% of exposure`
              : "nothing waived"
          }
          valueClassName={waived > 0 ? "text-amber-600 dark:text-amber-400" : undefined}
        />
        <LedgerStat
          label="Margin per stop"
          value={formatCurrency(marginPerStop)}
          detail={
            rollup.suppressedCount > 0
              ? `${rollup.suppressedCount} lost to no notice`
              : `${rollup.disputeCount} disputed`
          }
          valueClassName={deltaToneClass(marginPerStop)}
        />
      </div>
    </m.section>
  );
}
