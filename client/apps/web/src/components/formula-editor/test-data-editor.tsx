import { useT } from "@trenova/shared/i18n/use-t";
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
import {
  ChevronDownIcon,
  ChevronUpIcon,
  Database01Icon,
  RefreshCcw01Icon,
} from "@trenova/shared/components/icons";
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
  "serviceType.code": "STD",
  "serviceType.description": "Standard",
  "shipmentType.code": "TL",
  "shipmentType.description": "Truckload",
  "tractorType.code": "DC",
  "tractorType.description": "Day Cab",
  "trailerType.code": "RF",
  "trailerType.description": "Reefer",
  "origin.city": "Atlanta",
  "origin.state": "GA",
  "origin.zip": "30301",
  "destination.city": "Miami",
  "destination.state": "FL",
  "destination.zip": "33101",
  fuelPrice: 3.85,
  fuelPriceDate: "2026-06-01",
  fuelIndexCode: "DOE_US",
  totalDistance: 500,
  totalStops: 3,
  totalWeight: 10000,
  totalPieces: 25,
  totalLinearFeet: 20,
  totalCubicFeet: 1200,
  density: 8.3333,
  primaryFreightClass: "85",
  highestFreightClass: "100",
  totalHours: 6,
  pickupDate: "2026-06-02T09:00:00",
  deliveryDate: "2026-06-02T15:00:00",
  pickupDayOfWeek: 2,
  pickupHour: 9,
  pickupMonth: 6,
  isWeekendPickup: false,
  hasHazmat: false,
  requiresTemperatureControl: true,
  temperatureDifferential: 6,
  freightChargeAmount: 1500,
  otherChargeAmount: 250,
  currentTotalCharge: 1750,
  stops: [
    {
      type: "Pickup",
      status: "New",
      scheduleType: "Open",
      sequence: 0,
      pieces: 25,
      weight: 10000,
      windowStart: 1780405200,
      windowEnd: 1780408800,
      actualArrival: null,
      actualDeparture: null,
      city: "Atlanta",
      state: "GA",
      zip: "30301",
      timezone: "America/New_York",
    },
    {
      type: "Delivery",
      status: "New",
      scheduleType: "Open",
      sequence: 1,
      pieces: 10,
      weight: 4000,
      windowStart: 1780416000,
      windowEnd: 1780419600,
      actualArrival: null,
      actualDeparture: null,
      city: "Orlando",
      state: "FL",
      zip: "32801",
      timezone: "America/New_York",
    },
    {
      type: "Delivery",
      status: "New",
      scheduleType: "Open",
      sequence: 2,
      pieces: 15,
      weight: 6000,
      windowStart: 1780423200,
      windowEnd: 1780426800,
      actualArrival: null,
      actualDeparture: null,
      city: "Miami",
      state: "FL",
      zip: "33101",
      timezone: "America/New_York",
    },
  ],
  commodities: [
    {
      name: "General Freight",
      freightClass: "85",
      stackable: true,
      hazmat: false,
      weight: 6000,
      pieces: 15,
      lengthFeet: 4,
      widthFeet: 4,
      heightFeet: 5,
      cubicFeet: 1200,
      density: 5,
    },
    {
      name: "Produce",
      freightClass: "100",
      stackable: false,
      hazmat: false,
      weight: 4000,
      pieces: 10,
      lengthFeet: null,
      widthFeet: null,
      heightFeet: null,
      cubicFeet: null,
      density: null,
    },
  ],
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
  const t = useT();

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
          <Database01Icon className="text-muted-foreground size-3.5" />
          <span className="text-xs font-medium">{t("Sample data")}</span>
          <span className="text-muted-foreground text-xs">
            {t("({0} values)", Object.keys(values).length)}
          </span>
        </div>
        {isExpanded ? (
          <ChevronUpIcon className="text-muted-foreground size-3.5" />
        ) : (
          <ChevronDownIcon className="text-muted-foreground size-3.5" />
        )}
      </button>

      {isExpanded && (
        <div className="border-t px-3 py-3">
          <div className="mb-3 flex items-center justify-between">
            <p className="text-muted-foreground text-xs">
              {t("Edit values below to test your expression with different inputs")}
            </p>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleReset}
              className="text-muted-foreground h-6 gap-1 px-2 text-xs"
            >
              <RefreshCcw01Icon className="size-3" />
              {t("Reset")}
            </Button>
          </div>

          <div className="space-y-4">
            {groups.map(([category, categoryVars]) => (
              <div key={category}>
                <h4 className="text-muted-foreground mb-2 text-xs font-semibold">
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
                          className="text-muted-foreground block truncate text-2xs font-medium"
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
                              <SelectValue placeholder={t("Select a value")} />
                            </SelectTrigger>
                            <SelectContent>
                              {selectItemsFor(variable, kind).map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {t(item.label)}
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
