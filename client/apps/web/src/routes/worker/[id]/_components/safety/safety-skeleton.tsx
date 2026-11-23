import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ShimmerLine, ShimmerPanel } from "../tab-skeleton-parts";

/** The safety record's own layout in shimmer: the scorecard band, then events beside the stair. */
export function SafetySkeleton() {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading the safety record")} className="flex flex-col gap-4">
      <div aria-hidden className="contents">
        <div className="grid overflow-hidden rounded-lg border lg:grid-cols-[17rem_minmax(0,1fr)_minmax(0,25rem)]">
          <div className="flex flex-col gap-2.5 border-b px-4 py-3.5 lg:border-r lg:border-b-0">
            <ShimmerLine className="w-24" />
            <ShimmerLine className="h-7 w-32" />
            <ShimmerLine className="w-28" />
            <div className="mt-3 flex gap-2">
              <ShimmerLine className="h-7 w-28 rounded-md" />
              <ShimmerLine className="h-7 w-24 rounded-md" />
            </div>
          </div>
          <div className="flex flex-col gap-3 border-b px-5 py-3.5 lg:border-b-0">
            <div className="flex justify-between">
              <ShimmerLine className="w-24" />
              <ShimmerLine className="h-7 w-8" />
            </div>
            <ShimmerLine className="h-2.5 w-full rounded-xs" />
            <ShimmerLine className="w-48" />
          </div>
          <div className="grid grid-cols-3 lg:border-l">
            {Array.from({ length: 6 }, (_, index) => (
              <div key={index} className="flex flex-col gap-1.5 border-r border-b px-3.5 py-2.5">
                <ShimmerLine className="w-16" />
                <ShimmerLine className="w-10" />
              </div>
            ))}
          </div>
        </div>
        <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
          <ShimmerPanel rows={5} />
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-3 rounded-lg border p-3">
              <ShimmerLine className="w-24" />
              <div className="grid grid-cols-6 items-end gap-1">
                {["h-3.5", "h-5.5", "h-7.5", "h-9.5", "h-11.5", "h-13.5"].map((height) => (
                  <span key={height} className={cn("block ui-shimmer rounded-xs", height)} />
                ))}
              </div>
              <ShimmerLine className="w-40" />
            </div>
            <ShimmerPanel rows={2} />
            <ShimmerPanel rows={1} />
          </div>
        </div>
      </div>
    </div>
  );
}
