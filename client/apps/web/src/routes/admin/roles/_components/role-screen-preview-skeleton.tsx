import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The screen preview while its code loads: the screen picker and the prompt under it. */
export function RoleScreenPreviewSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the screen preview")}
      className="flex flex-col gap-4"
    >
      <div aria-hidden className="flex flex-col gap-4">
        <div className="flex flex-col gap-1.5">
          <Skeleton className="h-3 w-14" />
          <Skeleton className="h-8 w-full rounded-md" />
        </div>
        <Skeleton className="h-3 w-3/5" />
      </div>
    </div>
  );
}
