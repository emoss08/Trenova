import {
  DescriptionListSkeleton,
  SectionBodySkeleton,
  SectionPanelSkeleton,
} from "@/components/section-panel-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const NOTICE_ROWS = 2;

function NoticeRowsPlaceholder() {
  return (
    <div className="divide-y">
      {Array.from({ length: NOTICE_ROWS }, (_, index) => (
        <div key={index} className="flex flex-col gap-2 px-3 py-3">
          <div className="flex items-center gap-2">
            <Skeleton className="h-5 w-16 rounded-full" />
            <Skeleton className="h-4 w-40" />
          </div>
          <DescriptionListSkeleton items={4} columns={4} />
        </div>
      ))}
    </div>
  );
}

/** The notices list while it is read: each notice's status, customer and dates. */
export function FactoringNoticesBodySkeleton() {
  const t = useT();
  return (
    <SectionBodySkeleton label={t("Loading notices of assignment")}>
      <NoticeRowsPlaceholder />
    </SectionBodySkeleton>
  );
}

/** The notices section while its code loads. */
export function FactoringNoticesSectionSkeleton() {
  const t = useT();
  return (
    <SectionPanelSkeleton label={t("Loading notices of assignment")} action>
      <NoticeRowsPlaceholder />
    </SectionPanelSkeleton>
  );
}
