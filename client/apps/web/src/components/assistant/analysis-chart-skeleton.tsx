import { useT } from "@trenova/shared/i18n/use-t";

/** A chart an analysis showed while its drawing code loads: the frame it is drawn in. */
export function AnalysisChartSkeleton({ title }: { title: string }) {
  const t = useT();

  return (
    <div aria-busy="true" aria-label={t("Loading {0}", title)} className="flex flex-col gap-1.5">
      <div aria-hidden className="rounded-surface border p-2">
        <div className="ui-shimmer h-64 w-full rounded-md" />
      </div>
    </div>
  );
}
