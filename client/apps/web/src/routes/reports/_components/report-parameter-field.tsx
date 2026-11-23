import { useT } from "@trenova/shared/i18n/use-t";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Switch } from "@trenova/shared/components/ui/switch";
import { codeWords } from "@trenova/shared/lib/code-words";
import { cn } from "@trenova/shared/lib/utils";
import { parameterChoices, type ReportParameterOption } from "@/lib/report-parameter-options";
import type { ReportParameterDef } from "@/types/report";
import { ReportRefAutocomplete, ReportRefMultiAutocomplete } from "./report-ref-autocomplete";
import { fromUserWallClock, toISODateString, toUserWallClock } from "@trenova/shared/lib/date";

export function defaultParamValues(parameters: ReportParameterDef[]): Record<string, unknown> {
  const values: Record<string, unknown> = {};
  for (const param of parameters) {
    if (param.default !== undefined && param.default !== null) {
      values[param.name] = param.default;
    } else if (param.type === "bool") {
      values[param.name] = false;
    }
  }
  return values;
}

export function coerceParamValue(param: ReportParameterDef, raw: string): unknown {
  switch (param.type) {
    case "int": {
      const parsed = Number.parseInt(raw, 10);
      return Number.isNaN(parsed) ? undefined : parsed;
    }
    case "decimal": {
      const parsed = Number.parseFloat(raw);
      return Number.isNaN(parsed) ? undefined : parsed;
    }
    case "epoch": {
      if (!raw) return undefined;
      const parsed = new Date(`${raw}T00:00:00`);
      return fromUserWallClock(parsed);
    }
    default:
      return raw === "" ? undefined : raw;
  }
}

export function paramInputValue(param: ReportParameterDef, value: unknown): string {
  if (value === undefined || value === null) return "";
  if (param.type === "epoch" && typeof value === "number") {
    const wallClock = toUserWallClock(value);
    return wallClock ? toISODateString(wallClock) : "";
  }
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

export function missingRequiredParams(
  parameters: ReportParameterDef[],
  values: Record<string, unknown>,
): ReportParameterDef[] {
  return parameters.filter(
    (param) =>
      param.required &&
      param.type !== "bool" &&
      (values[param.name] === undefined || values[param.name] === ""),
  );
}

function multiListValue(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

export function ParameterField({
  param,
  value,
  onChange,
  compact,
  options,
}: {
  param: ReportParameterDef;
  value: unknown;
  onChange: (value: unknown) => void;
  compact?: boolean;
  /** The parameter's allowed values with their names; without them each code reads in words. */
  options?: readonly ReportParameterOption[];
}) {
  const t = useT();

  const label = param.label || param.name;
  const inputId = `report-param-${param.name}`;
  const allowedValues = param.allowedValues ?? [];
  const choices = options ?? parameterChoices(param);
  const controlHeight = compact ? "h-7" : undefined;

  const fieldLabel = (
    <Label htmlFor={inputId} className={compact ? "text-muted-foreground text-xs" : undefined}>
      {label}
      {param.required && <span className="text-destructive"> *</span>}
    </Label>
  );

  if (param.type === "bool") {
    return (
      <div className="flex items-center justify-between gap-2">
        {fieldLabel}
        <Switch id={inputId} checked={Boolean(value)} onCheckedChange={onChange} />
      </div>
    );
  }

  if (param.type === "ref") {
    if (param.multi) {
      return (
        <div className="flex flex-col gap-1.5">
          {fieldLabel}
          <ReportRefMultiAutocomplete
            entityKey={param.refEntity ?? ""}
            values={multiListValue(value).map(String)}
            onChange={(values) => onChange(values.length > 0 ? values : undefined)}
          />
        </div>
      );
    }
    return (
      <div className="flex flex-col gap-1.5">
        {fieldLabel}
        <ReportRefAutocomplete
          entityKey={param.refEntity ?? ""}
          value={typeof value === "string" ? value : ""}
          onChange={(next) => onChange(next || undefined)}
        />
      </div>
    );
  }

  if (allowedValues.length > 0 && param.multi) {
    const selected = multiListValue(value).map(String);
    const labelOf = new Map(choices.map((choice) => [choice.value, choice.label]));
    const order = new Map(choices.map((choice, index) => [choice.value, index]));
    return (
      <div className="flex flex-col gap-1.5">
        {fieldLabel}
        <Select
          multiple
          value={selected}
          onValueChange={(next: string[]) => {
            const sorted = [...next].sort(
              (a, b) => (order.get(a) ?? order.size) - (order.get(b) ?? order.size),
            );
            onChange(
              sorted.length > 0 ? sorted.map((v) => coerceParamValue(param, v) ?? v) : undefined,
            );
          }}
          items={choices}
        >
          <SelectTrigger className={cn("w-full", controlHeight)} id={inputId}>
            <SelectValue>
              {selected.length === 0 ? (
                <span className="text-muted-foreground">{t("Select values")}</span>
              ) : selected.length === 1 ? (
                (labelOf.get(selected[0]) ?? codeWords(selected[0]))
              ) : (
                t("{0} Selected", selected.length)
              )}
            </SelectValue>
          </SelectTrigger>
          <SelectContent className="max-h-64 w-auto min-w-(--anchor-width)">
            {choices.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    );
  }

  if (allowedValues.length > 0) {
    return (
      <div className="flex flex-col gap-1.5">
        {fieldLabel}
        <Select
          value={paramInputValue(param, value)}
          onValueChange={(next) => {
            if (next) onChange(coerceParamValue(param, next));
          }}
          items={choices}
        >
          <SelectTrigger className={cn("w-full", controlHeight)} id={inputId}>
            <SelectValue placeholder={t("Select value")} />
          </SelectTrigger>
          <SelectContent className="max-h-64 w-auto min-w-(--anchor-width)">
            {choices.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    );
  }

  if (param.multi) {
    const joined = multiListValue(value)
      .map((v) => paramInputValue(param, v))
      .join(", ");
    return (
      <div className="flex flex-col gap-1.5">
        {fieldLabel}
        <Input
          id={inputId}
          className={controlHeight}
          placeholder={t("Comma-separated values")}
          defaultValue={joined}
          onChange={(event) => {
            const values = event.target.value
              .split(",")
              .map((v) => v.trim())
              .filter(Boolean)
              .map((v) => coerceParamValue(param, v))
              .filter((v) => v !== undefined);
            onChange(values.length > 0 ? values : undefined);
          }}
        />
      </div>
    );
  }

  const inputType =
    param.type === "int" || param.type === "decimal"
      ? "number"
      : param.type === "epoch"
        ? "date"
        : "text";

  return (
    <div className="flex flex-col gap-1.5">
      {fieldLabel}
      <Input
        id={inputId}
        className={controlHeight}
        type={inputType}
        step={param.type === "decimal" ? "any" : undefined}
        value={paramInputValue(param, value)}
        onChange={(event) => onChange(coerceParamValue(param, event.target.value))}
      />
    </div>
  );
}
