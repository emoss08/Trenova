import { Input } from "@trenova/shared/components/ui/input";
import type { ReportCatalog, ReportCatalogEntity } from "@/lib/graphql/reports";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronRightIcon, SearchIcon } from "lucide-react";
import { m } from "motion/react";
import { useMemo, useState } from "react";
import { CategoryTile } from "../../_components/report-card-chrome";

function EntityTile({
  entity,
  index,
  onSelect,
}: {
  entity: ReportCatalogEntity;
  index: number;
  onSelect: () => void;
}) {
  const accessibleFields = entity.fields.filter((field) => field.accessible).length;

  return (
    <m.button
      type="button"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, delay: Math.min(index, 16) * 0.02, ease: "easeOut" }}
      onClick={onSelect}
      className={cn(
        "group border-border bg-card flex items-center gap-3 rounded-lg border p-3 text-left",
        "transition-[border-color,box-shadow,background-color] duration-200",
        "hover:border-brand hover:bg-muted hover:ring-brand/25 hover:ring-2",
      )}
    >
      <CategoryTile category={entity.category} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{entity.label}</p>
        <p className="text-muted-foreground truncate text-xs">
          {entity.description || `${accessibleFields} fields`}
        </p>
      </div>
      <ChevronRightIcon className="text-muted-foreground/50 group-hover:text-foreground size-4 shrink-0 transition-transform group-hover:translate-x-0.5" />
    </m.button>
  );
}

export function EntityPicker({
  catalog,
  onSelect,
}: {
  catalog: ReportCatalog;
  onSelect: (entityKey: string) => void;
}) {
  const [search, setSearch] = useState("");

  const grouped = useMemo(() => {
    const term = search.trim().toLowerCase();
    const entities = catalog.entities.filter(
      (entity) =>
        !term ||
        entity.label.toLowerCase().includes(term) ||
        entity.category.toLowerCase().includes(term) ||
        (entity.description ?? "").toLowerCase().includes(term),
    );

    const byCategory = new Map<string, ReportCatalogEntity[]>();
    for (const entity of entities) {
      const bucket = byCategory.get(entity.category) ?? [];
      bucket.push(entity);
      byCategory.set(entity.category, bucket);
    }
    return [...byCategory.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [catalog.entities, search]);

  return (
    <div className="flex flex-1 justify-center overflow-y-auto">
      <div className="w-full max-w-3xl px-6 py-10">
        <m.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut" }}
        >
          <h2 className="text-lg font-semibold">What is this report about?</h2>
          <p className="text-muted-foreground mt-1 text-sm">
            Every report has one primary entity — it defines what each row represents. You can bring
            in related data through joins afterward.
          </p>
          <div className="relative mt-4">
            <Input
              autoFocus
              className="pl-8"
              placeholder="Search entities..."
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </div>
        </m.div>

        <div className="mt-6 flex flex-col gap-6">
          {grouped.length === 0 && (
            <p className="text-muted-foreground py-12 text-center text-sm">
              No entities match &quot;{search}&quot;.
            </p>
          )}
          {grouped.map(([category, entities], groupIndex) => {
            const offset = grouped
              .slice(0, groupIndex)
              .reduce((total, [, groupEntities]) => total + groupEntities.length, 0);
            return (
              <div key={category}>
                <p className="text-2xs text-muted-foreground mb-2 font-medium tracking-wide uppercase">
                  {category}
                </p>
                <div className="grid gap-2 sm:grid-cols-2">
                  {entities.map((entity, entityIndex) => (
                    <EntityTile
                      key={entity.key}
                      entity={entity}
                      index={offset + entityIndex}
                      onSelect={() => onSelect(entity.key)}
                    />
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
