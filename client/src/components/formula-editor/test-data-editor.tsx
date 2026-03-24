import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { SHIPMENT_VARIABLES, VARIABLE_CATEGORIES } from "@/types/formula-template";
import { ChevronDown, ChevronUp, Database, RotateCcw } from "lucide-react";
import { useCallback, useState } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select";

export const DEFAULT_TEST_VALUES: Record<string, unknown> = {
  // Shipment Fields
  proNumber: "PRO-100234",
  status: "New",
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
  totalDistance: 500,
  totalStops: 3,
  totalWeight: 10000,
  totalPieces: 25,
  totalLinearFeet: 20,
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
    if (
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "bigint"
    ) {
      return String(value);
    }
    return JSON.stringify(value);
  };

  return (
    <div className={cn("rounded-lg border bg-muted/30", className)}>
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex w-full items-center justify-between px-3 py-2 text-left"
      >
        <div className="flex items-center gap-2">
          <Database className="size-3.5 text-muted-foreground" />
          <span className="text-xs font-medium">Test Data</span>
          <span className="text-xs text-muted-foreground">
            ({Object.keys(values).length} variables)
          </span>
        </div>
        {isExpanded ? (
          <ChevronUp className="size-3.5 text-muted-foreground" />
        ) : (
          <ChevronDown className="size-3.5 text-muted-foreground" />
        )}
      </button>

      {isExpanded && (
        <div className="border-t px-3 py-3">
          <div className="mb-3 flex items-center justify-between">
            <p className="text-xs text-muted-foreground">
              Edit values below to test your expression with different inputs
            </p>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleReset}
              className="h-6 gap-1 px-2 text-xs text-muted-foreground"
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
                  <h4 className="mb-2 text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
                    {category.label}
                  </h4>
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
                    {categoryVars.map((variable) => (
                      <div key={variable.name} className="space-y-1">
                        <label className="block text-[10px] font-medium text-muted-foreground">
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
