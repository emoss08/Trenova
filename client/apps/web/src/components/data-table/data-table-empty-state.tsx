import { Button } from "@/components/ui/button";

type DataTableEmptyStateProps = {
  hasActiveFilters?: boolean;
  onClearFilters?: () => void;
};

export function DataTableEmptyState({
  hasActiveFilters = false,
  onClearFilters,
}: DataTableEmptyStateProps) {
  return (
    <div className="flex size-full flex-col items-center justify-center overflow-hidden">
      <div className="relative size-full">
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-1">
          <p className="pointer-events-none bg-amber-300 px-1 py-0.5 text-center font-table text-sm/none font-medium text-amber-950 uppercase select-none dark:bg-amber-400 dark:text-neutral-900">
            No data available
          </p>
          <p className="pointer-events-none bg-neutral-900 px-1 py-0.5 text-center font-table text-sm/none font-medium text-white uppercase select-none dark:bg-neutral-500 dark:text-neutral-900">
            {hasActiveFilters
              ? "No records match your filters"
              : "Try adjusting your filters or search query"}
          </p>
          {hasActiveFilters && onClearFilters && (
            <Button type="button" variant="outline" size="xs" className="mt-2" onClick={onClearFilters}>
              Clear filters
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
