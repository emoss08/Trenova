import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
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
import { useRunReport } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { REPORT_FORMAT_CHOICES, type ReportParameterDef } from "@/types/report";
import { format } from "date-fns";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import {
  ReportRefAutocomplete,
  ReportRefMultiAutocomplete,
} from "./report-ref-autocomplete";

export type RunReportTarget =
  | { definitionId: string; cannedKey?: never }
  | { cannedKey: string; definitionId?: never };

type RunReportDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  target: RunReportTarget | null;
  reportName: string;
  defaultFormat: string;
  parameters: ReportParameterDef[];
};

function defaultParamValues(parameters: ReportParameterDef[]): Record<string, unknown> {
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

function coerceParamValue(param: ReportParameterDef, raw: string): unknown {
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
      return Number.isNaN(parsed.getTime()) ? undefined : Math.floor(parsed.getTime() / 1000);
    }
    default:
      return raw === "" ? undefined : raw;
  }
}

function paramInputValue(param: ReportParameterDef, value: unknown): string {
  if (value === undefined || value === null) return "";
  if (param.type === "epoch" && typeof value === "number") {
    return format(new Date(value * 1000), "yyyy-MM-dd");
  }
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

function multiListValue(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function ParameterField({
  param,
  value,
  onChange,
}: {
  param: ReportParameterDef;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const label = param.label || param.name;
  const inputId = `report-param-${param.name}`;
  const allowedValues = param.allowedValues ?? [];

  const fieldLabel = (
    <Label htmlFor={inputId}>
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
    return (
      <div className="flex flex-col gap-1.5">
        {fieldLabel}
        <div className="flex flex-wrap gap-x-4 gap-y-1.5">
          {allowedValues.map((allowed) => (
            <label key={allowed} className="flex items-center gap-1.5 text-sm">
              <Checkbox
                checked={selected.includes(allowed)}
                onCheckedChange={(checked) => {
                  const next = checked
                    ? [...selected, allowed]
                    : selected.filter((v) => v !== allowed);
                  onChange(
                    next.length > 0 ? next.map((v) => coerceParamValue(param, v) ?? v) : undefined,
                  );
                }}
              />
              {allowed}
            </label>
          ))}
        </div>
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
          items={allowedValues.map((allowed) => ({ value: allowed, label: allowed }))}
        >
          <SelectTrigger className="w-full" id={inputId}>
            <SelectValue placeholder="Select value" />
          </SelectTrigger>
          <SelectContent>
            {allowedValues.map((allowed) => (
              <SelectItem key={allowed} value={allowed}>
                {allowed}
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
          placeholder="Comma-separated values"
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
        type={inputType}
        step={param.type === "decimal" ? "any" : undefined}
        value={paramInputValue(param, value)}
        onChange={(event) => onChange(coerceParamValue(param, event.target.value))}
      />
    </div>
  );
}

export function RunReportDialog({
  open,
  onOpenChange,
  target,
  reportName,
  defaultFormat,
  parameters,
}: RunReportDialogProps) {
  const navigate = useNavigate();
  const runReport = useRunReport();
  const [format, setFormat] = useState(defaultFormat || "csv");
  const [paramValues, setParamValues] = useState<Record<string, unknown>>(() =>
    defaultParamValues(parameters),
  );

  useEffect(() => {
    if (open) {
      setFormat(defaultFormat || "csv");
      setParamValues(defaultParamValues(parameters));
    }
  }, [open, defaultFormat, parameters]);

  const missingRequired = useMemo(
    () =>
      parameters.filter(
        (param) =>
          param.required &&
          param.type !== "bool" &&
          (paramValues[param.name] === undefined || paramValues[param.name] === ""),
      ),
    [parameters, paramValues],
  );

  const handleSubmit = useCallback(() => {
    if (!target || missingRequired.length > 0) return;

    runReport.mutate(
      {
        definitionId: target.definitionId,
        cannedKey: target.cannedKey,
        format,
        params: Object.keys(paramValues).length > 0 ? paramValues : undefined,
      },
      {
        onSuccess: () => {
          onOpenChange(false);
          toast.success(`"${reportName}" queued for generation`, {
            action: {
              label: "View runs",
              onClick: () => navigate("/reports/runs"),
            },
          });
        },
        onError: (error) => {
          toast.error(graphQLErrorMessage(error, "Failed to queue the report run"));
        },
      },
    );
  }, [target, missingRequired, runReport, format, paramValues, onOpenChange, reportName, navigate]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xs">
        <DialogHeader>
          <DialogTitle>Run {reportName}</DialogTitle>
          <DialogDescription>
            The report is generated in the background — you&apos;ll be notified when it&apos;s ready
            to download.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          {parameters.map((param) => (
            <ParameterField
              key={param.name}
              param={param}
              value={paramValues[param.name]}
              onChange={(value) => setParamValues((prev) => ({ ...prev, [param.name]: value }))}
            />
          ))}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="report-run-format">Format</Label>
            <Select
              value={format}
              onValueChange={(value) => {
                if (value) setFormat(value);
              }}
              items={REPORT_FORMAT_CHOICES}
            >
              <SelectTrigger className="w-full" id="report-run-format">
                <SelectValue placeholder="Select format" />
              </SelectTrigger>
              <SelectContent>
                {REPORT_FORMAT_CHOICES.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={runReport.isPending || missingRequired.length > 0}
          >
            {runReport.isPending ? "Queuing..." : "Run Report"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
