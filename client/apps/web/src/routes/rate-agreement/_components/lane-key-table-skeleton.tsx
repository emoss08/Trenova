import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = 4;

/**
 * A table of lanes named by their keys while the rules module that reads the keys loads: a
 * header and a few rows, each a lane name over its key.
 */
export function LaneKeyTableSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the lanes…")}
      className="flex flex-col gap-2 overflow-hidden rounded-lg border p-3"
    >
      <Skeleton aria-hidden="true" className="h-3 w-1/3" />
      {Array.from({ length: ROWS }, (_, index) => (
        <div key={index} aria-hidden="true" className="flex flex-col gap-1">
          <Skeleton className="h-3 w-2/5" />
          <Skeleton className="h-2.5 w-3/5" />
        </div>
      ))}
    </div>
  );
}
