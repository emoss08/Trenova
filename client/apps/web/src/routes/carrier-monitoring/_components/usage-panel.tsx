import { useT } from "@trenova/shared/i18n/use-t";
import { CarrierIntelUsageSummaryView } from "@/components/carrier-intelligence/usage-summary";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { formatUsageMonth, recentUsageMonths } from "@/lib/carrier-intel-usage";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { RefreshCwIcon, SettingsIcon } from "lucide-react";
import { parseAsInteger, useQueryState } from "nuqs";
import { useMemo } from "react";
import { Link } from "react-router";

const USAGE_MONTHS_SHOWN = 12;

export function UsagePanel() {
  const t = useT();
  const now = useNowSeconds();
  const [month, setMonth] = useQueryState("month", parseAsInteger);

  const monthStarts = useMemo(() => recentUsageMonths(now, USAGE_MONTHS_SHOWN), [now]);
  const currentMonth = monthStarts[0];
  const selectedMonth = month !== null && monthStarts.includes(month) ? month : currentMonth;
  const isCurrentMonth = selectedMonth === currentMonth;

  const monthItems = useMemo(
    () =>
      monthStarts.map((start, index) => ({
        value: String(start),
        label:
          index === 0 ? t("{0} (month to date)", formatUsageMonth(start)) : formatUsageMonth(start),
      })),
    [monthStarts, t],
  );

  const usageQuery = useQuery({
    ...queries.carrierIntelSettings.usage(isCurrentMonth ? undefined : selectedMonth),
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-col">
          <h3 className="text-sm font-medium">{t("Provider spend")}</h3>
          <p className="text-muted-foreground text-xs">
            {t(
              "Estimated charges from the carrier intelligence provider, grouped by endpoint and day in UTC.",
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Select
            items={monthItems}
            value={String(selectedMonth)}
            onValueChange={(value) => {
              const next = Number(value);
              void setMonth(Number.isFinite(next) && next !== currentMonth ? next : null);
            }}
          >
            <SelectTrigger size="sm" className="min-w-52" aria-label={t("Month")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {monthItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            aria-label={t("Refresh usage")}
            isLoading={usageQuery.isRefetching}
            onClick={() => void usageQuery.refetch()}
          >
            <RefreshCwIcon />
          </Button>
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
          >
            <SettingsIcon className="size-3.5" />
            {t("Spend limits")}
          </Button>
        </div>
      </div>
      {usageQuery.isPending ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-48 w-full" />
        </div>
      ) : usageQuery.isError ? (
        <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
          <span>
            {t(
              "Usage could not be loaded. {0}",
              graphQLErrorMessage(usageQuery.error, t("Try again in a moment.")),
            )}
          </span>
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={() => void usageQuery.refetch()}
          >
            <RefreshCwIcon />
            {t("Retry")}
          </Button>
        </div>
      ) : (
        <CarrierIntelUsageSummaryView
          usage={usageQuery.data}
          emptyMessage={
            isCurrentMonth ? t("No provider calls this month.") : t("No provider calls that month.")
          }
        />
      )}
    </div>
  );
}
