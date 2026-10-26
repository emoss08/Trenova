import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { AccountingBriefingSkeleton } from "./accounting-briefing-skeleton";
import { AR_KPI_COUNT, AR_KPI_MIN_WIDTH } from "./ar-kpi-row";
import { ChartBodySkeleton, RowsBodySkeleton } from "./dashboard-panel";

/** A panel's frame while the page decides what to draw: its header bar and body. */
function PanelSkeleton({
  titleWidth,
  className,
  children,
}: {
  titleWidth: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("bg-card flex flex-col overflow-hidden rounded-lg border", className)}>
      <div className="border-border flex min-h-9 items-center border-b px-3 py-1.5" aria-hidden>
        <Skeleton className={cn("h-3", titleWidth)} />
      </div>
      {children}
    </div>
  );
}

/**
 * The dashboard before its KPIs say whether the book has ever invoiced: the brief, the KPI
 * strip and every panel in the place it lands. The panels' own reads wait for that answer, so
 * a book with nothing on it never asks for a trend it will not draw.
 */
export function AccountingDashboardSkeleton() {
  const t = useT();

  return (
    <div className="contents" aria-busy aria-label={t("Loading the accounting dashboard")}>
      <AccountingBriefingSkeleton />
      <KpiStripSkeleton count={AR_KPI_COUNT} size="lg" minItemWidth={AR_KPI_MIN_WIDTH} />
      <div className="grid gap-4 xl:grid-cols-2">
        <PanelSkeleton titleWidth="w-40">
          <ChartBodySkeleton label={t("Loading the DSO trend")} />
        </PanelSkeleton>
        <PanelSkeleton titleWidth="w-36">
          <ChartBodySkeleton label={t("Loading the cash-flow forecast")} />
        </PanelSkeleton>
      </div>
      <div className="grid gap-4 xl:grid-cols-2">
        <PanelSkeleton titleWidth="w-28">
          <ChartBodySkeleton label={t("Loading the aging trend")} />
        </PanelSkeleton>
        <PanelSkeleton titleWidth="w-44">
          <ChartBodySkeleton label={t("Loading collections performance")} />
        </PanelSkeleton>
      </div>
      <div className="grid gap-4 xl:grid-cols-5">
        <PanelSkeleton titleWidth="w-36" className="xl:col-span-3">
          <RowsBodySkeleton label={t("Loading the collections worklist")} />
        </PanelSkeleton>
        <PanelSkeleton titleWidth="w-44" className="xl:col-span-2">
          <RowsBodySkeleton label={t("Loading the top overdue customers")} lead="rank" />
        </PanelSkeleton>
      </div>
    </div>
  );
}
