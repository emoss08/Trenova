import {
  DescriptionListSkeleton,
  SectionBodySkeleton,
  SectionPanelSkeleton,
} from "@/components/section-panel-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

function AgreementRowsPlaceholder() {
  return (
    <div className="flex flex-col gap-3 px-3 py-3">
      <div className="flex items-center gap-2">
        <Skeleton className="h-5 w-16 rounded-full" />
        <Skeleton className="h-4 w-44" />
      </div>
      <DescriptionListSkeleton items={9} columns={3} />
    </div>
  );
}

/** The agreements list while it is read: one agreement's status, dates and terms. */
export function FactoringAgreementsBodySkeleton() {
  const t = useT();
  return (
    <SectionBodySkeleton label={t("Loading agreements")}>
      <AgreementRowsPlaceholder />
    </SectionBodySkeleton>
  );
}

/** The agreements section while its code loads. */
export function FactoringAgreementsSectionSkeleton() {
  const t = useT();
  return (
    <SectionPanelSkeleton label={t("Loading agreements")} action>
      <AgreementRowsPlaceholder />
    </SectionPanelSkeleton>
  );
}
