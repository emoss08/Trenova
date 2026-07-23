"use no memo";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

type DataTableSelectionBannerProps = {
  visible: boolean;
  selectedCount: number;
  totalCount: number;
  maxSelectable: number;
  isSelectingAll: boolean;
  onSelectAllMatching: () => void;
  onClearSelection: () => void;
};

export function DataTableSelectionBanner({
  visible,
  selectedCount,
  totalCount,
  maxSelectable,
  isSelectingAll,
  onSelectAllMatching,
  onClearSelection,
}: DataTableSelectionBannerProps) {
  if (!visible) return null;

  const target = Math.min(totalCount, maxSelectable);

  return (
    <div className="flex items-center justify-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-1 text-xs">
      <span className="text-muted-foreground">
        All <span className="font-medium text-foreground">{selectedCount}</span> rows on this page
        are selected.
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
              Selecting...
            </span>
          ) : (
            <>Select all {target.toLocaleString()} matching</>
          )}
        </Button>
      )}
      <Button
        type="button"
        variant="link"
        size="xs"
        className="h-auto p-0 text-xs text-muted-foreground"
        onClick={onClearSelection}
      >
        Clear selection
      </Button>
    </div>
  );
}
