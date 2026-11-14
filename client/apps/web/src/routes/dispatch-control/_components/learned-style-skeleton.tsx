import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = 3;

/** The learned style while it is read: the sentence, its weights and the reset, with no frame. */
export function LearnedStyleSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the learned dispatch style")}>
      <div aria-hidden className="flex flex-col gap-3">
        <Skeleton className="h-4 w-2/3" />
        {Array.from({ length: ROWS }, (_, index) => (
          <div key={index} className="flex items-center justify-between gap-4">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-4 w-24" />
          </div>
        ))}
        <Skeleton className="h-8 w-28" />
      </div>
    </div>
  );
}
