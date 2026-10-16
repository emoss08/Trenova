import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { SkeletonRegion } from "@/components/settings-form-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

const KPI_CELLS = 6;
const RECOMMENDATION_ROWS = 5;
const MEMBER_ROWS = 4;
const ACTIVITY_ROWS = 5;
const LEGEND_WIDTHS = ["w-16", "w-24", "w-20"] as const;

/**
 * The briefing's two rows (brief and ring, then recommendations and the side column) share
 * these columns, so the side column lines up from the ring to the last sign-in.
 */
export const BRIEFING_COLUMNS =
  "grid items-start gap-x-12 gap-y-8 xl:grid-cols-[minmax(0,1fr)_20rem]";

function SectionSkeleton({ titleWidth, children }: { titleWidth: string; children: ReactNode }) {
  return (
    <div className="flex flex-col">
      <div className="flex min-h-8 items-center pb-1.5">
        <Skeleton className={cn("h-3.5", titleWidth)} />
      </div>
      <div className="border-border-subtle divide-border-subtle divide-y border-y">{children}</div>
    </div>
  );
}

function AsideRowSkeleton() {
  return (
    <div className="flex items-center gap-3 py-2">
      <Skeleton className="size-6 shrink-0 rounded-md" />
      <div className="flex flex-1 flex-col gap-1.5">
        <Skeleton className="h-3.5 w-2/3" />
        <Skeleton className="h-3 w-1/3" />
      </div>
      <Skeleton className="h-3 w-6" />
    </div>
  );
}

/**
 * The briefing's own layout in shimmer: the brief at its reading size beside the ring and its
 * legend, the figures between their hairlines, then the recommendations beside the members
 * without two-factor and the latest activity.
 */
export function SecurityBriefingSkeleton() {
  const t = useT();

  return (
    <SkeletonRegion label={t("Loading security briefing")} className="flex flex-col gap-8">
      <div className={BRIEFING_COLUMNS}>
        <div className="flex min-w-0 max-w-full flex-col gap-2">
          <div className="flex flex-col gap-2 py-1 text-lg leading-relaxed">
            <Skeleton className="h-4 w-11/12" />
            <Skeleton className="h-4 w-4/5" />
            <Skeleton className="h-4 w-2/5" />
          </div>
          <Skeleton className="h-2.5 w-36" />
        </div>
        <div className="flex items-center gap-5">
          <Skeleton className="size-19 shrink-0 rounded-full" />
          <div className="flex flex-col gap-2">
            {LEGEND_WIDTHS.map((width) => (
              <Skeleton key={width} className={cn("h-3", width)} />
            ))}
          </div>
        </div>
      </div>
      <KpiStripSkeleton bare count={KPI_CELLS} minItemWidth="9rem" />
      <div className={BRIEFING_COLUMNS}>
        <SectionSkeleton titleWidth="w-32">
          {Array.from({ length: RECOMMENDATION_ROWS }, (_, row) => (
            <div key={row} className="flex gap-3 py-3">
              <Skeleton className="size-4 shrink-0 rounded-full" />
              <div className="flex flex-1 flex-col gap-1.5">
                <Skeleton className="h-3.5 w-2/3" />
                <Skeleton className="h-3 w-full max-w-prose" />
              </div>
            </div>
          ))}
        </SectionSkeleton>
        <div className="flex flex-col gap-8">
          <SectionSkeleton titleWidth="w-36">
            {Array.from({ length: MEMBER_ROWS }, (_, row) => (
              <AsideRowSkeleton key={row} />
            ))}
          </SectionSkeleton>
          <SectionSkeleton titleWidth="w-28">
            {Array.from({ length: ACTIVITY_ROWS }, (_, row) => (
              <AsideRowSkeleton key={row} />
            ))}
          </SectionSkeleton>
        </div>
      </div>
    </SkeletonRegion>
  );
}
