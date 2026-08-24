import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  operatorsForFieldType,
  type ReportColumnSpec,
  type ReportFieldFilter,
  type ReportFilterGroup,
  type ReportIR,
} from "@/types/report";
import { FilterIcon, PlusIcon, XIcon } from "lucide-react";
import { useState } from "react";
import {
  pathCrossesToMany,
  refLabel,
  refTargetEntity,
  resolvePathEntity,
  resolveField,
  type CatalogIndex,
} from "./builder-state";
import { CatalogFieldTree, type FieldSelection } from "./catalog-field-tree";
import { FilterValueEditor } from "./filter-value-editor";

type MeasureFilterEditorProps = {
  index: CatalogIndex;
  ir: ReportIR;
  column: ReportColumnSpec;
  onUpdate: (column: ReportColumnSpec) => void;
};

const GROUP_OP_CHOICES = [
  { value: "and", label: "Match all" },
  { value: "or", label: "Match any" },
];

function filterCount(group: ReportFilterGroup | undefined): number {
  return group?.filters?.length ?? 0;
}

/**
 * A measure that aggregates related records can only be narrowed by fields of
 * those records — the server rejects anything else, because a parent-level
 * condition would let children through that the author meant to exclude.
 */
function ScopedFieldPicker({
  index,
  ir,
  column,
  label,
  onSelect,
}: {
  index: CatalogIndex;
  ir: ReportIR;
  column: ReportColumnSpec;
  label: string;
  onSelect: (ref: { path?: string[]; field: string }, fieldType: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const scoped = column.ref ? pathCrossesToMany(index, ir.entity, column.ref.path) : false;
  const scopeEntity = scoped ? resolvePathEntity(index, ir.entity, column.ref?.path) : undefined;

  const handleSelection = (selection: FieldSelection) => {
    onSelect(selection.ref, selection.field.type);
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm" className="h-7 max-w-40 justify-start font-normal">
            <span className="truncate">{label}</span>
          </Button>
        }
      />
      <PopoverContent className="h-72 w-64 p-2" align="start">
        {scoped && scopeEntity ? (
          <div className="flex h-full min-h-0 flex-col gap-1">
            <p className="text-2xs text-muted-foreground px-1 pb-1">
              {scopeEntity.pluralLabel} fields
            </p>
            <div className="min-h-0 flex-1 overflow-y-auto">
              {scopeEntity.fields
                .filter((field) => field.filterable && field.accessible)
                .map((field) => (
                  <button
                    key={field.key}
                    type="button"
                    className="hover:bg-accent hover:text-accent-foreground flex w-full items-center rounded-md px-2 py-1 text-left text-xs transition-colors"
                    onClick={() =>
                      handleSelection({
                        ref: { path: column.ref?.path, field: field.key },
                        field,
                        crossesToMany: true,
                      })
                    }
                  >
                    <span className="truncate">{field.label}</span>
                  </button>
                ))}
            </div>
          </div>
        ) : (
          <CatalogFieldTree
            index={index}
            entityKey={ir.entity}
            className="h-full"
            filterFields={(field) => field.filterable && field.accessible}
            onSelectField={handleSelection}
          />
        )}
      </PopoverContent>
    </Popover>
  );
}

export function MeasureFilterEditor({ index, ir, column, onUpdate }: MeasureFilterEditorProps) {
  const group = column.filter ?? { op: "and", filters: [] };
  const filters = group.filters ?? [];

  const setFilters = (next: ReportFieldFilter[], op = group.op) => {
    onUpdate({
      ...column,
      filter: next.length > 0 ? { op, filters: next } : undefined,
    });
  };

  const addFilter = () => {
    const scoped = column.ref ? pathCrossesToMany(index, ir.entity, column.ref.path) : false;
    const entity = scoped
      ? resolvePathEntity(index, ir.entity, column.ref?.path)
      : index.entities.get(ir.entity);
    const field = entity?.fields.find((f) => f.filterable && f.accessible);
    if (!field) return;

    setFilters([
      ...filters,
      {
        ref: { path: scoped ? column.ref?.path : undefined, field: field.key },
        operator: operatorsForFieldType(field.type)[0]?.value ?? "eq",
      },
    ]);
  };

  return (
    <div className="border-border bg-muted/30 flex flex-col gap-2 rounded-md border border-dashed p-2">
      <div className="flex items-center gap-2">
        <FilterIcon className="text-muted-foreground size-3" />
        <span className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
          Only count
        </span>
        {filterCount(column.filter) > 1 && (
          <Select
            value={group.op}
            onValueChange={(op) => {
              if (op === "and" || op === "or") setFilters(filters, op);
            }}
            items={GROUP_OP_CHOICES}
          >
            <SelectTrigger className="h-6 w-24">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="and">Match all</SelectItem>
              <SelectItem value="or">Match any</SelectItem>
            </SelectContent>
          </Select>
        )}
        <div className="flex-1" />
        <Button variant="ghost" size="sm" className="h-6" onClick={addFilter}>
          <PlusIcon className="size-3.5" />
          Condition
        </Button>
      </div>

      {filters.length === 0 ? (
        <p className="text-2xs text-muted-foreground">
          This measure counts every matching record. Add a condition to make it a subset — the other
          columns keep their full totals.
        </p>
      ) : (
        filters.map((filter, filterIndex) => {
          const field = resolveField(index, ir.entity, filter.ref);
          const operators = field ? operatorsForFieldType(field.type) : [];
          const update = (updated: ReportFieldFilter) =>
            setFilters(filters.map((f, i) => (i === filterIndex ? updated : f)));

          return (
            <div key={filterIndex} className="flex flex-wrap items-center gap-1.5">
              <ScopedFieldPicker
                index={index}
                ir={ir}
                column={column}
                label={refLabel(index, ir.entity, filter.ref)}
                onSelect={(ref, fieldType) =>
                  update({
                    ref,
                    operator: operatorsForFieldType(fieldType)[0]?.value ?? "eq",
                    value: undefined,
                  })
                }
              />
              <Select
                value={filter.operator}
                onValueChange={(operator) => {
                  if (operator) update({ ...filter, operator, value: undefined });
                }}
                items={operators}
              >
                <SelectTrigger className="h-7 w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {operators.map((op) => (
                    <SelectItem key={op.value} value={op.value}>
                      {op.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FilterValueEditor
                field={field}
                operator={filter.operator}
                value={filter.value}
                refEntityKey={refTargetEntity(index, ir.entity, filter.ref)}
                onChange={(value) => update({ ...filter, value })}
              />
              <Button
                variant="ghost"
                size="icon"
                className="size-6"
                onClick={() => setFilters(filters.filter((_, i) => i !== filterIndex))}
                aria-label="Remove condition"
              >
                <XIcon className="size-3.5" />
              </Button>
            </div>
          );
        })
      )}
    </div>
  );
}
