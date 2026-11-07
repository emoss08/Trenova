import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The workspace in shimmer: the figures bar, the brief and board, and the brief panel. */
export function FreightOutlookSkeleton() {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the freight outlook")}
      className="flex min-h-135 flex-1 flex-col"
    >
      <KpiStripSkeleton count={5} className="rounded-none border-x-0 border-t-0" />
      <div className="flex min-h-0 flex-1 max-xl:flex-col">
        <div className="flex min-w-0 flex-1 flex-col gap-3 px-4 pt-3.5 pb-4">
          <div className="flex items-start gap-3">
            <span className="size-5.5 ui-shimmer rounded-md" />
            <div className="flex flex-1 flex-col gap-1.5">
              <span className="h-4.5 w-full max-w-4xl ui-shimmer rounded-sm" />
              <span className="h-4.5 w-2/3 max-w-3xl ui-shimmer rounded-sm" />
            </div>
          </div>
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-border-subtle">
            <div className="flex items-center gap-3 px-3.5 pt-2.5">
              <span className="h-4 w-36 ui-shimmer rounded-sm" />
              <span className="h-3 w-72 ui-shimmer rounded-sm" />
            </div>
            <span className="mx-3.5 mt-3 h-31 ui-shimmer rounded-md" />
            <span className="mt-3 h-7.5 ui-shimmer" />
            <div className="flex flex-col gap-1.5 p-3.5">
              {Array.from({ length: 10 }, (_, index) => (
                <span key={index} className="h-5 w-full ui-shimmer rounded-sm" />
              ))}
            </div>
          </div>
        </div>
        <div className="flex shrink-0 flex-col gap-4 border-border-subtle bg-card p-3.5 max-xl:border-t xl:w-100 xl:border-l">
          <span className="h-4 w-24 ui-shimmer rounded-sm" />
          <span className="h-30 w-full ui-shimmer rounded-md" />
          {Array.from({ length: 5 }, (_, index) => (
            <span key={index} className="h-9 w-full ui-shimmer rounded-sm" />
          ))}
        </div>
      </div>
    </div>
  );
}
