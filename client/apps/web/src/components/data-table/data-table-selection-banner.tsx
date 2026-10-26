import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useTableAtom, type TableAtomSource } from "@trenova/shared/hooks/use-table-atom";
import { countSelectedRows } from "@trenova/shared/lib/table-features";
import { formatNumber } from "@trenova/shared/i18n/format";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
import type { RowSelectionState } from "@tanstack/react-table";

type DataTableSelectionBannerProps = {
  selection: TableAtomSource<RowSelectionState>;
  pageRowIds: readonly string[];
  totalCount: number;
  /** False when totalCount is the server's estimate of a list too large to count. */
  totalCountExact?: boolean;
  maxSelectable: number;
  isSelectingAll: boolean;
  onSelectAllMatching: () => void;
  onClearSelection: () => void;
  /** Edits every row the filters match on the server, however many there are. */
  onEditAllMatching?: () => void;
};

export function DataTableSelectionBanner({
  selection,
  pageRowIds,
  totalCount,
  totalCountExact = true,
  maxSelectable,
  isSelectingAll,
  onSelectAllMatching,
  onClearSelection,
  onEditAllMatching,
}: DataTableSelectionBannerProps) {
  const t = useT();
  const rt = useRichT();
  const allPageRowsSelected = useTableAtom(
    selection,
    (current) => pageRowIds.length > 0 && pageRowIds.every((id) => current[id]),
  );
  const selectedCount = useTableAtom(selection, countSelectedRows);

  if (!allPageRowsSelected || totalCount <= pageRowIds.length) return null;

  const target = Math.min(totalCount, maxSelectable);

  return (
    <div className="border-border bg-muted/40 bleed:rounded-none bleed:border-x-0 bleed:border-t-0 flex items-center justify-center gap-2 rounded-md border px-3 py-1 text-xs">
      <span className="text-muted-foreground">
        {rt(
          "All <b>{0}</b> {0, plural, one {row on this page is selected.} other {rows on this page are selected.}}",
          { b: (c) => <span className="text-foreground font-medium">{c}</span> },
          selectedCount,
        )}
      </span>
      {selectedCount < target && (
        <Button
          type="button"
          variant="link"
          size="xs"
          className="h-auto p-0 text-xs"
          disabled={isSelectingAll}
          onClick={onSelectAllMatching}
        >
          {isSelectingAll ? (
            <span className="flex items-center gap-1.5">
              <Spinner className="size-3" />
              {t("Selecting...")}
            </span>
          ) : (
            <>{t("Select all {0} matching", formatNumber(target))}</>
          )}
        </Button>
      )}
      {onEditAllMatching ? (
        <Button
          type="button"
          variant="link"
          size="xs"
          className="h-auto p-0 text-xs"
          onClick={onEditAllMatching}
        >
          {totalCountExact
            ? t("Edit all {0} matching", formatRowTotal(totalCount, true))
            : t("Edit all about {0} matching", formatRowTotal(totalCount, false))}
        </Button>
      ) : null}
      <Button
        type="button"
        variant="link"
        size="xs"
        className="text-muted-foreground h-auto p-0 text-xs"
        onClick={onClearSelection}
      >
        {t("Clear selection")}
      </Button>
    </div>
  );
}
