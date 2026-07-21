"use no memo";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  FORMAT_RULE_COLOR_SWATCHES,
  generateFilterId,
  getOperatorLabel,
  operatorRequiresValue,
  stringifyUnknown,
} from "@/lib/data-table";
import { cn } from "@/lib/utils";
import type { FilterVariant } from "@/types/data-table";
import type {
  FormatRuleColor,
  FormatRuleOperator,
  TableFormatRule,
} from "@/types/table-configuration";
import type { SelectOption } from "@/types/fields";
import type { ColumnDef } from "@tanstack/react-table";
import { PaintbrushIcon, PlusIcon, TrashIcon } from "lucide-react";
import { useMemo } from "react";

type FormatColumn = {
  field: string;
  label: string;
  filterType: FilterVariant;
  filterOptions?: SelectOption[];
};

type DataTableFormatBuilderProps<TData> = {
  columns: ColumnDef<TData>[];
  rules: TableFormatRule[];
  onRulesChange: (rules: TableFormatRule[]) => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

const RULE_COLORS = Object.keys(FORMAT_RULE_COLOR_SWATCHES) as FormatRuleColor[];

function operatorsForVariant(variant: FilterVariant): FormatRuleOperator[] {
  switch (variant) {
    case "number":
      return ["eq", "ne", "gt", "gte", "lt", "lte", "isnull", "isnotnull"];
    case "select":
      return ["eq", "ne", "isnull", "isnotnull"];
    case "boolean":
      return ["eq"];
    case "date":
      return ["isnull", "isnotnull"];
    default:
      return ["eq", "ne", "contains", "isnull", "isnotnull"];
  }
}

function ColorSwatchPicker({
  value,
  onChange,
}: {
  value: FormatRuleColor;
  onChange: (color: FormatRuleColor) => void;
}) {
  return (
    <div className="flex shrink-0 items-center gap-1.5" role="radiogroup" aria-label="Highlight color">
      {RULE_COLORS.map((color) => (
        <button
          key={color}
          type="button"
          role="radio"
          aria-checked={value === color}
          aria-label={color}
          onClick={() => onChange(color)}
          className={cn(
            "size-4 cursor-pointer rounded-full transition-transform hover:scale-110",
            FORMAT_RULE_COLOR_SWATCHES[color],
            value === color && "ring-2 ring-ring ring-offset-1 ring-offset-background",
          )}
        />
      ))}
    </div>
  );
}

function RuleValueInput({
  column,
  operator,
  value,
  onChange,
}: {
  column: FormatColumn | undefined;
  operator: FormatRuleOperator;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  if (!column || !operatorRequiresValue(operator)) {
    return null;
  }

  if (column.filterType === "select" && column.filterOptions) {
    const stringValue = stringifyUnknown(value);
    const selectedLabel = column.filterOptions.find(
      (o) => stringifyUnknown(o.value) === stringValue,
    )?.label;
    return (
      <Select value={stringValue} onValueChange={(val) => onChange(val)}>
        <SelectTrigger className="min-w-0 flex-1">
          <SelectValue>{selectedLabel ?? "Select..."}</SelectValue>
        </SelectTrigger>
        <SelectContent className="w-auto">
          <SelectGroup>
            {column.filterOptions.map((option) => (
              <SelectItem key={String(option.value)} value={String(option.value)}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    );
  }

  if (column.filterType === "boolean") {
    return (
      <Select
        value={value === true ? "true" : value === false ? "false" : ""}
        onValueChange={(val) => onChange(val === "true")}
      >
        <SelectTrigger className="min-w-0 flex-1">
          <SelectValue>{value === true ? "Yes" : value === false ? "No" : "Select..."}</SelectValue>
        </SelectTrigger>
        <SelectContent className="w-auto">
          <SelectGroup>
            <SelectItem value="true">Yes</SelectItem>
            <SelectItem value="false">No</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
    );
  }

  return (
    <Input
      type={column.filterType === "number" ? "number" : "text"}
      className="h-7 min-w-0 flex-1 text-sm"
      value={stringifyUnknown(value)}
      onChange={(e) =>
        onChange(
          column.filterType === "number"
            ? e.target.value === ""
              ? null
              : Number(e.target.value)
            : e.target.value,
        )
      }
      placeholder="Value"
    />
  );
}

export default function DataTableFormatBuilder<TData>({
  columns,
  rules,
  onRulesChange,
  open,
  onOpenChange,
}: DataTableFormatBuilderProps<TData>) {
  const formatColumns = useMemo<FormatColumn[]>(
    () =>
      columns
        .filter((col) => col.meta?.filterable && col.meta.apiField)
        .map((col) => ({
          field: col.meta!.apiField!,
          label: col.meta!.label ?? col.meta!.apiField!,
          filterType: (col.meta!.filterType ?? "text") as FilterVariant,
          filterOptions: col.meta!.filterOptions,
        })),
    [columns],
  );

  const addRule = () => {
    const first = formatColumns[0];
    if (!first) return;
    onRulesChange([
      ...rules,
      {
        id: generateFilterId(),
        field: first.field,
        operator: operatorsForVariant(first.filterType)[0],
        value: null,
        color: "amber",
      },
    ]);
  };

  const updateRule = (id: string, patch: Partial<TableFormatRule>) => {
    onRulesChange(rules.map((rule) => (rule.id === id ? { ...rule, ...patch } : rule)));
  };

  const removeRule = (id: string) => {
    onRulesChange(rules.filter((rule) => rule.id !== id));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Conditional formatting</DialogTitle>
          <DialogDescription>
            Highlight rows that match a condition — the first matching rule wins.
          </DialogDescription>
        </DialogHeader>
        {rules.length === 0 ? (
          <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-center">
            <PaintbrushIcon className="size-4 text-muted-foreground" />
            <h3 className="text-sm font-medium">No formatting rules</h3>
            <p className="max-w-72 text-xs text-muted-foreground">
              Tint rows that need attention — for example, unassigned or late records.
            </p>
            <Button size="sm" onClick={addRule} disabled={formatColumns.length === 0}>
              Add Rule
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {rules.map((rule, index) => {
              const column = formatColumns.find((c) => c.field === rule.field);
              const operators = operatorsForVariant(column?.filterType ?? "text");
              const needsValue = operatorRequiresValue(rule.operator);
              return (
                <div
                  key={rule.id}
                  className="flex flex-col gap-2 rounded-lg border border-border bg-card p-2.5"
                >
                  <div className="flex items-center gap-2">
                    <span className="w-11 shrink-0 text-xs font-medium text-muted-foreground">
                      {index === 0 ? "When" : "Else if"}
                    </span>
                    <Select
                      value={rule.field}
                      onValueChange={(val) => {
                        const nextColumn = formatColumns.find((c) => c.field === val);
                        if (!nextColumn) return;
                        updateRule(rule.id, {
                          field: nextColumn.field,
                          operator: operatorsForVariant(nextColumn.filterType)[0],
                          value: null,
                        });
                      }}
                    >
                      <SelectTrigger className="min-w-0 flex-1">
                        <SelectValue>{column?.label ?? rule.field}</SelectValue>
                      </SelectTrigger>
                      <SelectContent className="w-auto">
                        <SelectGroup>
                          {formatColumns.map((col) => (
                            <SelectItem key={col.field} value={col.field}>
                              {col.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <Select
                      value={rule.operator}
                      onValueChange={(val) =>
                        updateRule(rule.id, { operator: val as FormatRuleOperator })
                      }
                    >
                      <SelectTrigger className="min-w-0 flex-1">
                        <SelectValue>{getOperatorLabel(rule.operator)}</SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {operators.map((op) => (
                            <SelectItem key={op} value={op}>
                              {getOperatorLabel(op)}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-11 shrink-0" />
                    <RuleValueInput
                      column={column}
                      operator={rule.operator}
                      value={rule.value}
                      onChange={(value) => updateRule(rule.id, { value })}
                    />
                    {!needsValue && <div className="min-w-0 flex-1" />}
                    <ColorSwatchPicker
                      value={rule.color}
                      onChange={(color) => updateRule(rule.id, { color })}
                    />
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-muted-foreground hover:text-destructive"
                      aria-label="Remove rule"
                      onClick={() => removeRule(rule.id)}
                    >
                      <TrashIcon className="size-3.5" />
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
        <DialogFooter className="sm:justify-between">
          {rules.length > 0 ? (
            <div className="flex items-center gap-1">
              <Button variant="outline" size="sm" onClick={addRule}>
                <PlusIcon className="size-3.5" />
                Add Rule
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="text-muted-foreground"
                onClick={() => onRulesChange([])}
              >
                Clear Rules
              </Button>
            </div>
          ) : (
            <span />
          )}
          <Button size="sm" onClick={() => onOpenChange(false)}>
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
