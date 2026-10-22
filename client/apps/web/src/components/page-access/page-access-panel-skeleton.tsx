import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The report's own layout while it loads: the badges, the needs and the records. */
export function PageAccessReportSkeleton() {
  const t = useT();

  return (
    <div aria-busy="true" aria-label={t("Loading page access")} className="flex flex-col gap-4">
      <div aria-hidden className="flex flex-col gap-4">
        <div className="flex gap-2">
          <Skeleton className="h-5 w-28" />
          <Skeleton className="h-5 w-36" />
        </div>
        <div className="flex flex-col gap-2">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-24 w-full rounded-md" />
        </div>
        <div className="flex flex-col gap-2">
          <Skeleton className="h-4 w-36" />
          <Skeleton className="h-40 w-full rounded-md" />
        </div>
      </div>
    </div>
  );
}
