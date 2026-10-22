import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const FIGURES = 6;

/** The money line while the rules module that prices its charges loads: six figures in a row. */
export function MoneyLineSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the charges…")}
      className="flex flex-wrap gap-x-8 gap-y-3"
    >
      {Array.from({ length: FIGURES }, (_, index) => (
        <div key={index} aria-hidden="true" className="flex flex-col gap-1.5">
          <Skeleton className="h-2.5 w-12" />
          <Skeleton className="h-3.5 w-16" />
        </div>
      ))}
    </div>
  );
}
