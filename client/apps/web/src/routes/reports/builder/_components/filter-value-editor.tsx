import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import type { ReportCatalogField } from "@/lib/graphql/reports";
import { operatorChoice } from "@/types/report";
import {
  REPORT_REF_ENTITIES,
  ReportRefAutocomplete,
  ReportRefMultiAutocomplete,
} from "../../_components/report-ref-autocomplete";
import { cn } from "@trenova/shared/lib/utils";
import { multiSelectVariants } from "@/lib/variants/async-multi-select";
import { AutoCompleteDatePicker } from "@/components/fields/date-field/date-picker";
import { ChevronDownIcon, XIcon } from "lucide-react";

const BOOL_CHOICES = [
  { value: "true", label: "Yes" },
  { value: "false", label: "No" },
];

type FilterValueEditorProps = {
  field: ReportCatalogField | undefined;
  operator: string;
  value: unknown;
  onChange: (value: unknown) => void;
  /**
   * The catalog entity a reference field points at. With it, filtering on an
   * id offers real records instead of asking for a raw identifier.
   */
  refEntityKey?: string;
};

/**
 * The app's multi-select is remote-driven, so an enum with a fixed value list
 * cannot use it directly — this matches its trigger and badge treatment via the
 * shared `multiSelectVariants` tokens so the two read as one control.
 */
function EnumMultiSelect({
  field,
  value,
  onChange,
}: {
  field: ReportCatalogField;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const selected = Array.isArray(value) ? (value as string[]) : [];
  const byValue = new Map(field.enumValues.map((choice) => [choice.value, choice.label]));

  const remove = (next: string) => onChange(selected.filter((v) => v !== next));

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            role="combobox"
            className={cn(
              "border-muted-foreground/20 bg-muted h-auto min-h-7 w-full cursor-pointer gap-2 rounded-md px-2 py-1 font-normal",
              "hover:bg-muted-foreground/20 justify-between whitespace-nowrap",
              "data-pressed:border-brand data-pressed:ring-brand/20 data-pressed:ring-4 data-pressed:outline-hidden",
              "transition-all duration-200 ease-in-out [&_svg]:size-3 [&_svg]:shrink-0",
            )}
          >
            {selected.length > 0 ? (
              <span className="flex min-w-0 flex-wrap items-center gap-1">
                {selected.map((entry) => (
                  <span
                    key={entry}
                    className={cn("min-w-0", multiSelectVariants({ variant: "default" }))}
                  >
                    <span className="min-w-0 truncate">{byValue.get(entry) ?? entry}</span>
                    <XIcon
                      className="size-4 shrink-0 cursor-pointer"
                      onClick={(event) => {
                        event.stopPropagation();
                        remove(entry);
                      }}
                    />
                  </span>
                ))}
              </span>
            ) : (
              <span className="text-muted-foreground">Select values...</span>
            )}
            <ChevronDownIcon className="text-muted-foreground size-3.5 shrink-0" />
          </Button>
        }
      />
      <PopoverContent className="max-h-64 w-56 overflow-y-auto p-2">
        <div className="flex flex-col gap-1.5">
          {field.enumValues.map((choice) => (
            <label key={choice.value} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={selected.includes(choice.value)}
                onCheckedChange={(checked) =>
                  onChange(
                    checked
                      ? [...selected, choice.value]
                      : selected.filter((v) => v !== choice.value),
                  )
                }
              />
              {choice.label}
            </label>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

/** Epoch columns get the app's date picker rather than a bare date input. */
function EpochValue({ value, onChange }: { value: unknown; onChange: (value: unknown) => void }) {
  return (
    <AutoCompleteDatePicker
      date={typeof value === "number" ? new Date(value * 1000) : undefined}
      setDate={(date) => onChange(date ? Math.floor(date.getTime() / 1000) : undefined)}
      clearable
    />
  );
}

export function FilterValueEditor({
  field,
  operator,
  value,
  onChange,
  refEntityKey,
}: FilterValueEditorProps) {
  const choice = operatorChoice(operator);
  if (!choice || !choice.requiresValue || !field) return null;

  if (operator === "daterange") {
    const range = Array.isArray(value) ? (value as unknown[]) : [undefined, undefined];
    return (
      <div className="flex items-center gap-1">
        <EpochValue value={range[0]} onChange={(next) => onChange([next, range[1]])} />
        <span className="text-muted-foreground text-xs">to</span>
        <EpochValue value={range[1]} onChange={(next) => onChange([range[0], next])} />
      </div>
    );
  }

  if (operator === "lastndays" || operator === "nextndays") {
    return (
      <Input
        className="h-7"
        type="number"
        min={1}
        value={typeof value === "number" ? String(value) : ""}
        placeholder="Days"
        onChange={(event) => {
          const parsed = Number.parseInt(event.target.value, 10);
          onChange(Number.isNaN(parsed) ? undefined : parsed);
        }}
      />
    );
  }

  const refEntity = refEntityKey && REPORT_REF_ENTITIES[refEntityKey] ? refEntityKey : undefined;

  if (choice.multiValue) {
    if (field.type === "enum" && field.enumValues.length > 0) {
      return <EnumMultiSelect field={field} value={value} onChange={onChange} />;
    }
    if (refEntity) {
      return (
        <ReportRefMultiAutocomplete
          entityKey={refEntity}
          values={Array.isArray(value) ? (value as unknown[]).map(String) : []}
          onChange={(values) => onChange(values.length > 0 ? values : undefined)}
        />
      );
    }
    const joined = Array.isArray(value) ? (value as unknown[]).join(", ") : "";
    return (
      <Input
        className="h-7"
        value={joined}
        placeholder="Comma-separated values"
        onChange={(event) => {
          const values = event.target.value
            .split(",")
            .map((v) => v.trim())
            .filter(Boolean);
          onChange(
            field.type === "int"
              ? values.map((v) => Number.parseInt(v, 10)).filter((v) => !Number.isNaN(v))
              : values,
          );
        }}
      />
    );
  }

  if (refEntity) {
    return (
      <ReportRefAutocomplete
        entityKey={refEntity}
        value={typeof value === "string" ? value : ""}
        onChange={(next) => onChange(next || undefined)}
      />
    );
  }

  if (field.type === "enum" && field.enumValues.length > 0) {
    return (
      <Select
        value={typeof value === "string" ? value : ""}
        onValueChange={(next) => {
          if (next) onChange(next);
        }}
        items={field.enumValues.map((enumValue) => ({
          value: enumValue.value,
          label: enumValue.label,
        }))}
      >
        <SelectTrigger className="h-7">
          <SelectValue placeholder="Select value" />
        </SelectTrigger>
        <SelectContent>
          {field.enumValues.map((enumValue) => (
            <SelectItem key={enumValue.value} value={enumValue.value}>
              {enumValue.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }

  if (field.type === "bool") {
    return (
      <Select
        value={value === true ? "true" : value === false ? "false" : ""}
        onValueChange={(next) => {
          if (next) onChange(next === "true");
        }}
        items={BOOL_CHOICES}
      >
        <SelectTrigger className="h-7">
          <SelectValue placeholder="Select" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="true">Yes</SelectItem>
          <SelectItem value="false">No</SelectItem>
        </SelectContent>
      </Select>
    );
  }

  if (field.type === "epoch") {
    return <EpochValue value={value} onChange={onChange} />;
  }

  if (field.type === "int" || field.type === "decimal") {
    return (
      <Input
        className="h-7"
        type="number"
        step={field.type === "decimal" ? "any" : undefined}
        value={typeof value === "number" ? String(value) : ""}
        onChange={(event) => {
          const raw = event.target.value;
          if (raw === "") {
            onChange(undefined);
            return;
          }
          const parsed = field.type === "int" ? Number.parseInt(raw, 10) : Number.parseFloat(raw);
          onChange(Number.isNaN(parsed) ? undefined : parsed);
        }}
      />
    );
  }

  return (
    <Input
      className="h-7"
      value={typeof value === "string" ? value : ""}
      placeholder="Value"
      onChange={(event) => onChange(event.target.value || undefined)}
    />
  );
}
