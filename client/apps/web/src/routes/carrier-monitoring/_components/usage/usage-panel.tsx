import { CarrierIntelUsageSummaryView } from "@/components/carrier-intelligence/usage-summary";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { formatUsageMonth, recentUsageMonths, shiftUsageMonth } from "@/lib/carrier-intel-usage";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { ChevronLeftIcon, ChevronRightIcon, RefreshCwIcon, SettingsIcon } from "lucide-react";
import { parseAsInteger, useQueryState } from "nuqs";
import { Link } from "react-router";

const USAGE_MONTHS_BACK = 24;

export function UsagePanel() {
  const t = useT();
  const now = useNowSeconds();
  const [month, setMonth] = useQueryState("month", parseAsInteger);

  const currentMonth = recentUsageMonths(now, 1)[0];
  const earliestMonth = shiftUsageMonth(currentMonth, -(USAGE_MONTHS_BACK - 1));
  const selectedMonth =
    month !== null && month >= earliestMonth && month <= currentMonth ? month : currentMonth;
  const isCurrentMonth = selectedMonth === currentMonth;

  const usageQuery = useQuery({
    ...queries.carrierIntelSettings.usage(isCurrentMonth ? undefined : selectedMonth),
  });

  const goToMonth = (next: number) => {
    void setMonth(next >= currentMonth ? null : next);
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-1" role="group" aria-label={t("Month")}>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t("Previous month")}
            disabled={selectedMonth <= earliestMonth}
            onClick={() => goToMonth(shiftUsageMonth(selectedMonth, -1))}
          >
            <ChevronLeftIcon className="size-4" />
          </Button>
          <span
            className="min-w-32 text-center text-sm font-medium tabular-nums"
            aria-live="polite"
          >
            {formatUsageMonth(selectedMonth)}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t("Next month")}
            disabled={isCurrentMonth}
            onClick={() => goToMonth(shiftUsageMonth(selectedMonth, 1))}
          >
            <ChevronRightIcon className="size-4" />
          </Button>
        </div>
        {isCurrentMonth ? (
          <span className="text-muted-foreground text-xs">{t("Month to date, UTC")}</span>
        ) : (
          <Button
            type="button"
            variant="ghost"
            className="text-muted-foreground h-8 text-xs"
            onClick={() => goToMonth(currentMonth)}
          >
            {t("Back to this month")}
          </Button>
        )}
        <div className="ml-auto flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t("Refresh usage")}
            isLoading={usageQuery.isRefetching}
            onClick={() => void usageQuery.refetch()}
          >
            <RefreshCwIcon className="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            className="h-8 text-xs"
            nativeButton={false}
            render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
          >
            <SettingsIcon className="size-3.5" />
            {t("Spend limits")}
          </Button>
        </div>
      </div>
      {usageQuery.isPending ? (
        <div className="flex flex-col gap-4">
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      ) : usageQuery.isError ? (
        <div className="flex flex-col items-center gap-3 rounded-lg border px-6 py-10 text-center">
          <p className="text-sm font-medium">{t("Usage could not be loaded")}</p>
          <p className="text-muted-foreground text-xs">
            {graphQLErrorMessage(usageQuery.error, t("Try again in a moment."))}
          </p>
          <Button
            type="button"
            variant="outline"
            className="h-8 text-xs"
            onClick={() => void usageQuery.refetch()}
          >
            <RefreshCwIcon className="size-3.5" />
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
