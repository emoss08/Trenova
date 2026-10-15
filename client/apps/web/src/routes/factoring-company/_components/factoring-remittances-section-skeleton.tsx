import { SectionBodySkeleton, SectionPanelSkeleton } from "@/components/section-panel-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const REMITTANCE_ROWS = 3;

function RemittanceRowsPlaceholder() {
  return (
    <div className="divide-border divide-y">
      {Array.from({ length: REMITTANCE_ROWS }, (_, index) => (
        <div key={index} className="flex items-center justify-between gap-3 px-3 py-2">
          <div className="flex min-w-0 flex-col gap-1">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-3 w-32" />
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-5 w-16 rounded-full" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** The remittances list while it is read: each report's file, date, total and status. */
export function FactoringRemittancesBodySkeleton() {
  const t = useT();
  return (
    <SectionBodySkeleton label={t("Loading remittances")}>
      <RemittanceRowsPlaceholder />
    </SectionBodySkeleton>
  );
}

/** The remittances section while its code loads. */
export function FactoringRemittancesSectionSkeleton() {
  const t = useT();
  return (
    <SectionPanelSkeleton label={t("Loading remittances")} action>
      <RemittanceRowsPlaceholder />
    </SectionPanelSkeleton>
  );
}
