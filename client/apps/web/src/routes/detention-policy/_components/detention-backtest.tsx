import { BAR_TRANSITION, DivergingBar } from "@/components/detention/detention-charts";
import { pricingFingerprint } from "@/lib/detention-policy";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Card } from "@trenova/shared/components/ui/card";
import { Label } from "@trenova/shared/components/ui/label";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { deltaToneClass, formatSignedCurrency } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BacktestBucket,
  BacktestResult,
  DetentionPolicy,
} from "@trenova/shared/types/detention";
import { useMutation } from "@tanstack/react-query";
import { HistoryIcon, RotateCwIcon, TriangleAlertIcon } from "lucide-react";
import { m } from "motion/react";
import { useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const WINDOW_OPTIONS = [
  { value: "30", label: "30d", days: 30 },
  { value: "90", label: "90d", days: 90 },
  { value: "180", label: "180d", days: 180 },
];

const DIMENSIONS = [
  { value: "customer", label: "By customer" },
  { value: "facility", label: "By facility" },
];

const DRIVER_PAY_RATE = "25.00";

function StatCell({
  label,
  value,
  className,
}: {
  label: string;
  value: string;
  className?: string;
}) {
  return (
    <div className="px-3 py-2.5">
      <p className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">{label}</p>
      <p className={cn("mt-0.5 text-sm font-medium tabular-nums", className)}>{value}</p>
    </div>
  );
}

function RiskChip({ tone, children }: { tone: "warn" | "bad" | "neutral"; children: string }) {
  return (
    <span
      className={cn(
        "text-2xs inline-flex items-center rounded-md px-1.5 py-0.5 font-medium",
        tone === "neutral" && "bg-muted text-muted-foreground",
        tone === "warn" && "bg-amber-500/15 text-amber-700 dark:text-amber-400",
        tone === "bad" && "bg-red-500/15 text-red-700 dark:text-red-400",
      )}
    >
      {children}
    </span>
  );
}

/**
 * Baseline against proposal on a shared scale. Two bars beat two numbers here:
 * the question an operator is answering is "how much does this move", and a
 * ratio is read faster than a subtraction.
 */
function RevenueComparison({ result }: { result: BacktestResult }) {
  const scale = Math.max(result.proposedRevenue, result.baselineRevenue, 1);

  const rows = [
    {
      key: "proposed",
      label: "Proposed",
      amount: result.proposedRevenue,
      className: "bg-foreground",
    },
    {
      key: "baseline",
      label: "Billed today",
      amount: result.baselineRevenue,
      className: "bg-muted-foreground/40",
    },
  ];

  return (
    <div className="flex flex-col gap-2">
      {rows.map((row) => (
        <div key={row.key} className="flex items-center gap-3">
          <span className="text-2xs text-muted-foreground w-20 shrink-0">{row.label}</span>
          <div className="bg-muted h-1.5 min-w-0 flex-1 overflow-hidden rounded-full">
            <m.div
              className={cn("h-full rounded-full", row.className)}
              initial={{ width: 0 }}
              animate={{ width: `${(row.amount / scale) * 100}%` }}
              transition={BAR_TRANSITION}
            />
          </div>
          <span className="w-24 shrink-0 text-right text-xs font-medium tabular-nums">
            {formatCurrency(row.amount)}
          </span>
        </div>
      ))}
    </div>
  );
}

function MoverRow({ bucket, scale }: { bucket: BacktestBucket; scale: number }) {
  return (
    <div className="flex items-center gap-3 px-3 py-2">
      <div className="min-w-0 flex-1">
        <p className="truncate text-xs">{bucket.label || bucket.key}</p>
        <p className="text-2xs text-muted-foreground tabular-nums">
          {bucket.stopCount} stops · {bucket.billableCount} billable
        </p>
      </div>

      <DivergingBar value={bucket.delta} scale={scale} />

      <span
        className={cn(
          "w-20 shrink-0 text-right text-xs font-medium tabular-nums",
          deltaToneClass(bucket.delta),
        )}
      >
        {formatSignedCurrency(bucket.delta)}
      </span>
    </div>
  );
}

function ResultView({ result, stale }: { result: BacktestResult; stale: boolean }) {
  const [dimension, setDimension] = useState(DIMENSIONS[0].value);

  const buckets = dimension === "customer" ? result.byCustomer : result.byFacility;
  const movers = useMemo(
    () => [...buckets].sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta)).slice(0, 6),
    [buckets],
  );
  const scale = movers.reduce((max, bucket) => Math.max(max, Math.abs(bucket.delta)), 0);

  const riskChips = [
    result.stopsForfeited > 0 && {
      tone: "warn" as const,
      text: `${result.stopsForfeited} forfeited to late arrival`,
    },
    result.stopsSuppressed > 0 && {
      tone: "bad" as const,
      text: `${result.stopsSuppressed} suppressed, no notice`,
    },
    result.negativeMarginStops > 0 && {
      tone: "bad" as const,
      text: `${result.negativeMarginStops} negative margin`,
    },
    result.truncated && { tone: "neutral" as const, text: "Sample truncated" },
  ].filter((chip): chip is { tone: "warn" | "bad" | "neutral"; text: string } => Boolean(chip));

  return (
    <m.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
      className={cn("flex flex-col gap-3", stale && "opacity-60")}
    >
      <Card className="gap-0 overflow-hidden rounded-lg p-0">
        <div className="px-4 py-3.5">
          <div className="flex items-end justify-between gap-3">
            <div className="min-w-0">
              <p className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
                Revenue change
              </p>
              <p
                className={cn(
                  "mt-1 text-3xl leading-none font-semibold tracking-tight tabular-nums",
                  deltaToneClass(result.revenueDelta),
                )}
              >
                {formatSignedCurrency(result.revenueDelta)}
              </p>
            </div>
            <p className="text-2xs text-muted-foreground shrink-0 text-right tabular-nums">
              {result.stopsMatched} of {result.stopsEvaluated} stops matched
              <br />
              {result.stopsBillable} would bill
            </p>
          </div>

          <div className="mt-3.5">
            <RevenueComparison result={result} />
          </div>

          {riskChips.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-1.5">
              {riskChips.map((chip) => (
                <RiskChip key={chip.text} tone={chip.tone}>
                  {chip.text}
                </RiskChip>
              ))}
            </div>
          )}
        </div>

        <div className="divide-border border-border grid grid-cols-3 divide-x border-t">
          <StatCell label="Driver pay" value={formatCurrency(result.proposedDriverPay)} />
          <StatCell
            label="Net margin"
            value={formatCurrency(result.proposedNetMargin)}
            className={deltaToneClass(result.proposedNetMargin)}
          />
          <StatCell
            label="Avg per billed stop"
            value={formatCurrency(
              result.stopsBillable > 0 ? result.proposedRevenue / result.stopsBillable : 0,
            )}
          />
        </div>
      </Card>

      <Card className="gap-0 overflow-hidden rounded-lg p-0">
        <div className="border-border flex items-center justify-between gap-3 border-b px-3 py-2">
          <p className="text-xs font-medium">Biggest movers</p>
          <SegmentedControl
            items={DIMENSIONS}
            value={dimension}
            onValueChange={setDimension}
            aria-label="Group movers by"
          />
        </div>

        {movers.length === 0 ? (
          <p className="text-muted-foreground px-3 py-6 text-center text-xs">
            No stops matched this policy in the selected window.
          </p>
        ) : (
          <div className="divide-border divide-y">
            {movers.map((bucket) => (
              <MoverRow key={bucket.key} bucket={bucket} scale={scale} />
            ))}
          </div>
        )}
      </Card>
    </m.div>
  );
}

/**
 * Backtest panel.
 *
 * Runs the proposed terms against real dwell history so a change can be argued
 * with a number rather than an intuition. The notice-compliance toggle switches
 * between "what are these terms worth" and "what would we actually have
 * collected given the notices we sent".
 */
export function DetentionBacktest() {
  const { control } = useFormContext<DetentionPolicy>();
  const values = useWatch({ control });
  const policy = values as DetentionPolicy;

  const [windowValue, setWindowValue] = useState(WINDOW_OPTIONS[1].value);
  const [assumeCompliance, setAssumeCompliance] = useState(true);
  const [ranWith, setRanWith] = useState<string | null>(null);

  const fingerprint = useMemo(() => pricingFingerprint(policy ?? {}), [policy]);

  const mutation = useMutation({
    mutationFn: async () => {
      const days = WINDOW_OPTIONS.find((option) => option.value === windowValue)?.days ?? 90;
      const now = Math.floor(Date.now() / 1000);

      return apiService.detentionAnalyticsService.backtest(policy, {
        from: now - days * 86400,
        to: now,
        assumeNoticeCompliance: assumeCompliance,
        driverPayRate: DRIVER_PAY_RATE,
      });
    },
    onSuccess: () => setRanWith(fingerprint),
  });

  const missing = [
    !policy?.name && "Name",
    !policy?.code && "Code",
    !policy?.accessorialChargeId && "Accessorial charge",
  ].filter((label): label is string => Boolean(label));
  const ready = missing.length === 0;
  const stale = Boolean(mutation.data) && ranWith !== fingerprint;

  const run = () => {
    if (ready && !mutation.isPending) {
      mutation.mutate();
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="min-w-0">
        <h3 className="text-2xs text-muted-foreground font-semibold tracking-wide uppercase">
          Backtest
        </h3>
        <p className="text-muted-foreground mt-1 text-xs">
          Re-price settled history under these terms. The engine is the same one that bills live
          shipments, so the projection is exact rather than estimated.
        </p>
      </div>

      <Card className="gap-0 overflow-hidden rounded-lg p-0">
        <div className="flex items-center justify-between gap-3 px-3 py-2.5">
          <SegmentedControl
            items={WINDOW_OPTIONS}
            value={windowValue}
            onValueChange={setWindowValue}
            aria-label="Backtest window"
          />
          <Button type="button" size="sm" className="h-7" disabled={!ready} onClick={run}>
            {mutation.isPending ? (
              <RotateCwIcon className="mr-1.5 size-3.5 animate-spin" />
            ) : (
              <HistoryIcon className="mr-1.5 size-3.5" />
            )}
            {mutation.isPending ? "Running…" : mutation.data ? "Re-run" : "Run backtest"}
          </Button>
        </div>

        <div className="border-border flex items-start justify-between gap-3 border-t px-3 py-2.5">
          <div className="min-w-0">
            <Label htmlFor="assume-compliance" className="text-xs font-medium">
              Assume notices were sent on time
            </Label>
            <p className="text-2xs text-muted-foreground mt-0.5">
              {assumeCompliance
                ? "Measures what the terms are worth if the notice process works."
                : "Replays the notices actually sent, showing what process failures cost."}
            </p>
          </div>
          <Switch
            id="assume-compliance"
            checked={assumeCompliance}
            onCheckedChange={setAssumeCompliance}
          />
        </div>
      </Card>

      {!ready && (
        <div className="border-border flex flex-wrap items-center gap-1.5 rounded-lg border border-dashed px-3 py-2.5">
          <span className="text-muted-foreground text-xs">Finish on the Terms tab first:</span>
          {missing.map((label) => (
            <span
              key={label}
              className="bg-muted text-2xs text-muted-foreground rounded-md px-1.5 py-0.5 font-medium"
            >
              {label}
            </span>
          ))}
        </div>
      )}

      {stale && (
        <button
          type="button"
          onClick={run}
          className="flex items-center gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-left transition-colors hover:bg-amber-500/15"
        >
          <TriangleAlertIcon className="size-3.5 shrink-0 text-amber-600 dark:text-amber-400" />
          <span className="text-xs">
            The terms changed since this run — re-run to see what they are worth.
          </span>
        </button>
      )}

      {mutation.isPending && !mutation.data && (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-[168px] w-full rounded-lg" />
          <Skeleton className="h-[196px] w-full rounded-lg" />
        </div>
      )}

      {mutation.isError && (
        <div className="border-border flex items-start gap-2.5 rounded-lg border px-3 py-2.5">
          <TriangleAlertIcon className="mt-px size-4 shrink-0 text-amber-500" />
          <div className="min-w-0">
            <p className="text-xs font-medium">The backtest could not run</p>
            <p className="text-muted-foreground mt-0.5 text-xs">
              {mutation.error instanceof Error
                ? mutation.error.message
                : "Resolve the validation errors on the Terms tab and try again."}
            </p>
          </div>
        </div>
      )}

      {mutation.data && !mutation.isPending && <ResultView result={mutation.data} stale={stale} />}

      {!mutation.data && !mutation.isPending && !mutation.isError && ready && (
        <div className="border-border flex flex-col items-center gap-2.5 rounded-lg border border-dashed py-10 text-center">
          <HistoryIcon className="text-muted-foreground/60 size-5" />
          <p className="text-muted-foreground max-w-[18rem] text-xs">
            Run the backtest to see what these terms would have billed over the last{" "}
            {WINDOW_OPTIONS.find((option) => option.value === windowValue)?.days} days.
          </p>
        </div>
      )}
    </div>
  );
}
