import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const CHIPS = [64, 80, 72, 88] as const;

/** The band editor's preview while the rules module that labels its ranges loads. */
export function BandPreviewSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading the ranges…")} className="flex flex-wrap gap-1">
      {CHIPS.map((width, index) => (
        <Skeleton key={index} aria-hidden="true" className="h-4" style={{ width }} />
      ))}
    </div>
  );
}
