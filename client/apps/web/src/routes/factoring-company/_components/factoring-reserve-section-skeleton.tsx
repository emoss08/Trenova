import {
  DescriptionListSkeleton,
  SectionBodySkeleton,
  SectionPanelSkeleton,
} from "@/components/section-panel-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ENTRY_ROWS = 3;
const ENTRY_COLUMNS = 6;

function ReservePlaceholder() {
  return (
    <div className="flex flex-col">
      <div className="p-3">
        <DescriptionListSkeleton items={7} columns={2} />
      </div>
      {Array.from({ length: ENTRY_ROWS + 1 }, (_, row) => (
        <div key={row} className="grid grid-cols-6 gap-3 border-t px-3 py-2">
          {Array.from({ length: ENTRY_COLUMNS }, (_, column) => (
            <Skeleton key={column} className={row === 0 ? "h-3 w-12" : "h-4 w-16"} />
          ))}
        </div>
      ))}
    </div>
  );
}

/** The reserve while it is read: the balances, then the latest movements. */
export function FactoringReserveBodySkeleton() {
  const t = useT();
  return (
    <SectionBodySkeleton label={t("Loading the reserve")}>
      <ReservePlaceholder />
    </SectionBodySkeleton>
  );
}

/** The reserve section while its code loads. */
export function FactoringReserveSectionSkeleton() {
  const t = useT();
  return (
    <SectionPanelSkeleton label={t("Loading the reserve")}>
      <ReservePlaceholder />
    </SectionPanelSkeleton>
  );
}
