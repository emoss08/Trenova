import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The remittance page as it loads: the totals strip, then the two line panels. */
export function RemittancePageSkeleton() {
  const t = useT();
  return (
    <div className="flex flex-col gap-4" aria-busy="true" aria-label={t("Loading remittance")}>
      <Skeleton className="h-16 w-full" />
      <div className="grid gap-4 xl:grid-cols-2">
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    </div>
  );
}
