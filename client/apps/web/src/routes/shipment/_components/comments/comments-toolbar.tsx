import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { commentPriorityChoices, commentTypeChoices } from "@trenova/shared/lib/comment-choices";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, ListFilterIcon, SearchIcon, XIcon } from "lucide-react";
import { useMemo, useState } from "react";
import type { ShipmentCommentFilter } from "@/types/shipment-comment";

export interface CommentToolbarFilterState {
  types: string[];
  priorities: string[];
  mentionsMe: boolean;
  unresolvedOnly: boolean;
  pinnedOnly: boolean;
  search: string;
}

export const EMPTY_TOOLBAR_FILTERS: CommentToolbarFilterState = {
  types: [],
  priorities: [],
  mentionsMe: false,
  unresolvedOnly: false,
  pinnedOnly: false,
  search: "",
};

export function toolbarFiltersToQueryFilter(
  filters: CommentToolbarFilterState,
  currentUserId: string | undefined,
): ShipmentCommentFilter | null {
  const filter: ShipmentCommentFilter = {};
  if (filters.types.length > 0) filter.types = filters.types as ShipmentCommentFilter["types"];
  if (filters.priorities.length > 0) {
    filter.priorities = filters.priorities as ShipmentCommentFilter["priorities"];
  }
  if (filters.mentionsMe && currentUserId) filter.mentionsUserId = currentUserId;
  if (filters.unresolvedOnly) filter.unresolvedOnly = true;
  if (filters.pinnedOnly) filter.pinnedOnly = true;
  if (filters.search.trim() !== "") filter.search = filters.search.trim();
  return Object.keys(filter).length > 0 ? filter : null;
}

export function countActiveFilters(filters: CommentToolbarFilterState): number {
  return (
    filters.types.length +
    filters.priorities.length +
    (filters.mentionsMe ? 1 : 0) +
    (filters.unresolvedOnly ? 1 : 0) +
    (filters.pinnedOnly ? 1 : 0) +
    (filters.search.trim() !== "" ? 1 : 0)
  );
}

function FilterToggleRow({
  label,
  checked,
  onToggle,
}: {
  label: string;
  checked: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "hover:bg-accent flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs",
        checked && "bg-accent",
      )}
      onClick={onToggle}
    >
      <span className="truncate">{label}</span>
      {checked && <CheckIcon className="ml-auto size-3 shrink-0" />}
    </button>
  );
}

export function CommentsToolbar({
  filters,
  onFiltersChange,
  isFiltering,
}: {
  filters: CommentToolbarFilterState;
  onFiltersChange: (filters: CommentToolbarFilterState) => void;
  isFiltering: boolean;
}) {
  const [filterOpen, setFilterOpen] = useState(false);
  const activeCount = countActiveFilters(filters);

  const activePills = useMemo(() => {
    const pills: Array<{ key: string; label: string; onRemove: () => void }> = [];
    for (const type of filters.types) {
      pills.push({
        key: `type-${type}`,
        label: commentTypeChoices.find((c) => c.value === type)?.label ?? type,
        onRemove: () =>
          onFiltersChange({ ...filters, types: filters.types.filter((t) => t !== type) }),
      });
    }
    for (const priority of filters.priorities) {
      pills.push({
        key: `priority-${priority}`,
        label: commentPriorityChoices.find((c) => c.value === priority)?.label ?? priority,
        onRemove: () =>
          onFiltersChange({
            ...filters,
            priorities: filters.priorities.filter((p) => p !== priority),
          }),
      });
    }
    if (filters.mentionsMe) {
      pills.push({
        key: "mentions-me",
        label: "Mentions me",
        onRemove: () => onFiltersChange({ ...filters, mentionsMe: false }),
      });
    }
    if (filters.unresolvedOnly) {
      pills.push({
        key: "unresolved",
        label: "Unresolved",
        onRemove: () => onFiltersChange({ ...filters, unresolvedOnly: false }),
      });
    }
    if (filters.pinnedOnly) {
      pills.push({
        key: "pinned",
        label: "Pinned",
        onRemove: () => onFiltersChange({ ...filters, pinnedOnly: false }),
      });
    }
    return pills;
  }, [filters, onFiltersChange]);

  return (
    <div className="border-border border-b px-4 py-2">
      <div className="flex items-center gap-2">
        <div className="relative min-w-0 flex-1">
          <Input
            value={filters.search}
            onChange={(event) => onFiltersChange({ ...filters, search: event.target.value })}
            placeholder="Search comments…"
            leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
            className="h-7 pl-7 text-xs"
            aria-label="Search comments"
          />
          {isFiltering && (
            <span className="border-muted-foreground/40 absolute top-1/2 right-2 size-3 -translate-y-1/2 animate-spin rounded-full border border-t-transparent" />
          )}
        </div>
        <Popover open={filterOpen} onOpenChange={setFilterOpen}>
          <PopoverTrigger
            render={
              <Button
                type="button"
                variant={activeCount > 0 ? "secondary" : "ghost"}
                size="xs"
                className="h-7 gap-1.5 px-2 text-xs"
              >
                <ListFilterIcon className="size-3.5" />
                Filter
                {activeCount > 0 && (
                  <span className="bg-brand text-2xs rounded-full px-1.5 font-medium text-white">
                    {activeCount}
                  </span>
                )}
              </Button>
            }
          />
          <PopoverContent align="end" className="dark max-h-80 w-52 gap-1 overflow-y-auto p-1">
            <div className="text-2xs text-muted-foreground px-2 py-1 font-medium">
              Quick filters
            </div>
            <FilterToggleRow
              label="Mentions me"
              checked={filters.mentionsMe}
              onToggle={() => onFiltersChange({ ...filters, mentionsMe: !filters.mentionsMe })}
            />
            <FilterToggleRow
              label="Unresolved"
              checked={filters.unresolvedOnly}
              onToggle={() =>
                onFiltersChange({ ...filters, unresolvedOnly: !filters.unresolvedOnly })
              }
            />
            <FilterToggleRow
              label="Pinned"
              checked={filters.pinnedOnly}
              onToggle={() => onFiltersChange({ ...filters, pinnedOnly: !filters.pinnedOnly })}
            />
            <div className="text-2xs text-muted-foreground px-2 py-1 font-medium">Priority</div>
            {commentPriorityChoices.map((choice) => (
              <FilterToggleRow
                key={choice.value}
                label={choice.label}
                checked={filters.priorities.includes(choice.value)}
                onToggle={() =>
                  onFiltersChange({
                    ...filters,
                    priorities: filters.priorities.includes(choice.value)
                      ? filters.priorities.filter((p) => p !== choice.value)
                      : [...filters.priorities, choice.value],
                  })
                }
              />
            ))}
            <div className="text-2xs text-muted-foreground px-2 py-1 font-medium">Type</div>
            {commentTypeChoices.map((choice) => (
              <FilterToggleRow
                key={choice.value}
                label={choice.label}
                checked={filters.types.includes(choice.value)}
                onToggle={() =>
                  onFiltersChange({
                    ...filters,
                    types: filters.types.includes(choice.value)
                      ? filters.types.filter((t) => t !== choice.value)
                      : [...filters.types, choice.value],
                  })
                }
              />
            ))}
          </PopoverContent>
        </Popover>
      </div>
      {activePills.length > 0 && (
        <div className="mt-1.5 flex flex-wrap items-center gap-1">
          {activePills.map((pill) => (
            <button
              key={pill.key}
              type="button"
              className="border-border bg-muted text-2xs hover:bg-accent flex items-center gap-1 rounded-full border px-2 py-0.5 transition-colors"
              onClick={pill.onRemove}
            >
              {pill.label}
              <XIcon className="size-2.5" />
            </button>
          ))}
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="text-2xs text-muted-foreground h-5 px-1.5"
            onClick={() => onFiltersChange(EMPTY_TOOLBAR_FILTERS)}
          >
            Clear all
          </Button>
        </div>
      )}
    </div>
  );
}
