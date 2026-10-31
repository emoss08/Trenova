import { useT } from "@trenova/shared/i18n/use-t";

/** The permits tab's shape while its assessment loads: the summary, the four meters, three states. */
export function PermitsTabSkeleton() {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading permits")} className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <span className="ui-shimmer h-4 w-40 rounded-sm" />
        <span className="ui-shimmer h-1.5 w-full rounded-full" />
        <span className="ui-shimmer h-3 w-64 rounded-sm" />
      </div>
      <div className="flex flex-col gap-3">
        {[0, 1, 2, 3].map((row) => (
          <div key={row} className="flex items-center gap-3">
            <span className="ui-shimmer h-3 w-12 rounded-sm" />
            <span className="ui-shimmer h-1.5 flex-1 rounded-full" />
            <span className="ui-shimmer h-3 w-24 rounded-sm" />
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-3">
        {[0, 1, 2].map((row) => (
          <div key={row} className="flex items-center gap-3">
            <span className="ui-shimmer size-8 rounded-md" />
            <div className="flex flex-1 flex-col gap-1">
              <span className="ui-shimmer h-3 w-32 rounded-sm" />
              <span className="ui-shimmer h-2.5 w-48 rounded-sm" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
