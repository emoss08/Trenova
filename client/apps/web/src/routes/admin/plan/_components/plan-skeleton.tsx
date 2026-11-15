import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

function PanelSkeleton({ rows }: { rows: number }) {
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border bg-card">
      <div className="flex min-h-9 items-center border-b border-border px-3 py-1.5">
        <Skeleton className="h-4 w-40" />
      </div>
      <div className="flex flex-col divide-y divide-border">
        {Array.from({ length: rows }, (_, index) => (
          <div key={index} className="flex flex-col gap-1.5 px-3 py-3">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-3 w-72" />
          </div>
        ))}
      </div>
    </div>
  );
}

/** The plan page as it lands: what every plan includes, the add-ons on and off it, and changes. */
export function PlanSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading your plan")}>
      <div aria-hidden className="flex flex-col gap-4">
        <PanelSkeleton rows={1} />
        <PanelSkeleton rows={2} />
        <PanelSkeleton rows={3} />
        <PanelSkeleton rows={2} />
      </div>
    </div>
  );
}
