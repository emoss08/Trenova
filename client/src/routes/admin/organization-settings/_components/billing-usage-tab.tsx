import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import type { BillingUsageSummary } from "@/types/platform-billing";
import { useQuery } from "@tanstack/react-query";
import {
  ActivityIcon,
  CheckCircle2Icon,
  CircleAlertIcon,
  GaugeIcon,
  KeyRoundIcon,
  RefreshCcwIcon,
  ShieldCheckIcon,
} from "lucide-react";
import type { ComponentType } from "react";

const numberFormatter = new Intl.NumberFormat();
const dateFormatter = new Intl.DateTimeFormat(undefined, {
  month: "short",
  day: "numeric",
  year: "numeric",
});

export function BillingUsageTab() {
  const summaryQuery = useQuery({
    ...queries.platformBilling.summary(),
  });

  const summary = summaryQuery.data;
  const allowedFeatures = summary?.features.filter((feature) => feature.allowed).length ?? 0;
  const deniedFeatures = summary?.features.filter((feature) => !feature.allowed).length ?? 0;
  const trackedMeters = summary?.usage.length ?? 0;

  if (summaryQuery.isLoading) {
    return <BillingSkeleton />;
  }

  if (summaryQuery.isError) {
    return (
      <Alert variant="destructive">
        <CircleAlertIcon className="size-4" />
        <AlertTitle>Unable to load billing status</AlertTitle>
        <AlertDescription>The subscription and usage summary could not be loaded.</AlertDescription>
      </Alert>
    );
  }

  if (!summary) {
    return null;
  }

  return (
    <div className="space-y-4">
      {!summary.active ? (
        <Alert variant="warning">
          <CircleAlertIcon className="size-4" />
          <AlertTitle>Access is not active</AlertTitle>
          <AlertDescription>{formatReason(summary.reason)}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-2.5 md:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          icon={ShieldCheckIcon}
          label="Access State"
          value={summary.active ? "Active" : "Blocked"}
          detail={formatReason(summary.reason)}
          tone={summary.active ? "active" : "inactive"}
        />
        <SummaryCard
          icon={KeyRoundIcon}
          label="Plan"
          value={summary.plan?.name ?? "Not assigned"}
          detail={summary.plan?.key ?? "No plan key"}
          tone="info"
        />
        <SummaryCard
          icon={CheckCircle2Icon}
          label="Features"
          value={numberFormatter.format(allowedFeatures)}
          detail={
            deniedFeatures > 0
              ? `${numberFormatter.format(deniedFeatures)} denied`
              : "All listed features enabled"
          }
          tone="teal"
        />
        <SummaryCard
          icon={GaugeIcon}
          label="Tracked Usage"
          value={numberFormatter.format(trackedMeters)}
          detail={formatPeriod(
            summary.subscription?.currentPeriodStart,
            summary.subscription?.currentPeriodEnd,
          )}
          tone="orange"
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(360px,0.65fr)]">
        <Card className="rounded-md">
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <CardTitle className="text-sm font-semibold">Usage This Period</CardTitle>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void summaryQuery.refetch()}
              disabled={summaryQuery.isFetching}
            >
              <RefreshCcwIcon className="size-3.5" />
              Refresh
            </Button>
          </CardHeader>
          <CardContent>
            {summary.usage.length > 0 ? (
              <div className="grid gap-2.5 lg:grid-cols-2">
                {summary.usage.map((usage) => (
                  <UsageMeter key={usage.meterKey} usage={usage} />
                ))}
              </div>
            ) : (
              <EmptyState
                icon={ActivityIcon}
                title="No metered usage"
                description="This tenant does not have any metered usage configured."
              />
            )}
          </CardContent>
        </Card>

        <Card className="rounded-md">
          <CardHeader>
            <CardTitle className="text-sm font-semibold">Enabled Features</CardTitle>
          </CardHeader>
          <CardContent>
            {summary.features.length > 0 ? (
              <div className="space-y-1.5">
                {summary.features.map((feature) => (
                  <div
                    key={feature.featureKey}
                    className="flex min-h-9 items-center justify-between gap-3 rounded-md border border-border bg-background px-2.5"
                  >
                    <span className="truncate text-sm font-medium">
                      {formatCatalogKey(feature.featureKey)}
                    </span>
                    <Badge variant={feature.allowed ? "active" : "inactive"}>
                      {feature.allowed ? "Enabled" : "Denied"}
                    </Badge>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={ShieldCheckIcon}
                title="No feature data"
                description="No entitlement records were returned for this tenant."
              />
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function SummaryCard({
  icon: Icon,
  label,
  value,
  detail,
  tone,
}: {
  icon: ComponentType<{ className?: string }>;
  label: string;
  value: string;
  detail: string;
  tone: "active" | "inactive" | "info" | "teal" | "orange";
}) {
  return (
    <Card className="rounded-md">
      <CardContent className="flex min-h-24 items-center gap-3 p-3">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted">
          <Icon className="size-4 text-muted-foreground" />
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-xs font-medium text-muted-foreground">{label}</p>
            <Badge variant={tone}>{tone === "inactive" ? "Needs action" : "Current"}</Badge>
          </div>
          <p className="mt-1 truncate text-lg font-semibold">{value}</p>
          <p className="mt-0.5 truncate text-xs text-muted-foreground">{detail}</p>
        </div>
      </CardContent>
    </Card>
  );
}

function UsageMeter({ usage }: { usage: BillingUsageSummary }) {
  const limited = usage.limit > 0;
  const percent = limited ? Math.min(Math.round((usage.used / usage.limit) * 100), 100) : 0;

  return (
    <div className="rounded-md border border-border bg-background p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold">{formatCatalogKey(usage.meterKey)}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {formatPeriod(usage.windowStart, usage.windowEnd)}
          </p>
        </div>
        <Badge variant={limited ? "info" : "active"}>{limited ? `${percent}%` : "Unlimited"}</Badge>
      </div>

      <div className="mt-4">
        {limited ? (
          <Progress
            value={usage.used}
            max={usage.limit}
            size="sm"
            variant={progressVariant(percent)}
          />
        ) : (
          <div className="h-1 rounded-full bg-primary/30" />
        )}
      </div>

      <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
        <UsageStat label="Used" value={formatUsageValue(usage.used, usage.unit)} />
        <UsageStat
          label="Limit"
          value={limited ? formatUsageValue(usage.limit, usage.unit) : "Unlimited"}
        />
        <UsageStat
          label="Remaining"
          value={limited ? formatUsageValue(usage.remaining, usage.unit) : "Unlimited"}
        />
      </div>
    </div>
  );
}

function UsageStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md bg-muted px-2 py-1.5">
      <p className="text-muted-foreground">{label}</p>
      <p className="truncate font-medium">{value}</p>
    </div>
  );
}

function EmptyState({
  icon: Icon,
  title,
  description,
}: {
  icon: ComponentType<{ className?: string }>;
  title: string;
  description: string;
}) {
  return (
    <div className="flex min-h-40 flex-col items-center justify-center rounded-md border border-dashed border-border bg-muted/30 p-6 text-center">
      <Icon className="size-5 text-muted-foreground" />
      <p className="mt-3 text-sm font-semibold">{title}</p>
      <p className="mt-1 max-w-sm text-sm text-muted-foreground">{description}</p>
    </div>
  );
}

function BillingSkeleton() {
  return (
    <div className="space-y-4">
      <div className="grid gap-2.5 md:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-24 rounded-md" />
        ))}
      </div>
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(360px,0.65fr)]">
        <Skeleton className="h-96 rounded-md" />
        <Skeleton className="h-96 rounded-md" />
      </div>
    </div>
  );
}

function formatCatalogKey(value: string) {
  return value
    .split(".")
    .filter(Boolean)
    .map((part) => part.replace(/_/g, " "))
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" / ");
}

function formatUsageValue(value: number, unit: string) {
  const suffix = unit ? ` ${unit}` : "";
  return `${numberFormatter.format(value)}${suffix}`;
}

function formatUnixDate(value?: number) {
  if (!value) return "Not set";
  return dateFormatter.format(new Date(value * 1000));
}

function formatPeriod(start?: number, end?: number) {
  if (!start || !end) return "No billing period";
  return `${formatUnixDate(start)} to ${formatUnixDate(end)}`;
}

function formatReason(reason?: string) {
  if (!reason) return "No status reason";
  return reason
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function progressVariant(percent: number) {
  if (percent >= 90) return "error";
  if (percent >= 75) return "warning";
  return "default";
}
