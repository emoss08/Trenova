import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import type { CatalogFragment, CatalogOperation, CatalogSelection } from "@/types/graphql-catalog";
import { useVirtualizer } from "@tanstack/react-virtual";
import { SearchIcon, SearchXIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { type CatalogFilter, searchCatalog } from "./catalog";

const ROW_HEIGHT = 30;
const HEADER_HEIGHT = 28;

const FILTERS: { value: CatalogFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "query", label: "Queries" },
  { value: "mutation", label: "Mutations" },
  { value: "fragment", label: "Fragments" },
];

type GlyphKind = "query" | "mutation" | "subscription" | "fragment";

type ListItem =
  | { type: "header"; label: string; count: number }
  | { type: "operation"; operation: CatalogOperation }
  | { type: "fragment"; fragment: CatalogFragment };

function itemKey(item: ListItem): string {
  switch (item.type) {
    case "header":
      return `header:${item.label}`;
    case "operation":
      return `op:${item.operation.name}`;
    case "fragment":
      return `fr:${item.fragment.name}`;
  }
}

function itemSelection(item: ListItem): CatalogSelection | null {
  switch (item.type) {
    case "operation":
      return { kind: "operation", name: item.operation.name };
    case "fragment":
      return { kind: "fragment", name: item.fragment.name };
    default:
      return null;
  }
}

function KindGlyph({ kind }: { kind: GlyphKind }) {
  const config: Record<GlyphKind, { letter: string; className: string }> = {
    query: { letter: "Q", className: "bg-sky-500/15 text-sky-600 dark:text-sky-400" },
    mutation: { letter: "M", className: "bg-amber-500/15 text-amber-600 dark:text-amber-400" },
    subscription: {
      letter: "S",
      className: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
    },
    fragment: { letter: "F", className: "bg-violet-500/15 text-violet-600 dark:text-violet-400" },
  };
  const { letter, className } = config[kind];
  return (
    <span
      className={cn(
        "flex size-5 shrink-0 items-center justify-center rounded font-mono text-2xs font-semibold",
        className,
      )}
    >
      {letter}
    </span>
  );
}

function HighlightedName({ name, needle }: { name: string; needle: string }) {
  if (!needle) {
    return name;
  }
  const index = name.toLowerCase().indexOf(needle);
  if (index < 0) {
    return name;
  }
  return (
    <>
      {name.slice(0, index)}
      <span className="text-primary">{name.slice(index, index + needle.length)}</span>
      {name.slice(index + needle.length)}
    </>
  );
}

function Row({
  name,
  kind,
  domain,
  needle,
  selected,
  active,
  onSelect,
}: {
  name: string;
  kind: GlyphKind;
  domain: string;
  needle: string;
  selected: boolean;
  active: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "flex h-full w-full items-center gap-2 rounded-md px-2 text-left transition-colors",
        selected ? "bg-primary/10 text-foreground" : "hover:bg-muted/60",
        active && !selected && "bg-muted",
        active && "ring-1 ring-primary/30 ring-inset",
      )}
    >
      <KindGlyph kind={kind} />
      <span className="min-w-0 flex-1 truncate font-mono text-xs">
        <HighlightedName name={name} needle={needle} />
      </span>
      <span className="shrink-0 text-2xs text-muted-foreground/70">{domain}</span>
    </button>
  );
}

export function ListPanel({
  query,
  filter,
  selection,
  onQueryChange,
  onFilterChange,
  onSelect,
}: {
  query: string;
  filter: CatalogFilter;
  selection: CatalogSelection | null;
  onQueryChange: (value: string) => void;
  onFilterChange: (value: CatalogFilter) => void;
  onSelect: (selection: CatalogSelection) => void;
}) {
  const results = useMemo(() => searchCatalog(query, filter), [query, filter]);
  const needle = query.trim().toLowerCase();

  const counts = useMemo(() => {
    const all = searchCatalog(query, "all");
    let queries = 0;
    let mutations = 0;
    for (const operation of all.operations) {
      if (operation.kind === "mutation") {
        mutations += 1;
      } else {
        queries += 1;
      }
    }
    return { all: all.total, query: queries, mutation: mutations, fragment: all.fragments.length };
  }, [query]);

  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) {
        return;
      }
      const target = event.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable)
      ) {
        return;
      }
      event.preventDefault();
      inputRef.current?.focus();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const items = useMemo<ListItem[]>(() => {
    const list: ListItem[] = [];
    if (results.operations.length > 0) {
      list.push({ type: "header", label: "Operations", count: results.operations.length });
      for (const operation of results.operations) {
        list.push({ type: "operation", operation });
      }
    }
    if (results.fragments.length > 0) {
      list.push({ type: "header", label: "Fragments", count: results.fragments.length });
      for (const fragment of results.fragments) {
        list.push({ type: "fragment", fragment });
      }
    }
    return list;
  }, [results]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const getScrollElement = useCallback(() => {
    const root = scrollRef.current;
    if (!root) {
      return null;
    }
    return (
      root.querySelector<HTMLDivElement>('[data-slot="scroll-area-viewport"]') ??
      (root.firstElementChild as HTMLDivElement | null)
    );
  }, []);
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement,
    estimateSize: (index) => (items[index].type === "header" ? HEADER_HEIGHT : ROW_HEIGHT),
    getItemKey: (index) => itemKey(items[index]),
    overscan: 12,
    paddingStart: 6,
    paddingEnd: 6,
  });

  const selectableIndices = useMemo(
    () =>
      items.reduce<number[]>((acc, item, index) => {
        if (item.type !== "header") {
          acc.push(index);
        }
        return acc;
      }, []),
    [items],
  );

  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  useEffect(() => {
    setActiveIndex(null);
  }, [query, filter]);

  const moveActive = (direction: 1 | -1) => {
    if (selectableIndices.length === 0) {
      return;
    }
    const position =
      activeIndex === null
        ? direction === 1
          ? 0
          : selectableIndices.length - 1
        : Math.min(
            Math.max(selectableIndices.indexOf(activeIndex) + direction, 0),
            selectableIndices.length - 1,
          );
    const next = selectableIndices[position];
    setActiveIndex(next);
    virtualizer.scrollToIndex(next);
  };

  const handleSearchKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      moveActive(1);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      moveActive(-1);
    } else if (event.key === "Enter" && activeIndex !== null) {
      event.preventDefault();
      const target = items[activeIndex] ? itemSelection(items[activeIndex]) : null;
      if (target) {
        onSelect(target);
      }
    } else if (event.key === "Escape" && query) {
      event.preventDefault();
      onQueryChange("");
    }
  };

  const isSelected = (kind: CatalogSelection["kind"], name: string) =>
    selection?.kind === kind && selection.name === name;

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-col gap-2 border-b p-2.5">
        <Input
          ref={inputRef}
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          onKeyDown={handleSearchKeyDown}
          placeholder="Search operations…"
          className="h-8 pl-8 text-sm"
          leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
          autoFocus
        />
        <div className="flex items-center gap-0.5 rounded-lg bg-muted/60 p-0.5">
          {FILTERS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => onFilterChange(option.value)}
              className={cn(
                "flex flex-1 items-center justify-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors",
                filter === option.value
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {option.label}
              <span
                className={cn(
                  "text-2xs tabular-nums",
                  filter === option.value ? "text-muted-foreground" : "text-muted-foreground/60",
                )}
              >
                {counts[option.value]}
              </span>
            </button>
          ))}
        </div>
      </div>

      <ScrollArea
        ref={scrollRef}
        className="min-h-0 flex-1 [&_[data-slot=scroll-area-viewport]>div]:block!"
      >
        {items.length === 0 ? (
          <div className="flex flex-col items-center gap-2 px-4 py-10 text-center">
            <SearchXIcon className="size-5 text-muted-foreground/50" />
            <p className="text-xs text-muted-foreground">
              No matches{query ? ` for “${query}”` : ""}
            </p>
          </div>
        ) : (
          <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
            {virtualizer.getVirtualItems().map((virtualItem) => {
              const item = items[virtualItem.index];
              return (
                <div
                  key={virtualItem.key}
                  className={cn(
                    "absolute inset-x-0 top-0 px-1.5",
                    item.type !== "header" && "py-px",
                  )}
                  style={{
                    transform: `translateY(${virtualItem.start}px)`,
                    height: virtualItem.size,
                  }}
                >
                  {item.type === "header" ? (
                    <div className="flex h-full items-end px-2 pb-1">
                      <span className="text-2xs font-medium tracking-wider text-muted-foreground/60 uppercase">
                        {item.label} · {item.count}
                      </span>
                    </div>
                  ) : item.type === "operation" ? (
                    <Row
                      name={item.operation.name}
                      kind={item.operation.kind}
                      domain={item.operation.domain}
                      needle={needle}
                      selected={isSelected("operation", item.operation.name)}
                      active={virtualItem.index === activeIndex}
                      onSelect={() => onSelect({ kind: "operation", name: item.operation.name })}
                    />
                  ) : (
                    <Row
                      name={item.fragment.name}
                      kind="fragment"
                      domain={item.fragment.domain}
                      needle={needle}
                      selected={isSelected("fragment", item.fragment.name)}
                      active={virtualItem.index === activeIndex}
                      onSelect={() => onSelect({ kind: "fragment", name: item.fragment.name })}
                    />
                  )}
                </div>
              );
            })}
          </div>
        )}
      </ScrollArea>

      <div className="flex items-center gap-3 border-t px-3 py-1.5 text-2xs text-muted-foreground/70">
        <span className="flex items-center gap-1">
          <Kbd>↑↓</Kbd> navigate
        </span>
        <span className="flex items-center gap-1">
          <Kbd>⏎</Kbd> open
        </span>
        <span className="flex items-center gap-1">
          <Kbd>/</Kbd> search
        </span>
        <span className="ml-auto flex items-center gap-1">
          <Kbd>⌘⏎</Kbd> run
        </span>
      </div>
    </div>
  );
}
