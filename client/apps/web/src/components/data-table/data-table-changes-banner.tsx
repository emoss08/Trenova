import { isChangedSince } from "@/lib/data-table";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useShallow } from "zustand/react/shallow";
import { useDataTableView, useDataTableViewActions } from "./data-table-view-context";
import type { DataTableViewState } from "./data-table-view-store";

type DataTableChangesBannerProps = {
  /** The rows on screen, the ones it counts. */
  rows: readonly unknown[];
};

const selectChanges = (state: DataTableViewState) => ({
  since: state.markChangesSinceLastVisit ? state.changesSince : 0,
  seenRowIds: state.seenRowIds,
});

/**
 * Says how many rows on the page changed since the person last had the table open and have
 * not been opened since; nothing when the person does not mark changes, or none are left.
 */
export function DataTableChangesBanner({ rows }: DataTableChangesBannerProps) {
  const t = useT();
  const { since, seenRowIds } = useDataTableView(useShallow(selectChanges));
  const { markAllSeen } = useDataTableViewActions();

  let count = 0;
  if (since > 0) {
    for (const row of rows) {
      const id = (row as { id?: string }).id;
      if (id && !seenRowIds.has(id) && isChangedSince(row, since)) count += 1;
    }
  }
  if (count === 0) return null;

  return (
    <div className="border-border bg-muted/40 bleed:rounded-none bleed:border-x-0 bleed:border-t-0 flex items-center justify-center gap-2 rounded-md border px-3 py-1 text-xs">
      <span aria-hidden className="bg-info size-1.5 rounded-full" />
      <span className="text-muted-foreground">
        {t(
          "{0, plural, one {# row changed since your last visit} other {# rows changed since your last visit}}",
          count,
        )}
      </span>
      <Button
        type="button"
        variant="link"
        size="xs"
        className="h-auto p-0 text-xs"
        onClick={markAllSeen}
      >
        {t("Mark all seen")}
      </Button>
    </div>
  );
}
