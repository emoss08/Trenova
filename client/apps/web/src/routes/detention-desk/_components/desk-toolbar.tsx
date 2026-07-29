import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import {
  DESK_FILTERS,
  DESK_SORTS,
  type DeskFilterCounts,
  type DeskFilterId,
  type DeskSortId,
} from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronDownIcon, SearchIcon, XIcon } from "lucide-react";
import { useEffect, useRef } from "react";

type DeskToolbarProps = {
  lane: DeskFilterId;
  laneCounts: DeskFilterCounts;
  sort: DeskSortId;
  search: string;
  onLane: (lane: DeskFilterId) => void;
  onSort: (sort: DeskSortId) => void;
  onSearch: (search: string) => void;
};

/**
 * The lane being worked, the order it is read in, and the search. Every control
 * writes to the URL, so the view survives a refresh mid-shift and can be handed
 * to whoever picks the desk up next.
 */
export function DeskToolbar({
  lane,
  laneCounts,
  sort,
  search,
  onLane,
  onSort,
  onSearch,
}: DeskToolbarProps) {
  const searchRef = useRef<HTMLInputElement>(null);

  // "/" is the standard reach for search, but only when the keystroke is not
  // already going into a field.
  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }

      const target = event.target as HTMLElement | null;
      if (
        target instanceof HTMLInputElement ||
        target instanceof HTMLTextAreaElement ||
        target?.isContentEditable
      ) {
        return;
      }

      event.preventDefault();
      searchRef.current?.focus();
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []);

  const activeSort = DESK_SORTS.find((option) => option.id === sort) ?? DESK_SORTS[0];

  return (
    <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-2 border-b px-4">
      <div role="tablist" aria-label="Detention lane" className="flex items-center gap-4">
        {DESK_FILTERS.map((filter) => {
          const isActive = filter.id === lane;
          const count = laneCounts[filter.id];

          return (
            <button
              key={filter.id}
              type="button"
              role="tab"
              aria-selected={isActive}
              title={filter.hint}
              disabled={count === 0 && filter.id !== "all"}
              onClick={() => onLane(filter.id)}
              className={cn(
                "-mb-px border-b-2 pt-1 pb-2 text-xs transition-colors",
                "focus-visible:ring-ring focus-visible:-outline-offset-2 focus-visible:ring-2 focus-visible:outline-none",
                "disabled:pointer-events-none disabled:opacity-40",
                isActive
                  ? "border-foreground text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground",
              )}
            >
              {filter.label}
              <span className="ml-1.5 text-muted-foreground tabular-nums">{count}</span>
            </button>
          );
        })}
      </div>

      <div className="flex items-center gap-1.5 py-1.5">
        <Input
          ref={searchRef}
          value={search}
          onChange={(event) => onSearch(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape" && search.length > 0) {
              event.preventDefault();
              onSearch("");
            }
          }}
          placeholder="Search facility, customer, PRO"
          aria-label="Search the detention desk"
          inputContainerClassName="w-56"
          leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
          rightElement={
            search.length > 0 ? (
              <Button
                type="button"
                variant="outline"
                size="icon-xs"
                aria-label="Clear search"
                onClick={() => onSearch("")}
              >
                <XIcon className="size-3" />
              </Button>
            ) : (
              <Kbd className="mr-1 bg-transparent text-muted-foreground/60">/</Kbd>
            )
          }
        />

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="ghost"
                size="sm"
                className="gap-1 px-2 text-xs text-muted-foreground hover:text-foreground"
              >
                {activeSort.label}
                <ChevronDownIcon className="size-3" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="w-52">
            <DropdownMenuRadioGroup
              value={sort}
              onValueChange={(next) => onSort(next as DeskSortId)}
            >
              {DESK_SORTS.map((option) => (
                <DropdownMenuRadioItem key={option.id} value={option.id}>
                  <span className="flex flex-col">
                    <span>{option.label}</span>
                    <span className="text-2xs text-muted-foreground">{option.hint}</span>
                  </span>
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
