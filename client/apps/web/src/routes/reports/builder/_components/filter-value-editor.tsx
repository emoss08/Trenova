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
import { format } from "date-fns";
import { ChevronDownIcon } from "lucide-react";

const BOOL_CHOICES = [
  { value: "true", label: "Yes" },
  { value: "false", label: "No" },
];

type FilterValueEditorProps = {
  field: ReportCatalogField | undefined;
  operator: string;
  value: unknown;
  onChange: (value: unknown) => void;
};

function toDateInput(value: unknown): string {
  if (typeof value !== "number") return "";
  return format(new Date(value * 1000), "yyyy-MM-dd");
}

function fromDateInput(raw: string): number | undefined {
  if (!raw) return undefined;
  const parsed = new Date(`${raw}T00:00:00`);
  return Number.isNaN(parsed.getTime()) ? undefined : Math.floor(parsed.getTime() / 1000);
}

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

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button variant="outline" className="h-7 justify-between font-normal">
            {selected.length > 0 ? `${selected.length} selected` : "Select values"}
            <ChevronDownIcon className="size-3.5" />
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

export function FilterValueEditor({ field, operator, value, onChange }: FilterValueEditorProps) {
  const choice = operatorChoice(operator);
  if (!choice || !choice.requiresValue || !field) return null;

  if (operator === "daterange") {
    const range = Array.isArray(value) ? (value as unknown[]) : [undefined, undefined];
    return (
      <div className="flex items-center gap-1">
        <Input
          className="h-7"
          type="date"
          value={toDateInput(range[0])}
          onChange={(event) => onChange([fromDateInput(event.target.value), range[1]])}
        />
        <span className="text-xs text-muted-foreground">to</span>
        <Input
          className="h-7"
          type="date"
          value={toDateInput(range[1])}
          onChange={(event) => onChange([range[0], fromDateInput(event.target.value)])}
        />
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

  if (choice.multiValue) {
    if (field.type === "enum" && field.enumValues.length > 0) {
      return <EnumMultiSelect field={field} value={value} onChange={onChange} />;
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
    return (
      <Input
        className="h-7"
        type="date"
        value={toDateInput(value)}
        onChange={(event) => onChange(fromDateInput(event.target.value))}
      />
    );
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
