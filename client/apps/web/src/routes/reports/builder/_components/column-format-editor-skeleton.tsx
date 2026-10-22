import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * A column's formatting editor while the rules module that resolves its display loads: the
 * heading with its sample, and two rows of fields.
 */
export function ColumnFormatEditorSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the formatting…")}
      className="border-border flex flex-col gap-2 rounded-md border border-dashed p-2"
    >
      <div aria-hidden="true" className="flex items-center gap-2">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-4 w-20" />
      </div>
      <div aria-hidden="true" className="grid grid-cols-2 gap-2">
        <Skeleton className="h-7" />
        <Skeleton className="h-7" />
        <Skeleton className="h-7" />
        <Skeleton className="h-7" />
      </div>
    </div>
  );
}
