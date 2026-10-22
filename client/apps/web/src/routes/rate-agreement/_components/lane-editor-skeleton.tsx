import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const LANES = 2;

/**
 * The lane grid while the rules module that keys and scores the lanes loads: each lane's card
 * with its label, key and specificity, the origin and destination side by side, and the
 * add-lane button.
 */
export function LaneEditorSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading the lanes…")} className="flex flex-col gap-4">
      {Array.from({ length: LANES }, (_, index) => (
        <div key={index} aria-hidden="true" className="flex flex-col gap-4 rounded-md border p-4">
          <div className="flex flex-col gap-1.5">
            <Skeleton className="h-3.5 w-32" />
            <div className="flex gap-1.5">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-24" />
            </div>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            <Skeleton className="h-24" />
            <Skeleton className="h-24" />
          </div>
        </div>
      ))}
      <Skeleton aria-hidden="true" className="h-8 w-full" />
    </div>
  );
}
