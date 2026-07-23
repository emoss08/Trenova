import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import type { ReportCatalogEntity, ReportCatalogField } from "@/lib/graphql/reports";
import { cn } from "@/lib/utils";
import type { ReportFieldRef } from "@/types/report";
import { ChevronDownIcon, ChevronRightIcon, LockIcon, PlusIcon, SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { MAX_PATH_DEPTH, type CatalogIndex } from "./builder-state";

export type FieldSelection = {
  ref: ReportFieldRef;
  field: ReportCatalogField;
  crossesToMany: boolean;
};

type FieldTreeProps = {
  index: CatalogIndex;
  entityKey: string;
  onSelectField: (selection: FieldSelection) => void;
  filterFields?: (field: ReportCatalogField, crossesToMany: boolean) => boolean;
  className?: string;
};

const FIELD_TYPE_BADGES: Record<string, string> = {
  string: "abc",
  int: "123",
  decimal: "1.5",
  bool: "y/n",
  enum: "enum",
  epoch: "date",
  ref: "ref",
  json: "json",
};

function FieldRow({
  field,
  onSelect,
  disabled,
}: {
  field: ReportCatalogField;
  onSelect: () => void;
  disabled: boolean;
}) {
  return (
    <button
      type="button"
      className={cn(
        "group/field relative flex h-6.5 w-full items-center justify-between gap-2 rounded-md px-2 text-left text-xs",
        disabled
          ? "cursor-not-allowed text-muted-foreground/60"
          : "transition-colors hover:bg-accent hover:text-accent-foreground",
      )}
      onClick={onSelect}
      disabled={disabled}
      title={field.description ?? undefined}
    >
      <span className="flex min-w-0 items-center gap-1.5">
        {!field.accessible && <LockIcon className="size-3 shrink-0" />}
        <span className="truncate">{field.label}</span>
      </span>
      <span className="flex shrink-0 items-center gap-1">
        <Badge
          variant="outline"
          className="font-mono text-[10px] transition-opacity group-hover/field:opacity-0"
        >
          {FIELD_TYPE_BADGES[field.type] ?? field.type}
        </Badge>
        {!disabled && (
          <PlusIcon className="absolute right-2 size-3.5 text-muted-foreground opacity-0 transition-opacity group-hover/field:opacity-100" />
        )}
      </span>
    </button>
  );
}

function EntityFields({
  index,
  entity,
  path,
  crossesToMany,
  search,
  onSelectField,
  filterFields,
  depth,
}: {
  index: CatalogIndex;
  entity: ReportCatalogEntity;
  path: string[];
  crossesToMany: boolean;
  search: string;
  onSelectField: (selection: FieldSelection) => void;
  filterFields?: (field: ReportCatalogField, crossesToMany: boolean) => boolean;
  depth: number;
}) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const visibleFields = entity.fields.filter((field) => {
    if (filterFields && !filterFields(field, crossesToMany)) return false;
    if (!search) return true;
    return field.label.toLowerCase().includes(search) || field.key.toLowerCase().includes(search);
  });

  const traversableEdges = depth < MAX_PATH_DEPTH ? entity.edges.filter((e) => e.traversable) : [];

  return (
    <div className="flex flex-col">
      {visibleFields.map((field) => (
        <FieldRow
          key={field.key}
          field={field}
          disabled={!field.accessible}
          onSelect={() =>
            onSelectField({
              ref: { path: path.length > 0 ? path : undefined, field: field.key },
              field,
              crossesToMany,
            })
          }
        />
      ))}
      {traversableEdges.map((edge) => {
        const target = index.entities.get(edge.target);
        if (!target) return null;
        const isOpen = expanded.has(edge.name) || search !== "";
        const edgeCrossesToMany = crossesToMany || edge.cardinality !== "one";
        return (
          <div key={edge.name} className="flex flex-col">
            <button
              type="button"
              className="flex w-full items-center gap-1 rounded-md px-2 py-1 text-left text-sm font-medium hover:bg-accent hover:text-accent-foreground"
              onClick={() =>
                setExpanded((prev) => {
                  const next = new Set(prev);
                  if (next.has(edge.name)) {
                    next.delete(edge.name);
                  } else {
                    next.add(edge.name);
                  }
                  return next;
                })
              }
            >
              {isOpen ? (
                <ChevronDownIcon className="size-3.5 shrink-0" />
              ) : (
                <ChevronRightIcon className="size-3.5 shrink-0" />
              )}
              <span className="truncate">{edge.label}</span>
              {edge.cardinality !== "one" && (
                <Badge variant="outline" className="text-[10px]">
                  many
                </Badge>
              )}
            </button>
            {isOpen && (
              <div className="ml-3 border-l border-border pl-1">
                <EntityFields
                  index={index}
                  entity={target}
                  path={[...path, edge.name]}
                  crossesToMany={edgeCrossesToMany}
                  search={search}
                  onSelectField={onSelectField}
                  filterFields={filterFields}
                  depth={depth + 1}
                />
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

export function CatalogFieldTree({
  index,
  entityKey,
  onSelectField,
  filterFields,
  className,
}: FieldTreeProps) {
  const [search, setSearch] = useState("");
  const entity = useMemo(() => index.entities.get(entityKey), [index, entityKey]);

  if (!entity) return null;

  return (
    <div className={cn("flex min-h-0 flex-col gap-2", className)}>
      <div className="relative">
        <SearchIcon className="absolute top-2.5 left-2 size-3.5 text-muted-foreground" />
        <Input
          className="h-8 pl-7"
          placeholder="Search fields..."
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto">
        <EntityFields
          index={index}
          entity={entity}
          path={[]}
          crossesToMany={false}
          search={search.trim().toLowerCase()}
          onSelectField={onSelectField}
          filterFields={filterFields}
          depth={0}
        />
      </div>
    </div>
  );
}
