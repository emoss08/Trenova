import { SkeletonRegion } from "@/components/settings-form-skeleton";
import { Card } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const GROUP_ROWS = [1, 2, 4] as const;

export function DataRetentionSkeleton() {
  const t = useT();

  return (
    <SkeletonRegion label={t("Loading data retention settings")} className="flex flex-col gap-4">
      <Card className="gap-0 py-0">
        <div className="border-border grid grid-cols-[minmax(0,1fr)_12rem_9rem_10rem] gap-4 border-b px-4 py-2.5">
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-3 w-10" />
        </div>
        {GROUP_ROWS.map((rows, group) => (
          <div key={group}>
            <div className="bg-sunken px-4 py-2">
              <Skeleton className="h-3 w-10" />
            </div>
            {Array.from({ length: rows }, (_, row) => (
              <div
                key={row}
                className="border-border-subtle grid grid-cols-[minmax(0,1fr)_12rem_9rem_10rem] gap-4 border-b px-4 py-3 last:border-b-0"
              >
                <div className="flex flex-col gap-1.5">
                  <Skeleton className="h-3.5 w-32" />
                  <Skeleton className="h-3 w-3/4" />
                </div>
                <Skeleton className="h-8 w-44" />
                <Skeleton className="mt-2 h-3 w-20" />
                <Skeleton className="mt-2 h-3 w-16" />
              </div>
            ))}
          </div>
        ))}
      </Card>
    </SkeletonRegion>
  );
}
