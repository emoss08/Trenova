import { EmptyTable } from "@trenova/shared/components/ui/empty-table";

const HISTORY_COLUMNS = [
  { label: "Status" },
  { label: "Settlement #" },
  { label: "Driver" },
  { label: "Type" },
  { label: "Period end" },
  { label: "Pay date" },
  { label: "Loads", numeric: true },
  { label: "Gross", numeric: true },
  { label: "Deductions", numeric: true },
  { label: "Net pay", numeric: true },
] as const;

type SettlementHistoryEmptyProps = {
  hasActiveFilters: boolean;
  onClearFilters: () => void;
};

/**
 * The history as it will look with settlements in it: the table's own
 * columns with nothing on the lines. A filter that hides every row offers
 * the way back; an empty record says where the first row comes from.
 */
export function SettlementHistoryEmpty({
  hasActiveFilters,
  onClearFilters,
}: SettlementHistoryEmptyProps) {
  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? "Nothing matches" : "No settlements yet"}
      description={
        hasActiveFilters
          ? "No settlement fits the search and filters. Widen them, or clear them to see the whole record."
          : "A settlement is written here when a pay period's statements are generated in the workspace, and stays as the record once it is paid."
      }
      columns={HISTORY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}
