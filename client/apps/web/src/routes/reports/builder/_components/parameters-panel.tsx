import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  REPORT_PARAMETER_TYPE_CHOICES,
  type ReportIR,
  type ReportParameterDef,
} from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";
import { REPORT_REF_ENTITY_CHOICES } from "../../_components/report-ref-autocomplete";

type ParametersPanelProps = {
  ir: ReportIR;
  onChange: (parameters: ReportParameterDef[], rename?: { from: string; to: string }) => void;
};

function coerceDefault(param: ReportParameterDef, raw: string): unknown {
  if (raw === "") return undefined;
  switch (param.type) {
    case "int": {
      const parsed = Number.parseInt(raw, 10);
      return Number.isNaN(parsed) ? undefined : parsed;
    }
    case "decimal": {
      const parsed = Number.parseFloat(raw);
      return Number.isNaN(parsed) ? undefined : parsed;
    }
    case "bool":
      return raw === "true";
    default:
      return raw;
  }
}

function uniqueParamName(parameters: ReportParameterDef[]): string {
  let suffix = parameters.length + 1;
  while (parameters.some((param) => param.name === `param${suffix}`)) {
    suffix += 1;
  }
  return `param${suffix}`;
}

export function ParametersPanel({ ir, onChange }: ParametersPanelProps) {
  const parameters = ir.parameters ?? [];

  return (
    <div className="flex flex-col gap-2">
      {parameters.length === 0 && (
        <p className="px-2 py-2 text-center text-sm text-muted-foreground">
          Parameters prompt the runner for values — bind them to filters for reusable reports.
        </p>
      )}
      {parameters.map((param, paramIndex) => {
        const update = (updated: ReportParameterDef) =>
          onChange(parameters.map((p, i) => (i === paramIndex ? updated : p)));
        return (
          <div key={paramIndex} className="flex flex-col gap-2 rounded-md border border-border p-2">
            <div className="flex items-center gap-1.5">
              <Input
                className="h-7 flex-1 font-mono text-xs"
                value={param.name}
                placeholder="parameterName"
                onChange={(event) => {
                  const nextName = event.target.value.replace(/[^a-zA-Z0-9_]/g, "");
                  onChange(
                    parameters.map((p, i) => (i === paramIndex ? { ...p, name: nextName } : p)),
                    { from: param.name, to: nextName },
                  );
                }}
              />
              <Button
                variant="ghost"
                size="icon"
                className="size-6"
                onClick={() => onChange(parameters.filter((_, i) => i !== paramIndex))}
                aria-label="Remove parameter"
              >
                <XIcon className="size-3.5" />
              </Button>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">Label</Label>
                <Input
                  className="h-7"
                  value={param.label ?? ""}
                  placeholder={param.name}
                  onChange={(event) => update({ ...param, label: event.target.value || undefined })}
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">Type</Label>
                <Select
                  value={param.type}
                  onValueChange={(type) => {
                    if (!type) return;
                    update({
                      ...param,
                      type,
                      default: undefined,
                      allowedValues: type === "ref" ? undefined : param.allowedValues,
                      refEntity: type === "ref" ? param.refEntity : undefined,
                    });
                  }}
                  items={REPORT_PARAMETER_TYPE_CHOICES}
                >
                  <SelectTrigger className="h-7">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {REPORT_PARAMETER_TYPE_CHOICES.map((choice) => (
                      <SelectItem key={choice.value} value={choice.value}>
                        {choice.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {param.type === "ref" ? (
                <div className="flex flex-col gap-1">
                  <Label className="text-xs text-muted-foreground">Entity</Label>
                  <Select
                    value={param.refEntity ?? ""}
                    onValueChange={(refEntity) => {
                      if (refEntity) update({ ...param, refEntity });
                    }}
                    items={REPORT_REF_ENTITY_CHOICES}
                  >
                    <SelectTrigger className="h-7">
                      <SelectValue placeholder="Select entity" />
                    </SelectTrigger>
                    <SelectContent>
                      {REPORT_REF_ENTITY_CHOICES.map((choice) => (
                        <SelectItem key={choice.value} value={choice.value}>
                          {choice.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              ) : (
                <div className="flex flex-col gap-1">
                  <Label className="text-xs text-muted-foreground">Default</Label>
                  <Input
                    className="h-7"
                    type={param.type === "int" || param.type === "decimal" ? "number" : "text"}
                    step={param.type === "decimal" ? "any" : undefined}
                    value={
                      typeof param.default === "string" ||
                      typeof param.default === "number" ||
                      typeof param.default === "boolean"
                        ? String(param.default)
                        : ""
                    }
                    onChange={(event) =>
                      update({ ...param, default: coerceDefault(param, event.target.value) })
                    }
                  />
                </div>
              )}
              <div className="flex flex-col gap-2 pt-1">
                <div className="flex items-center justify-between gap-2">
                  <Label className="text-xs text-muted-foreground">Required</Label>
                  <Switch
                    checked={param.required}
                    onCheckedChange={(required) => update({ ...param, required })}
                  />
                </div>
                <div className="flex items-center justify-between gap-2">
                  <Label className="text-xs text-muted-foreground">Multiple</Label>
                  <Switch
                    checked={param.multi ?? false}
                    onCheckedChange={(multi) => update({ ...param, multi, default: undefined })}
                  />
                </div>
              </div>
              {param.type !== "bool" && param.type !== "epoch" && param.type !== "ref" && (
                <div className="col-span-2 flex flex-col gap-1">
                  <Label className="text-xs text-muted-foreground">Allowed Values</Label>
                  <Input
                    className="h-7"
                    placeholder="Any value — or comma-separated choices"
                    value={(param.allowedValues ?? []).join(", ")}
                    onChange={(event) => {
                      const allowedValues = event.target.value
                        .split(",")
                        .map((value) => value.trim())
                        .filter(Boolean);
                      update({
                        ...param,
                        allowedValues: allowedValues.length > 0 ? allowedValues : undefined,
                      });
                    }}
                  />
                  <p className="text-2xs text-muted-foreground">
                    When set, the runner picks from these instead of typing a value.
                  </p>
                </div>
              )}
            </div>
          </div>
        );
      })}
      <Button
        variant="outline"
        size="sm"
        className="h-7 self-start"
        onClick={() =>
          onChange([
            ...parameters,
            { name: uniqueParamName(parameters), type: "int", required: false },
          ])
        }
      >
        <PlusIcon className="size-3.5" />
        Parameter
      </Button>
    </div>
  );
}
