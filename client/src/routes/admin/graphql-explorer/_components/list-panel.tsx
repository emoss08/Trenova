import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type {
  CatalogFragment,
  CatalogOperation,
  CatalogSelection,
} from "@/types/graphql-catalog";
import { SearchIcon } from "lucide-react";
import { useMemo } from "react";
import { type CatalogFilter, searchCatalog } from "./catalog";

const FILTERS: { value: CatalogFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "query", label: "Queries" },
  { value: "mutation", label: "Mutations" },
  { value: "fragment", label: "Fragments" },
];

type GlyphKind = "query" | "mutation" | "subscription" | "fragment";

function KindGlyph({ kind }: { kind: GlyphKind }) {
  const config: Record<GlyphKind, { letter: string; className: string }> = {
    query: { letter: "Q", className: "bg-sky-500/15 text-sky-600 dark:text-sky-400" },
    mutation: { letter: "M", className: "bg-amber-500/15 text-amber-600 dark:text-amber-400" },
    subscription: { letter: "S", className: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400" },
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

function Row({
  name,
  kind,
  domain,
  selected,
  onSelect,
}: {
  name: string;
  kind: GlyphKind;
  domain: string;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors",
        selected ? "bg-primary/10 text-foreground" : "hover:bg-muted/60",
      )}
    >
      <KindGlyph kind={kind} />
      <span className="min-w-0 flex-1 truncate font-mono text-xs">{name}</span>
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

  const isSelected = (kind: CatalogSelection["kind"], name: string) =>
    selection?.kind === kind && selection.name === name;

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-col gap-2 border-b p-2.5">
        <div className="relative">
          <SearchIcon className="absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="Search operations…"
            className="h-8 pl-8 text-sm"
            autoFocus
          />
        </div>
        <div className="flex items-center gap-1">
          {FILTERS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => onFilterChange(option.value)}
              className={cn(
                "rounded-md px-2 py-1 text-xs font-medium transition-colors",
                filter === option.value
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-muted",
              )}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto p-1.5">
        {results.total === 0 ? (
          <p className="px-2 py-8 text-center text-xs text-muted-foreground">No matches</p>
        ) : (
          <div className="flex flex-col gap-3">
            {results.operations.length > 0 && (
              <Group
                label="Operations"
                count={results.operations.length}
                items={results.operations}
                render={(op: CatalogOperation) => (
                  <Row
                    key={`op:${op.name}`}
                    name={op.name}
                    kind={op.kind}
                    domain={op.domain}
                    selected={isSelected("operation", op.name)}
                    onSelect={() => onSelect({ kind: "operation", name: op.name })}
                  />
                )}
              />
            )}
            {results.fragments.length > 0 && (
              <Group
                label="Fragments"
                count={results.fragments.length}
                items={results.fragments}
                render={(fragment: CatalogFragment) => (
                  <Row
                    key={`fr:${fragment.name}`}
                    name={fragment.name}
                    kind="fragment"
                    domain={fragment.domain}
                    selected={isSelected("fragment", fragment.name)}
                    onSelect={() => onSelect({ kind: "fragment", name: fragment.name })}
                  />
                )}
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function Group<T>({
  label,
  count,
  items,
  render,
}: {
  label: string;
  count: number;
  items: T[];
  render: (item: T) => React.ReactNode;
}) {
  return (
    <div>
      <div className="px-2 pb-1">
        <span className="text-2xs font-medium tracking-wider text-muted-foreground/60 uppercase">
          {label} · {count}
        </span>
      </div>
      <div className="flex flex-col gap-0.5">{items.map(render)}</div>
    </div>
  );
}
