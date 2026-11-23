import type { WorkerViewOf } from "@/routes/worker/use-worker-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { ShimmerLine, ShimmerMainAside, ShimmerPanel } from "../tab-skeleton-parts";

export function PaySkeleton({ view = "pay-profile" }: { view?: WorkerViewOf<"pay"> }) {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={view === "pay-profile" ? t("Loading the pay profile") : t("Loading the benefits")}
      className="flex flex-col gap-4"
    >
      <div aria-hidden className="contents">
        {view === "pay-profile" ? (
          <ShimmerMainAside main={2} aside={1} />
        ) : (
          <>
            <div className="flex flex-col gap-3 rounded-lg border px-4 pt-3.5 pb-4">
              <ShimmerLine className="w-44" />
              <ShimmerLine className="h-6 w-32" />
              <ShimmerLine className="mt-5 h-4 w-full rounded-xs" />
              <div className="flex gap-5">
                <ShimmerLine className="w-28" />
                <ShimmerLine className="w-36" />
                <ShimmerLine className="w-40" />
              </div>
            </div>
            <ShimmerPanel rows={3} />
          </>
        )}
      </div>
    </div>
  );
}
