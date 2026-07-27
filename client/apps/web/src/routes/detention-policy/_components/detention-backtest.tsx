import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Label } from "@trenova/shared/components/ui/label";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BacktestBucket,
  BacktestResult,
  DetentionPolicy,
} from "@trenova/shared/types/detention";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { useWatch, type Control } from "react-hook-form";

const WINDOW_OPTIONS = [
  { label: "30 days", days: 30 },
  { label: "90 days", days: 90 },
  { label: "180 days", days: 180 },
];

function DeltaBadge({ delta }: { delta: number }) {
  if (delta === 0) {
    return <Badge variant="outline">No change</Badge>;
  }

  const positive = delta > 0;

  return (
    <Badge
      className={cn(
        "border-none tabular-nums",
        positive
          ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400"
          : "bg-red-500/15 text-red-700 dark:text-red-400",
      )}
    >
      {positive ? "+" : ""}
      {formatCurrency(delta)}
    </Badge>
  );
}

function BucketList({ title, buckets }: { title: string; buckets: BacktestBucket[] }) {
  if (buckets.length === 0) {
    return null;
  }

  return (
    <div>
      <p className="mb-2 text-xs font-medium">{title}</p>
      <div className="space-y-1">
        {buckets.slice(0, 5).map((bucket) => (
          <div
            key={bucket.key}
            className="flex items-center justify-between gap-2 rounded-md border px-2 py-1.5"
          >
            <span className="truncate font-mono text-[11px]">{bucket.key}</span>
            <div className="flex shrink-0 items-center gap-2">
              <span className="text-muted-foreground text-[11px] tabular-nums">
                {bucket.stopCount} stops
              </span>
              <DeltaBadge delta={bucket.delta} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function ResultView({ result }: { result: BacktestResult }) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-2">
        <div className="rounded-md border p-3">
          <p className="text-muted-foreground text-xs">Projected revenue</p>
          <p className="text-xl font-semibold tabular-nums">
            {formatCurrency(result.proposedRevenue)}
          </p>
          <p className="text-muted-foreground text-[11px] tabular-nums">
            vs {formatCurrency(result.baselineRevenue)} actual
          </p>
        </div>
        <div className="rounded-md border p-3">
          <p className="text-muted-foreground text-xs">Change</p>
          <p
            className={cn(
              "text-xl font-semibold tabular-nums",
              result.revenueDelta > 0
                ? "text-emerald-600 dark:text-emerald-400"
                : result.revenueDelta < 0
                  ? "text-red-600 dark:text-red-400"
                  : undefined,
            )}
          >
            {result.revenueDelta > 0 ? "+" : ""}
            {formatCurrency(result.revenueDelta)}
          </p>
          <p className="text-muted-foreground text-[11px] tabular-nums">
            over {result.stopsMatched} matching stops
          </p>
        </div>
      </div>

      <div className="flex flex-wrap gap-1.5">
        <Badge variant="outline" className="text-[10px]">
          {result.stopsEvaluated} evaluated
        </Badge>
        <Badge variant="outline" className="text-[10px]">
          {result.stopsBillable} billable
        </Badge>
        {result.stopsForfeited > 0 && (
          <Badge className="border-none bg-amber-500/15 text-[10px] text-amber-700 dark:text-amber-400">
            {result.stopsForfeited} forfeited to late arrival
          </Badge>
        )}
        {result.stopsSuppressed > 0 && (
          <Badge className="border-none bg-red-500/15 text-[10px] text-red-700 dark:text-red-400">
            {result.stopsSuppressed} suppressed, no notice
          </Badge>
        )}
        {result.negativeMarginStops > 0 && (
          <Badge className="border-none bg-red-500/15 text-[10px] text-red-700 dark:text-red-400">
            {result.negativeMarginStops} negative margin
          </Badge>
        )}
        {result.truncated && (
          <Badge variant="outline" className="text-[10px]">
            Sample truncated
          </Badge>
        )}
      </div>

      <BucketList title="Biggest movers by customer" buckets={result.byCustomer} />
      <BucketList title="Biggest movers by facility" buckets={result.byFacility} />
    </div>
  );
}

type DetentionBacktestProps = {
  control: Control<DetentionPolicy>;
};

/**
 * Backtest panel.
 *
 * Runs the proposed terms against real dwell history so a change can be argued
 * with a number rather than an intuition. The notice-compliance toggle switches
 * between "what are these terms worth" and "what would we actually have
 * collected given the notices we sent".
 */
export function DetentionBacktest({ control }: DetentionBacktestProps) {
  const values = useWatch({ control });
  const policy = values as DetentionPolicy;

  const [days, setDays] = useState(90);
  const [assumeCompliance, setAssumeCompliance] = useState(true);

  const mutation = useMutation({
    mutationFn: async () => {
      const now = Math.floor(Date.now() / 1000);
      return apiService.detentionAnalyticsService.backtest(policy, {
        from: now - days * 86400,
        to: now,
        assumeNoticeCompliance: assumeCompliance,
        driverPayRate: "25.00",
      });
    },
  });

  const ready = Boolean(policy?.accessorialChargeId && policy?.code && policy?.name);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Backtest</CardTitle>
        <CardDescription>
          Re-price settled history under these terms. The engine is the same one
          that bills live shipments, so the projection is exact rather than
          estimated.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          {WINDOW_OPTIONS.map((option) => (
            <Button
              key={option.days}
              type="button"
              size="sm"
              variant={days === option.days ? "default" : "outline"}
              className="h-7 text-xs"
              onClick={() => setDays(option.days)}
            >
              {option.label}
            </Button>
          ))}
        </div>

        <div className="flex items-center gap-2">
          <Switch
            id="assume-compliance"
            checked={assumeCompliance}
            onCheckedChange={setAssumeCompliance}
          />
          <Label htmlFor="assume-compliance" className="text-xs">
            Assume notices were sent on time
          </Label>
        </div>
        <p className="text-muted-foreground text-[11px]">
          {assumeCompliance
            ? "Measures what the terms are worth if the notice process works."
            : "Replays the notices actually sent, showing what process failures cost."}
        </p>

        <Button
          type="button"
          size="sm"
          disabled={!ready || mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          {mutation.isPending ? "Running…" : "Run backtest"}
        </Button>

        {!ready && (
          <p className="text-muted-foreground text-xs">
            Name the policy and choose an accessorial charge first.
          </p>
        )}

        {mutation.isPending && <Skeleton className="h-40 w-full" />}

        {mutation.isError && (
          <p className="text-muted-foreground text-xs">
            {mutation.error instanceof Error
              ? mutation.error.message
              : "The backtest could not run."}
          </p>
        )}

        {mutation.data && <ResultView result={mutation.data} />}
      </CardContent>
    </Card>
  );
}
