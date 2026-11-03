import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const CARDS = 3;

/**
 * The packet board while the shipment's documents load, drawn as the board draws
 * them: the requirements line, then a lane of letter pages each over its name.
 */
export function BillingQueueDocumentBoardSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4" aria-busy aria-label={t("Loading documents...")}>
      <div className="contents" aria-hidden>
        <div className="flex items-center gap-3">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-3 w-20" />
        </div>
        <div className="flex flex-col gap-2.5">
          <Skeleton className="h-3 w-14" />
          <div className="grid grid-cols-[repeat(auto-fill,minmax(6.5rem,1fr))] gap-2.5">
            {Array.from({ length: CARDS }, (_, index) => (
              <div key={index} className="flex flex-col gap-1.5">
                <Skeleton className="aspect-[17/22] w-full rounded-md" />
                <Skeleton className="h-2.5 w-3/4" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
