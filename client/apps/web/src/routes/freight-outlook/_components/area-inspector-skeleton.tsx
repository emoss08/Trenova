import { useT } from "@trenova/shared/i18n/use-t";

export function AreaInspectorSkeleton() {
  const t = useT();
  return (
    <div aria-busy aria-label={t("Loading the area")} className="flex flex-col gap-5">
      <div className="flex flex-col gap-2 border-b pb-4">
        <span className="h-3 w-28 ui-shimmer rounded-sm" />
        <span className="h-5 w-44 ui-shimmer rounded-sm" />
        <span className="h-4 w-full ui-shimmer rounded-sm" />
        <span className="h-4 w-3/4 ui-shimmer rounded-sm" />
      </div>
      <span className="h-32 w-full ui-shimmer rounded-md" />
      <div className="flex flex-col gap-2">
        {Array.from({ length: 3 }, (_, index) => (
          <span key={index} className="h-9 w-full ui-shimmer rounded-sm" />
        ))}
      </div>
      <span className="aspect-[2/1] w-full ui-shimmer rounded-lg" />
    </div>
  );
}
