import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

function GrantSkeletonBody() {
  return (
    <div className="bg-card border-border flex flex-col gap-2 rounded-lg border p-4" aria-hidden>
      <div className="flex items-center gap-2">
        <Skeleton className="h-4 w-52" />
        <Skeleton className="h-5 w-14" />
        <Skeleton className="h-5 w-20" />
      </div>
      <Skeleton className="h-3 w-72" />
      <Skeleton className="h-3 w-60" />
    </div>
  );
}

export function SupportGrantsSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-3" aria-busy aria-label={t("Loading support access")}>
      <GrantSkeletonBody />
      <GrantSkeletonBody />
      <GrantSkeletonBody />
    </div>
  );
}

export function SupportActionsSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-2 py-2" aria-busy aria-label={t("Loading what support did")}>
      <Skeleton className="h-3 w-80" />
      <Skeleton className="h-3 w-64" />
      <Skeleton className="h-3 w-72" />
    </div>
  );
}
