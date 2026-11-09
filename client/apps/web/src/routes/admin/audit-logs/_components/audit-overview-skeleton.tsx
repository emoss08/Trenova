import { useT } from "@trenova/shared/i18n/use-t";

/** The overview's own layout in shimmer: the briefing, five figures and the three cards. */
export function AuditOverviewSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading the audit overview")} className="flex flex-col gap-4">
      <div aria-hidden className="flex flex-col gap-2">
        <span className="block h-3 w-48 ui-shimmer rounded-sm" />
        <span className="block h-5 w-11/12 ui-shimmer rounded-sm" />
        <span className="block h-5 w-2/3 ui-shimmer rounded-sm" />
      </div>
      <div aria-hidden className="grid grid-cols-2 gap-4 border-y py-3 md:grid-cols-5">
        {Array.from({ length: 5 }, (_, i) => (
          <div key={i} className="flex flex-col gap-2">
            <span className="block h-3 w-24 ui-shimmer rounded-sm" />
            <span className="block h-6 w-16 ui-shimmer rounded-sm" />
            <span className="block h-3 w-32 ui-shimmer rounded-sm" />
          </div>
        ))}
      </div>
      <div aria-hidden className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_18rem_23rem]">
        <span className="block h-60 ui-shimmer rounded-md" />
        <span className="block h-60 ui-shimmer rounded-md" />
        <span className="block h-60 ui-shimmer rounded-md" />
      </div>
    </div>
  );
}
