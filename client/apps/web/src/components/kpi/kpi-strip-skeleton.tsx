import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { KpiStrip, KpiStripItem } from "./kpi-strip";

type KpiStripSkeletonProps = {
  count: number;
  size?: "md" | "lg";
  sub?: boolean;
  minItemWidth?: string;
  bare?: boolean;
  className?: string;
};

export function KpiStripSkeleton({
  count,
  size = "md",
  sub = true,
  minItemWidth,
  bare,
  className,
}: KpiStripSkeletonProps) {
  return (
    <KpiStrip minItemWidth={minItemWidth} bare={bare} className={className}>
      {Array.from({ length: count }, (_, index) => (
        <KpiStripItem
          key={index}
          size={size}
          label={<Skeleton className="h-3.5 w-20" />}
          value={<Skeleton className={size === "lg" ? "h-7 w-24" : "h-6 w-20"} />}
          sub={sub ? <Skeleton className="h-3.5 w-28" /> : undefined}
        />
      ))}
    </KpiStrip>
  );
}
