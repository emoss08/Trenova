import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { REPORT_AGGREGATION_LABELS, type ReportIR, type ReportPivotSpec } from "@/types/report";
import { XIcon } from "lucide-react";
import { useState } from "react";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import {
  aggregateColumns,
  columnDisplayLabel,
  refLabel,
  resolveField,
  type CatalogIndex,
} from "./builder-state";
import { CatalogFieldTree } from "./catalog-field-tree";

type PivotPanelProps = {
  index: CatalogIndex;
  ir: ReportIR;
  onChange: (pivot: ReportPivotSpec | null) => void;
};

export function PivotPanel({ index, ir, onChange }: PivotPanelProps) {
  const [fieldPickerOpen, setFieldPickerOpen] = useState(false);
  const measures = aggregateColumns(ir);
  const pivot = ir.pivot ?? null;
  const pivotField = pivot ? resolveField(index, ir.entity, pivot.ref) : undefined;

  if (measures.length === 0) {
    return (
      <p className="px-2 py-4 text-center text-sm text-muted-foreground">
        Pivots spread measures across the values of a dimension — add a measure column first.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <Popover open={fieldPickerOpen} onOpenChange={setFieldPickerOpen}>
          <PopoverTrigger
            render={
              <Button variant="outline" size="sm" className="h-7 justify-start font-normal">
                {pivot ? refLabel(index, ir.entity, pivot.ref) : "Choose pivot field"}
              </Button>
            }
          />
          <PopoverContent className="h-80 w-72 p-2" align="start">
            <CatalogFieldTree
              index={index}
              entityKey={ir.entity}
              className="h-full"
              filterFields={(field, crossesToMany) =>
                field.groupable && field.accessible && !crossesToMany
              }
              onSelectField={(selection) => {
                const enumValues = selection.field.enumValues.map((value) => value.value);
                onChange({
                  ref: selection.ref,
                  values: enumValues.slice(0, 10),
                  measureIds: measures.map((column) => column.id),
                  includeOther: enumValues.length > 10,
                });
                setFieldPickerOpen(false);
              }}
            />
          </PopoverContent>
        </Popover>
        {pivot && (
          <Button
            variant="ghost"
            size="icon"
            className="size-6"
            onClick={() => onChange(null)}
            aria-label="Remove pivot"
          >
            <XIcon className="size-3.5" />
          </Button>
        )}
      </div>
      {pivot && (
        <>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs text-muted-foreground">Pivot Values</Label>
            {pivotField && pivotField.enumValues.length > 0 ? (
              <div className="flex flex-col gap-1">
                {pivotField.enumValues.map((enumValue) => (
                  <label key={enumValue.value} className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={pivot.values.includes(enumValue.value)}
                      onCheckedChange={(checked) =>
                        onChange({
                          ...pivot,
                          values: checked
                            ? [...pivot.values, enumValue.value]
                            : pivot.values.filter((v) => v !== enumValue.value),
                        })
                      }
                    />
                    {enumValue.label}
                  </label>
                ))}
              </div>
            ) : (
              <Input
                className="h-7"
                placeholder="Comma-separated values"
                value={pivot.values.join(", ")}
                onChange={(event) =>
                  onChange({
                    ...pivot,
                    values: event.target.value
                      .split(",")
                      .map((v) => v.trim())
                      .filter(Boolean),
                  })
                }
              />
            )}
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs text-muted-foreground">Measures to Pivot</Label>
            <div className="flex flex-col gap-1">
              {measures.map((column) => (
                <label key={column.id} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={pivot.measureIds.includes(column.id)}
                    onCheckedChange={(checked) =>
                      onChange({
                        ...pivot,
                        measureIds: checked
                          ? [...pivot.measureIds, column.id]
                          : pivot.measureIds.filter((id) => id !== column.id),
                      })
                    }
                  />
                  {column.agg ? `${REPORT_AGGREGATION_LABELS[column.agg]} of ` : ""}
                  {columnDisplayLabel(index, ir, column)}
                </label>
              ))}
            </div>
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="pivot-include-other" className="text-xs text-muted-foreground">
              Include &quot;Other&quot; bucket
            </Label>
            <Switch
              id="pivot-include-other"
              checked={pivot.includeOther ?? false}
              onCheckedChange={(includeOther) => onChange({ ...pivot, includeOther })}
            />
          </div>
        </>
      )}
    </div>
  );
}
