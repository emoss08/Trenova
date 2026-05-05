import { cn } from "@/lib/utils";
import type { ReactNode } from "react";
import { SAVED_VIEWS, type SavedViewId } from "./saved-views";
import { useCommandCenterUrl } from "./url-state";
import { useSavedViewCounts } from "./use-view-counts";

function ViewTab({
  id,
  label,
  count,
  isActive,
  onSelect,
}: {
  id: SavedViewId;
  label: string;
  count: number | undefined;
  isActive: boolean;
  onSelect: (id: SavedViewId) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(id)}
      aria-current={isActive ? "page" : undefined}
      className={cn(
        "-mb-px flex items-center gap-1.5 border-b-2 px-2.5 py-1.5 text-[11.5px] transition-colors",
        isActive
          ? "border-brand font-semibold text-foreground"
          : "border-transparent font-medium text-muted-foreground hover:text-foreground",
      )}
    >
      <span>{label}</span>
      <span
        className={cn(
          "inline-flex min-w-[20px] justify-center rounded-full px-1.5 py-px font-table text-[10px] tabular-nums",
          isActive
            ? "bg-brand/15 text-brand"
            : "bg-muted text-muted-foreground",
        )}
      >
        {count ?? "—"}
      </span>
    </button>
  );
}

export function SavedViewsBar({ rightSlot }: { rightSlot?: ReactNode }) {
  const [{ view }, setUrl] = useCommandCenterUrl();
  const counts = useSavedViewCounts();

  // Switching tabs resets pagination + collapses the open expansion so the
  // dispatcher lands on a clean page within the new view.
  const onSelect = (next: SavedViewId) => {
    void setUrl({ view: next, page: 1, expanded: null });
  };

  return (
    <div className="flex items-center justify-between gap-3 border-b border-border px-3 py-1">
      <div
        className="no-scrollbar flex flex-1 items-center gap-1 overflow-x-auto"
        role="tablist"
        aria-label="Saved views"
      >
        {SAVED_VIEWS.map((v) => (
          <ViewTab
            key={v.id}
            id={v.id}
            label={v.label}
            count={counts[v.id]}
            isActive={view === v.id}
            onSelect={onSelect}
          />
        ))}
      </div>
      {rightSlot && <div className="flex items-center gap-1.5">{rightSlot}</div>}
    </div>
  );
}
