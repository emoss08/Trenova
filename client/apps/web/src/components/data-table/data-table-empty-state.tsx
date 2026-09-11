import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable, type EmptyTableColumn } from "@trenova/shared/components/ui/empty-table";
import { pluralize } from "@trenova/shared/lib/utils";
import { PlusIcon } from "lucide-react";

type DataTableEmptyStateProps = {
  /** The record the table lists, as the table names itself. */
  name: string;
  /** The table's own visible columns, so the sketch is that table with nothing in it. */
  columns: readonly EmptyTableColumn[];
  hasActiveFilters: boolean;
  onClearFilters: () => void;
  /** The table's default create action, when the viewer may add a record. */
  onAddRecord?: () => void;
};

/**
 * What every data table shows in place of itself when a page comes back
 * empty. With a search or filter on, the way forward is to clear it; with
 * nothing recorded at all, it is to add the first record.
 */
export function DataTableEmptyState({
  name,
  columns,
  hasActiveFilters,
  onClearFilters,
  onAddRecord,
}: DataTableEmptyStateProps) {
  const t = useT();

  const records = pluralize(name.toLowerCase(), 2);
  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : `No ${records} yet`}
      description={
        hasActiveFilters
          ? `No ${name.toLowerCase()} fits the search and filters. Widen them, or clear them to see every one.`
          : onAddRecord
            ? `Nothing has been recorded here yet. Add the first ${name.toLowerCase()} and it appears here.`
            : `Nothing has been recorded here yet. The first ${name.toLowerCase()} appears here as soon as it exists.`
      }
      columns={columns}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
      action={
        onAddRecord ? (
          <Button variant="outline" size="sm" onClick={onAddRecord}>
            <PlusIcon className="size-3.5" />
            {t("Add {0}", name)}
          </Button>
        ) : null
      }
    />
  );
}
