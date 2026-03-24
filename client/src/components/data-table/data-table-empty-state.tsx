export function DataTableEmptyState() {
  return (
    <div className="flex size-full flex-col items-center justify-center overflow-hidden">
      <div className="relative size-full">
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-1">
          <p className="bg-amber-300 px-1 py-0.5 text-center font-table text-sm/none font-medium text-amber-950 uppercase select-none dark:bg-amber-400 dark:text-neutral-900">
            No data available
          </p>
          <p className="bg-neutral-900 px-1 py-0.5 text-center font-table text-sm/none font-medium text-white uppercase select-none dark:bg-neutral-500 dark:text-neutral-900">
            Try adjusting your filters or search query
          </p>
        </div>
      </div>
    </div>
  );
}
