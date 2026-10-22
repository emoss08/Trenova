import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { LaneKeyTableSkeleton } from "./lane-key-table-skeleton";

/**
 * The simulation tab while the rules module that names its lanes loads: the replay controls
 * over the table of lanes the last run read.
 */
export function SimulationPanelSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading the simulation…")} className="space-y-4">
      <div aria-hidden="true" className="flex flex-col gap-3 rounded-lg border p-3">
        <Skeleton className="h-3.5 w-48" />
        <Skeleton className="h-3 w-3/4" />
        <div className="flex items-end gap-3">
          <Skeleton className="h-8 w-32" />
          <Skeleton className="h-8 w-36" />
        </div>
      </div>
      <LaneKeyTableSkeleton />
    </div>
  );
}
