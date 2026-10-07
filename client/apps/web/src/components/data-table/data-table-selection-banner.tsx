"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useRichT } from "@trenova/shared/i18n/rich";

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
  const t = useT();
  const rt = useRichT();

  if (!visible) return null;

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
            <>{t("Select all {0} matching", target.toLocaleString())}</>
          )}
        </Button>
      )}
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
