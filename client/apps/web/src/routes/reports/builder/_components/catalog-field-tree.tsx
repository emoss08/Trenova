import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import type {
  ReportCatalogEdge,
  ReportCatalogEntity,
  ReportCatalogField,
} from "@/lib/graphql/reports";
import { isEdgeAvailable, isFieldAvailable, listedFields } from "@/lib/report-catalog-access";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { cn } from "@trenova/shared/lib/utils";
import type { ReportFieldRef } from "@/types/report";
import {
  ChevronDownIcon,
  ChevronRightIcon,
  Lock01Icon,
  PlusIcon,
  SearchLgIcon,
} from "@trenova/shared/components/icons";
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

/** The protected field groups, as a role grants them. */
export const PROTECTED_GROUP_LABELS: Readonly<Record<string, string>> = defineLabels({
  Personal: "Personal",
  Margins: "Margins",
});

/**
 * Why a field cannot be used, in the words the row shows beside it; empty
 * for a field the report may use.
 */
export function fieldUnavailableReason(
  t: ReturnType<typeof useT>,
  field: ReportCatalogField,
  entity: Pick<ReportCatalogEntity, "pluralLabel">,
): string {
  switch (field.access) {
    case "Available":
      return "";
    case "NeedsFieldGroup":
      return t(
        "Needs the {0} field group",
        PROTECTED_GROUP_LABELS[field.protection?.group ?? ""] ?? field.protection?.group ?? "",
      );
    case "WithheldByCeiling":
      return t("Unavailable at your role's sensitivity level");
    case "NeedsKindRead":
      return t("Needs {0} read", t(entity.pluralLabel));
    default:
      return t("Unavailable");
  }
}

function FieldRow({
  field,
  entity,
  onSelect,
}: {
  field: ReportCatalogField;
  entity: ReportCatalogEntity;
  onSelect: () => void;
}) {
  const t = useT();
  const disabled = !isFieldAvailable(field);
  const reason = fieldUnavailableReason(t, field, entity);
  const group = field.protection?.group;

  return (
    <button
      type="button"
      className={cn(
        "group/field relative flex h-6.5 w-full items-center justify-between gap-2 rounded-md px-2 text-left text-xs",
        disabled
          ? "text-muted-foreground cursor-not-allowed"
          : "hover:bg-accent hover:text-accent-foreground transition-colors",
      )}
      onClick={onSelect}
      disabled={disabled}
      title={reason || field.description || undefined}
    >
      <span className="flex min-w-0 items-center gap-1.5">
        {(disabled || group) && <Lock01Icon className="size-3 shrink-0" aria-hidden />}
        <span className="truncate">{t(field.label)}</span>
        {group && (
          <Badge variant="neutral" appearance="outline" className="text-2xs shrink-0">
            {PROTECTED_GROUP_LABELS[group] ?? group}
          </Badge>
        )}
      </span>
      <span className="flex min-w-0 shrink-0 items-center gap-1">
        {reason ? (
          <span className="text-muted-foreground text-2xs truncate">{reason}</span>
        ) : (
          <Badge
            variant="neutral"
            appearance="outline"
            className="font-mono text-2xs transition-opacity group-hover/field:opacity-0"
          >
            {FIELD_TYPE_BADGES[field.type] ?? field.type}
          </Badge>
        )}
        {!disabled && (
          <PlusIcon className="text-muted-foreground absolute right-2 size-3.5 opacity-0 transition-opacity group-hover/field:opacity-100" />
        )}
      </span>
    </button>
  );
}

/** An edge the caller may not follow, listed so the report's limits are visible. */
function LockedEdgeRow({ edge }: { edge: ReportCatalogEdge }) {
  const t = useT();
  const reason = edge.needsReadOf
    ? t("Needs {0} read", t(edge.needsReadOf.label))
    : t("Unavailable");

  return (
    <div
      className="text-muted-foreground flex w-full items-center gap-1 rounded-md px-2 py-1 text-sm"
      title={reason}
    >
      <Lock01Icon className="size-3.5 shrink-0" aria-hidden />
      <span className="truncate">{t(edge.label)}</span>
      <span className="text-2xs ml-auto truncate">{reason}</span>
    </div>
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
  const t = useT();

  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const visibleFields = listedFields(entity).filter((field) => {
    if (filterFields && !filterFields(field, crossesToMany)) return false;
    if (!search) return true;
    return field.label.toLowerCase().includes(search) || field.key.toLowerCase().includes(search);
  });

  const traversableEdges = depth < MAX_PATH_DEPTH ? entity.edges.filter((e) => e.traversable) : [];

  return (
    <div className="flex flex-col">
      {!entity.readable && (
        <p className="bg-muted text-muted-foreground text-2xs mb-1 rounded-md px-2 py-1">
          {t("Reference only — the rest needs {0} read", t(entity.pluralLabel))}
        </p>
      )}
      {visibleFields.map((field) => (
        <FieldRow
          key={field.key}
          field={field}
          entity={entity}
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
        if (!isEdgeAvailable(edge)) {
          if (search) return null;
          return <LockedEdgeRow key={edge.name} edge={edge} />;
        }
        const target = index.entities.get(edge.target);
        if (!target) return null;
        const isOpen = expanded.has(edge.name) || search !== "";
        const edgeCrossesToMany = crossesToMany || edge.cardinality !== "one";
        return (
          <div key={edge.name} className="flex flex-col">
            <button
              type="button"
              className="hover:bg-accent hover:text-accent-foreground flex w-full items-center gap-1 rounded-md px-2 py-1 text-left text-sm font-medium"
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
              <span className="truncate">{t(edge.label)}</span>
              {edge.cardinality !== "one" && (
                <Badge variant="neutral" appearance="outline" className="text-2xs">
                  {t("many")}
                </Badge>
              )}
              {!target.readable && (
                <Badge variant="neutral" appearance="outline" className="text-2xs">
                  {t("Reference only")}
                </Badge>
              )}
            </button>
            {isOpen && (
              <div className="border-border ml-3 border-l pl-1">
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
  const t = useT();

  const [search, setSearch] = useState("");
  const entity = useMemo(() => index.entities.get(entityKey), [index, entityKey]);

  if (!entity) return null;

  return (
    <div className={cn("flex min-h-0 flex-col gap-2", className)}>
      <div className="relative">
        <Input
          className="h-8 pl-7"
          placeholder={t("Search fields...")}
          value={search}
          leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
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
