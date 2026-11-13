import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = ["w-4/5", "w-3/5", "w-2/3"] as const;

/** The record's memories while they load: the list's own rows, shimmering. */
export function RecordMemoriesSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-2" aria-busy="true" aria-label={t("Loading memories")}>
      {ROWS.map((width) => (
        <div
          key={width}
          className="border-border flex flex-col gap-2 rounded-md border px-3 py-2.5"
        >
          <span className={`ui-shimmer h-4 rounded ${width}`} />
          <span className="ui-shimmer h-3 w-1/3 rounded" />
        </div>
      ))}
    </div>
  );
}
