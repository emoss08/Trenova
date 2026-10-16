import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** What the memory table shows at first: what was said, its kind, scope, status and uses. */
const MEMORY_COLUMNS = 5;

/** The Memory tab while its code is read: the table of memories in the section that holds it. */
export function MemoryTabSkeleton() {
  const t = useT();

  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading memory")}>
      <section className="sec">
        <DataTableSkeleton columnCount={MEMORY_COLUMNS} aria-hidden />
      </section>
    </div>
  );
}
