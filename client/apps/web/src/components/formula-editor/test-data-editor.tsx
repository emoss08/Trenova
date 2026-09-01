import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { cn } from "@trenova/shared/lib/utils";
import { SHIPMENT_VARIABLES, VARIABLE_CATEGORIES } from "@trenova/shared/types/formula-template";
import { ChevronDown, ChevronUp, Database, RotateCcw } from "lucide-react";
import { useCallback, useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";

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

type TestDataEditorProps = {
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
  className?: string;
};

export function TestDataEditor({ values, onChange, className }: TestDataEditorProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const booleanSelectItems = [
    { label: "true", value: "true" },
    { label: "false", value: "false" },
  ];

  const handleValueChange = useCallback(
    (name: string, rawValue: string) => {
      const variable = SHIPMENT_VARIABLES.find((v) => v.name === name);
      let parsedValue: unknown = rawValue;

      if (variable?.type === "Number" || variable?.type === "Integer") {
        const num = parseFloat(rawValue);
        parsedValue = isNaN(num) ? 0 : num;
      } else if (variable?.type === "Boolean") {
        parsedValue = rawValue === "true";
      }

      onChange({ ...values, [name]: parsedValue });
    },
    [values, onChange],
  );

  const handleReset = useCallback(() => {
    onChange({ ...DEFAULT_TEST_VALUES });
  }, [onChange]);

  const formatValue = (value: unknown): string => {
    if (typeof value === "boolean") {
      return value ? "true" : "false";
    }
    if (value === null || value === undefined) {
      return "";
    }
    if (typeof value === "string" || typeof value === "number" || typeof value === "bigint") {
      return String(value);
    }
    return JSON.stringify(value);
  };

  return (
    <div className={cn("bg-muted/30 rounded-lg border", className)}>
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex w-full items-center justify-between px-3 py-2 text-left"
      >
        <div className="flex items-center gap-2">
          <Database className="text-muted-foreground size-3.5" />
          <span className="text-xs font-medium">Test Data</span>
          <span className="text-muted-foreground text-xs">
            ({Object.keys(values).length} variables)
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
            {VARIABLE_CATEGORIES.map((category) => {
              const categoryVars = SHIPMENT_VARIABLES.filter((v) => v.category === category.id);
              if (categoryVars.length === 0) return null;

              return (
                <div key={category.id}>
                  <h4 className="text-muted-foreground mb-2 text-[10px] font-semibold tracking-wide uppercase">
                    {category.label}
                  </h4>
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
                    {categoryVars.map((variable) => (
                      <div key={variable.name} className="space-y-1">
                        <label className="text-muted-foreground block text-[10px] font-medium">
                          {variable.name}
                        </label>
                        {variable.type === "Boolean" ? (
                          <Select
                            value={formatValue(values[variable.name])}
                            items={booleanSelectItems}
                            onValueChange={(value) => handleValueChange(variable.name, value ?? "")}
                          >
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder="Select a value" />
                            </SelectTrigger>
                            <SelectContent>
                              {booleanSelectItems.map((item) => (
                                <SelectItem key={item.value} value={item.value}>
                                  {item.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        ) : (
                          <Input
                            type={
                              variable.type === "Number" || variable.type === "Integer"
                                ? "number"
                                : "text"
                            }
                            value={formatValue(values[variable.name])}
                            onChange={(e) => handleValueChange(variable.name, e.target.value)}
                            className="h-7 text-xs"
                          />
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
