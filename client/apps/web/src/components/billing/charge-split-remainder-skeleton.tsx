import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The split editor's remainder line while the rules module that measures it loads. */
export function ChargeSplitRemainderSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Measuring the split…")} className="flex h-4 items-center">
      <Skeleton aria-hidden="true" className="h-2.5 w-44" />
    </div>
  );
}
