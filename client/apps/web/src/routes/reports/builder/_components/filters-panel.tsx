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
  operatorChoice,
  operatorsForFieldType,
  type ReportFieldFilter,
  type ReportFilterGroup,
  type ReportIR,
} from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { refLabel, resolveField, type CatalogIndex } from "./builder-state";
import { CatalogFieldTree, type FieldSelection } from "./catalog-field-tree";
import { FilterValueEditor } from "./filter-value-editor";

const MAX_GROUP_DEPTH = 3;

const GROUP_OP_CHOICES = [
  { value: "and", label: "Match all" },
  { value: "or", label: "Match any" },
];

type FiltersPanelProps = {
  index: CatalogIndex;
  ir: ReportIR;
  group: ReportFilterGroup | null | undefined;
  onChange: (group: ReportFilterGroup) => void;
};

function FieldPickerButton({
  index,
  ir,
  label,
  onSelect,
}: {
  index: CatalogIndex;
  ir: ReportIR;
  label: string;
  onSelect: (selection: FieldSelection) => void;
}) {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm" className="h-7 max-w-48 justify-start font-normal">
            <span className="truncate">{label}</span>
          </Button>
        }
      />
      <PopoverContent className="h-80 w-72 p-2" align="start">
        <CatalogFieldTree
          index={index}
          entityKey={ir.entity}
          className="h-full"
          filterFields={(field) => field.filterable && field.accessible}
          onSelectField={(selection) => {
            onSelect(selection);
            setOpen(false);
          }}
        />
      </PopoverContent>
    </Popover>
  );
}

function FilterRow({
  index,
  ir,
  filter,
  onUpdate,
  onRemove,
}: {
  index: CatalogIndex;
  ir: ReportIR;
  filter: ReportFieldFilter;
  onUpdate: (filter: ReportFieldFilter) => void;
  onRemove: () => void;
}) {
  const field = resolveField(index, ir.entity, filter.ref);
  const operators = field ? operatorsForFieldType(field.type) : [];
  const parameters = ir.parameters ?? [];
  const operatorMeta = operatorChoice(filter.operator);
  const compatibleParams = field
    ? parameters.filter(
        (param) =>
          (param.type === field.type || field.type === "enum") &&
          Boolean(param.multi) === Boolean(operatorMeta?.multiValue),
      )
    : [];
  const paramBindingChoices = [
    { value: "__value__", label: "Fixed value" },
    ...compatibleParams.map((param) => ({
      value: param.name,
      label: `Param: ${param.label || param.name}`,
    })),
  ];

  return (
    <div className="flex flex-wrap items-center gap-1.5 rounded-md border border-border p-2">
      <FieldPickerButton
        index={index}
        ir={ir}
        label={refLabel(index, ir.entity, filter.ref)}
        onSelect={(selection) => {
          const nextOperators = operatorsForFieldType(selection.field.type);
          onUpdate({
            ref: selection.ref,
            operator: nextOperators[0]?.value ?? "eq",
            value: undefined,
            param: undefined,
          });
        }}
      />
      <Select
        value={filter.operator}
        onValueChange={(operator) => {
          if (!operator) return;
          onUpdate({ ...filter, operator, value: undefined });
        }}
        items={operators}
      >
        <SelectTrigger className="h-7 w-40">
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
      {compatibleParams.length > 0 && (
        <Select
          value={filter.param ?? "__value__"}
          onValueChange={(next) => {
            if (!next) return;
            if (next === "__value__") {
              onUpdate({ ...filter, param: undefined });
            } else {
              onUpdate({ ...filter, param: next, value: undefined });
            }
          }}
          items={paramBindingChoices}
        >
          <SelectTrigger className="h-7 w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__value__">Fixed value</SelectItem>
            {compatibleParams.map((param) => (
              <SelectItem key={param.name} value={param.name}>
                Param: {param.label || param.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {!filter.param && (
        <FilterValueEditor
          field={field}
          operator={filter.operator}
          value={filter.value}
          onChange={(value) => onUpdate({ ...filter, value })}
        />
      )}
      <Button
        variant="ghost"
        size="icon"
        className="size-6"
        onClick={onRemove}
        aria-label="Remove filter"
      >
        <XIcon className="size-3.5" />
      </Button>
    </div>
  );
}

function GroupEditor({
  index,
  ir,
  group,
  onChange,
  onRemove,
  depth,
}: {
  index: CatalogIndex;
  ir: ReportIR;
  group: ReportFilterGroup;
  onChange: (group: ReportFilterGroup) => void;
  onRemove?: () => void;
  depth: number;
}) {
  const filters = group.filters ?? [];
  const groups = group.groups ?? [];

  const defaultFilter = (): ReportFieldFilter | null => {
    const entity = index.entities.get(ir.entity);
    const firstField = entity?.fields.find((field) => field.filterable && field.accessible);
    if (!firstField) return null;
    return {
      ref: { field: firstField.key },
      operator: operatorsForFieldType(firstField.type)[0]?.value ?? "eq",
    };
  };

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border p-2">
      <div className="flex items-center gap-2">
        <Select
          value={group.op}
          onValueChange={(op) => {
            if (op === "and" || op === "or") onChange({ ...group, op });
          }}
          items={GROUP_OP_CHOICES}
        >
          <SelectTrigger className="h-7 w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="and">Match all</SelectItem>
            <SelectItem value="or">Match any</SelectItem>
          </SelectContent>
        </Select>
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="sm"
          className="h-7"
          onClick={() => {
            const filter = defaultFilter();
            if (filter) onChange({ ...group, filters: [...filters, filter] });
          }}
        >
          <PlusIcon className="size-3.5" />
          Condition
        </Button>
        {depth < MAX_GROUP_DEPTH && (
          <Button
            variant="ghost"
            size="sm"
            className="h-7"
            onClick={() => onChange({ ...group, groups: [...groups, { op: "and", filters: [] }] })}
          >
            <PlusIcon className="size-3.5" />
            Group
          </Button>
        )}
        {onRemove && (
          <Button
            variant="ghost"
            size="icon"
            className="size-6"
            onClick={onRemove}
            aria-label="Remove group"
          >
            <XIcon className="size-3.5" />
          </Button>
        )}
      </div>
      {filters.length === 0 && groups.length === 0 && (
        <p className="px-1 text-xs text-muted-foreground">No conditions in this group.</p>
      )}
      {filters.map((filter, filterIndex) => (
        <FilterRow
          key={filterIndex}
          index={index}
          ir={ir}
          filter={filter}
          onUpdate={(updated) =>
            onChange({
              ...group,
              filters: filters.map((f, i) => (i === filterIndex ? updated : f)),
            })
          }
          onRemove={() =>
            onChange({ ...group, filters: filters.filter((_, i) => i !== filterIndex) })
          }
        />
      ))}
      {groups.map((nested, groupIndex) => (
        <GroupEditor
          key={groupIndex}
          index={index}
          ir={ir}
          group={nested}
          depth={depth + 1}
          onChange={(updated) =>
            onChange({
              ...group,
              groups: groups.map((g, i) => (i === groupIndex ? updated : g)),
            })
          }
          onRemove={() => onChange({ ...group, groups: groups.filter((_, i) => i !== groupIndex) })}
        />
      ))}
    </div>
  );
}

export function FiltersPanel({ index, ir, group, onChange }: FiltersPanelProps) {
  return (
    <GroupEditor
      index={index}
      ir={ir}
      group={group ?? { op: "and", filters: [] }}
      onChange={onChange}
      depth={1}
    />
  );
}
