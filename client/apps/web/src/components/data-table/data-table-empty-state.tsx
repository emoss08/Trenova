import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable, type EmptyTableColumn } from "@trenova/shared/components/ui/empty-table";
import { PlusIcon } from "@trenova/shared/components/icons";

type DataTableEmptyStateProps = {
  /** The table's own visible columns, so the sketch is that table with nothing in it. */
  columns: readonly EmptyTableColumn[];
  hasActiveFilters: boolean;
  /** The translated title naming the table's records, shown when no filter is on. */
  title?: string;
  onClearFilters: () => void;
  /** The table's default create action, when the viewer may add a record. */
  addRecord?: {
    /** The translated label the table's own create action carries. */
    label: string;
    onClick: () => void;
  };
};

/**
 * What every data table shows in place of itself when a page comes back
 * empty. With a search or filter on, the way forward is to clear it; with
 * nothing recorded at all, it is to add the first record.
 */
export function DataTableEmptyState({
  columns,
  hasActiveFilters,
  title,
  onClearFilters,
  addRecord,
}: DataTableEmptyStateProps) {
  const t = useT();

  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? t("Nothing matches") : (title ?? t("No records yet"))}
      description={
        hasActiveFilters
          ? t("No record fits the search and filters. Widen them, or clear them to see every one.")
          : addRecord
            ? t("Nothing has been recorded here yet. Add the first record and it appears here.")
            : t(
                "Nothing has been recorded here yet. The first record appears here as soon as it exists.",
              )
      }
      columns={columns}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
      action={
        addRecord ? (
          <Button size="sm" onClick={addRecord.onClick}>
            <PlusIcon className="size-3.5" />
            {addRecord.label}
          </Button>
        ) : null
      }
    />
  );
}
