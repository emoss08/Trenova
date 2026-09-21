import { useT } from "@trenova/shared/i18n/use-t";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import type { Tone } from "@/components/kpi/tone";
import { ShareBreakdown, type ShareSegment } from "@/components/detention/detention-charts";
import { RingGauge, type RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatCurrency } from "@trenova/shared/lib/utils";
import NumberFlow from "@number-flow/react";
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

function signTone(value: number): Tone | undefined {
  if (value > 0) return "success";
  if (value < 0) return "danger";
  return undefined;
}

export function DetentionLedgerSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      <KpiStripSkeleton count={5} />
      <Skeleton className="h-32 w-full rounded-lg" />
    </div>
  );
}

/**
 * The page's opening argument: of every dollar detention put in play, how much
 * was kept, how much went straight back out as driver pay, and how much was
 * handed back at someone's discretion.
 */
export function DetentionLedger({ rollup }: { rollup: DetentionRollup }) {
  const t = useT();

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
      label: t("Kept"),
      value: kept,
      className: "bg-success",
      caption: formatCurrency(kept),
    },
    {
      key: "driverPay",
      label: t("Driver pay"),
      value: driverPay,
      className: "bg-info",
      caption: formatCurrency(driverPay),
    },
    {
      key: "waived",
      label: t("Forgiven"),
      value: waived,
      className: "bg-warning",
      caption: formatCurrency(waived),
    },
  ];

  return (
    <section className="flex flex-col gap-3">
      <KpiStrip>
        <KpiStripItem
          size="lg"
          className="col-span-full"
          label={t("Net detention margin")}
          tone={signTone(netMargin)}
          value={<NumberFlow value={netMargin} format={CURRENCY_FORMAT} />}
          sub={t(
            "{0} settled {1} across {2} {3} · {4}% ran past free time {5}",
            stopCount.toLocaleString(),
            stopCount === 1 ? "stop" : "stops",
            rollup.facilityCount,
            rollup.facilityCount === 1 ? "facility" : "facilities",
            Math.round(breachRate * 100),
            rollup.truncated ? ` ${t("· top facilities only")}` : "",
          )}
        />
        <KpiStripItem
          label={t("Billed")}
          value={formatCurrency(billed)}
          sub={`${formatCurrency(exposure)} put in play`}
        />
        <KpiStripItem
          label={t("Driver pay")}
          value={formatCurrency(driverPay)}
          sub={
            billed > 0
              ? `${Math.round((driverPay / billed) * 100)}% of billed`
              : "no billed detention"
          }
        />
        <KpiStripItem
          label={t("Forgiven")}
          tone={waived > 0 ? "warning" : undefined}
          value={formatCurrency(waived)}
          sub={
            exposure > 0
              ? `${Math.round((waived / exposure) * 100)}% of exposure`
              : "nothing waived"
          }
        />
        <KpiStripItem
          label={t("Margin per stop")}
          tone={signTone(marginPerStop)}
          value={formatCurrency(marginPerStop)}
          sub={
            rollup.suppressedCount > 0
              ? `${rollup.suppressedCount} lost to no notice`
              : `${rollup.disputeCount} disputed`
          }
        />
      </KpiStrip>

      <div className="border-border bg-card flex flex-col gap-5 rounded-lg border px-4 py-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1">
          <div className="max-w-xl">
            <ShareBreakdown segments={segments} />
          </div>

          {overrun ? (
            <p className="text-danger-foreground mt-3 text-xs">
              {t(
                "Driver detention pay exceeded what was billed — the free-time concessions granted to customers are wider than the driver contract allows for.",
              )}
            </p>
          ) : null}
        </div>

        <div className="flex shrink-0 items-center gap-4 sm:flex-col sm:gap-2">
          <RingGauge
            value={Math.max(retention, 0)}
            size={96}
            strokeWidth={7}
            tone={retentionTone(retention)}
            aria-label={t("Share of billed detention retained after driver pay")}
          >
            <div className="text-center">
              <p className="text-lg leading-none font-semibold tabular-nums">
                {Math.round(retention * 100)}%
              </p>
              <p className="text-muted-foreground mt-0.5 text-xs">retained</p>
            </div>
          </RingGauge>
          <p className="text-muted-foreground max-w-[9rem] text-xs leading-snug sm:text-center">
            {t("of every billed detention dollar survives driver pay")}
          </p>
        </div>
      </div>
    </section>
  );
}
