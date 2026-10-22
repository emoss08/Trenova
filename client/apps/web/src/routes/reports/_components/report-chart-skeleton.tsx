import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/** A report chart while the rules module that labels its values loads: the plot's area. */
export function ReportChartSkeleton({ className }: { className?: string }) {
  const t = useT();
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the chart…")}
      className={cn("relative flex h-full min-h-32 w-full", className)}
    >
      <Skeleton aria-hidden="true" className="h-full w-full" />
    </div>
  );
}
