import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const COLUMN_HEIGHTS = [92, 52, 44, 32, 28, 24, 20, 14, 10, 9, 8, 5];

/** The page as it lays out once the read lands: the brief, figures, next steps and the ranking. */
export function DetentionIntelligenceSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy="true"
      aria-label={t("Loading detention intelligence")}
      className="flex flex-col gap-10"
    >
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <span className="ui-shimmer block h-3 w-40 rounded-sm" />
          <span className="ui-shimmer block h-5 w-11/12 rounded-sm" />
          <span className="ui-shimmer block h-5 w-2/3 rounded-sm" />
        </div>
        <KpiStripSkeleton count={5} size="lg" bare />
        <div className="bg-card rounded-lg border">
          <div className="border-b px-4 py-3">
            <span className="ui-shimmer block h-3 w-32 rounded-sm" />
          </div>
          {Array.from({ length: 4 }, (_, index) => (
            <div key={index} className="flex items-center gap-4 border-b px-4 py-3 last:border-b-0">
              <span className="ui-shimmer block size-6 rounded-md" />
              <div className="flex flex-1 flex-col gap-1.5">
                <span className="ui-shimmer block h-3 w-1/2 rounded-sm" />
                <span className="ui-shimmer block h-2.5 w-4/5 rounded-sm" />
              </div>
              <span className="ui-shimmer block h-4 w-16 rounded-sm" />
              <span className="ui-shimmer block h-7 w-28 rounded-md" />
            </div>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-3">
        <span className="ui-shimmer block h-4 w-36 rounded-sm" />
        <span className="ui-shimmer block h-3 w-1/2 rounded-sm" />
        <div className="bg-card rounded-lg border p-4">
          <div className="flex h-56 items-end gap-4 px-6">
            {COLUMN_HEIGHTS.map((height, index) => (
              <span
                key={index}
                className="ui-shimmer block flex-1 rounded-t-sm"
                style={{ height: `${height}%` }}
              />
            ))}
          </div>
          <span className="ui-shimmer mt-4 block h-5 w-full rounded-md" />
        </div>
      </div>
    </div>
  );
}
