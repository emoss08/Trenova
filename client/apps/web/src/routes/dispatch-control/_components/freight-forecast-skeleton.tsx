import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const FIGURES = 3;

/** The forecast's standing and figures while they are read, with no frame. */
export function FreightForecastSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the freight forecast")}>
      <div aria-hidden className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-5 w-24 rounded-badge" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
        </div>
        <div className="grid grid-cols-2 gap-x-6 gap-y-3">
          {Array.from({ length: FIGURES }, (_, index) => (
            <div key={index} className="flex flex-col gap-1">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-32" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
