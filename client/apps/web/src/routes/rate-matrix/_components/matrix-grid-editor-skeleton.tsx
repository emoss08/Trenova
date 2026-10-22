import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const COLUMNS = 5;
const ROWS = 6;

/**
 * The tariff sheet while its rates or the rules module that checks them load: the header row
 * of buckets over rows of rate cells, and the save bar beneath.
 */
export function MatrixGridEditorSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading rates")} className="flex flex-col gap-4">
      <div aria-hidden="true" className="overflow-hidden rounded-md border">
        {Array.from({ length: ROWS + 1 }, (_, row) => (
          <div key={row} className="flex gap-3 border-b px-3 py-2 last:border-b-0">
            {Array.from({ length: COLUMNS }, (_, column) => (
              <Skeleton key={column} className={row === 0 ? "h-3 flex-1" : "h-7 flex-1"} />
            ))}
          </div>
        ))}
      </div>
      <div aria-hidden="true" className="flex justify-end">
        <Skeleton className="h-8 w-28" />
      </div>
    </div>
  );
}
