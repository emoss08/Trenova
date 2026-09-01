import { useKnownIdentifiers } from "@/hooks/use-formula-schema";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { cn } from "@trenova/shared/lib/utils";
import type { VariableDefinitionInput } from "@trenova/shared/types/formula-template";
import { ChevronDown, ChevronUp, Database, RotateCcw } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import type { VariableDoc } from "./known-identifiers";
import { CATEGORY_LABELS, categoryLabel, sampleInputKind } from "./schema-labels";

export const DEFAULT_TEST_VALUES: Record<string, unknown> = {
  // Shipment Fields
  proNumber: "PRO-100234",
  status: "New",
  baseRate: 3.25,
  weight: 10000,
  pieces: 25,
  temperatureMin: 34,
  temperatureMax: 40,
  ratingUnit: 1,
  "customer.name": "Acme Logistics",
  "customer.code": "ACME",
  "tractorType.name": "Day Cab",
  "tractorType.code": "DC",
  "tractorType.costPerMile": 1.75,
  "trailerType.name": "Dry Van",
  "trailerType.code": "DV",
  "trailerType.costPerMile": 0.5,
  "origin.city": "Atlanta",
  "origin.state": "GA",
  "origin.zip": "30301",
  "destination.city": "Miami",
  "destination.state": "FL",
  "destination.zip": "33101",
  totalDistance: 500,
  totalStops: 3,
  totalWeight: 10000,
  totalPieces: 25,
  totalLinearFeet: 20,
  totalHours: 6,
  pickupDayOfWeek: 2,
  pickupHour: 9,
  pickupMonth: 6,
  isWeekendPickup: false,
  hasHazmat: false,
  requiresTemperatureControl: false,
  temperatureDifferential: 0,
  freightChargeAmount: 1500,
  otherChargeAmount: 250,
  currentTotalCharge: 1750,
};

const BOOLEAN_ITEMS = [
  { label: "true", value: "true" },
  { label: "false", value: "false" },
];

type TestDataEditorProps = {
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
  /** Schema the template rates against; drives which fields are offered. */
  schemaId?: string;
  /** The template's own variables, offered in their own section. */
  customVariables?: VariableDefinitionInput[];
  className?: string;
};

function formatValue(value: unknown): string {
  if (typeof value === "boolean") return value ? "true" : "false";
  if (value === null || value === undefined) return "";
  if (typeof value === "string" || typeof value === "number" || typeof value === "bigint") {
    return String(value);
  }
  return JSON.stringify(value);
}

function parseValue(variable: VariableDoc, raw: string): unknown {
  switch (sampleInputKind(variable.type, variable.enum)) {
    case "number": {
      if (raw.trim() === "") return undefined;
      const parsed = Number(raw);
      return Number.isNaN(parsed) ? raw : parsed;
    }
    case "boolean":
      return raw === "true";
    default:
      return raw;
  }
}

function selectItemsFor(variable: VariableDoc, kind: "boolean" | "enum") {
  if (kind === "boolean") return BOOLEAN_ITEMS;
  return (variable.enum ?? []).map((option) => ({ label: option, value: option }));
}

function groupByCategory(variables: VariableDoc[]): Array<[string, VariableDoc[]]> {
  const groups = new Map<string, VariableDoc[]>();
  for (const variable of variables) {
    const key = variable.custom ? "custom" : variable.category || "shipment";
    const existing = groups.get(key);
    if (existing) existing.push(variable);
    else groups.set(key, [variable]);
  }
  // Custom variables belong to the author; they go first.
  return [...groups.entries()].sort(([a], [b]) => (a === "custom" ? -1 : b === "custom" ? 1 : 0));
}

/**
 * Sample inputs for a preview or a scenario, driven by the live schema so a
 * field added on the server, an enum, or the template's own variables all
 * show up here without a client release.
 */
export function TestDataEditor({
  values,
  onChange,
  schemaId = "shipment",
  customVariables = [],
  className,
}: TestDataEditorProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const known = useKnownIdentifiers(schemaId, customVariables);
  const groups = useMemo(() => groupByCategory(known.variables), [known.variables]);
  const byName = useMemo(
    () => new Map(known.variables.map((variable) => [variable.name, variable])),
    [known.variables],
  );

  const handleValueChange = useCallback(
    (name: string, rawValue: string) => {
      const variable = byName.get(name);
      const parsed = variable ? parseValue(variable, rawValue) : rawValue;
      if (parsed === undefined) {
        onChange(Object.fromEntries(Object.entries(values).filter(([key]) => key !== name)));
        return;
      }
      onChange({ ...values, [name]: parsed });
    },
    [byName, values, onChange],
  );

  const handleReset = useCallback(() => {
    onChange({ ...DEFAULT_TEST_VALUES });
  }, [onChange]);

  return (
    <div className={cn("bg-muted/30 rounded-lg border", className)}>
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        aria-expanded={isExpanded}
        className="flex w-full items-center justify-between px-3 py-2 text-left"
      >
        <div className="flex items-center gap-2">
          <Database className="text-muted-foreground size-3.5" />
          <span className="text-xs font-medium">Sample Data</span>
          <span className="text-muted-foreground text-xs">
            ({Object.keys(values).length} values)
          </span>
        </div>
        {isExpanded ? (
          <ChevronUp className="text-muted-foreground size-3.5" />
        ) : (
          <ChevronDown className="text-muted-foreground size-3.5" />
        )}
      </button>

      {isExpanded && (
        <div className="border-t px-3 py-3">
          <div className="mb-3 flex items-center justify-between">
            <p className="text-muted-foreground text-xs">
              Edit values below to test your expression with different inputs
            </p>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleReset}
              className="text-muted-foreground h-6 gap-1 px-2 text-xs"
            >
              <RotateCcw className="size-3" />
              Reset
            </Button>
          </div>

          <div className="space-y-4">
            {groups.map(([category, categoryVars]) => (
              <div key={category}>
                <h4 className="text-muted-foreground mb-2 text-[10px] font-semibold tracking-wide uppercase">
                  {categoryLabel(category, CATEGORY_LABELS)}
                </h4>
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
                  {categoryVars.map((variable) => {
                    const inputId = `sample-${variable.name.replace(/\W/g, "-")}`;
                    const kind = sampleInputKind(variable.type, variable.enum);
                    const current = formatValue(values[variable.name]);
                    return (
                      <div key={variable.name} className="space-y-1">
                        <label
                          htmlFor={inputId}
                          className="text-muted-foreground block truncate text-[10px] font-medium"
                          title={variable.description || variable.name}
                        >
                          {variable.name}
                        </label>
                        {kind === "boolean" || kind === "enum" ? (
                          <Select
                            value={current}
                            items={selectItemsFor(variable, kind)}
                            onValueChange={(value) => handleValueChange(variable.name, value ?? "")}
                          >
                            <SelectTrigger id={inputId} className="w-full">
                              <SelectValue placeholder="Select a value" />
                            </SelectTrigger>
                            <SelectContent>
                              {selectItemsFor(variable, kind).map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        ) : (
                          <Input
                            id={inputId}
                            type={kind === "number" ? "number" : "text"}
                            value={current}
                            onChange={(e) => handleValueChange(variable.name, e.target.value)}
                            placeholder={variable.nullable ? "empty" : undefined}
                            className="h-7 text-xs"
                          />
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
