import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import {
  locationCodeComponents,
  type LocationCodeComponent,
  type SequenceConfigDocument,
} from "@/types/sequence-config";
import { Building2, MailIcon, MapIcon, TagIcon, type LucideIcon } from "lucide-react";
import { Controller, useFormContext } from "react-hook-form";
import { casingOptions, separatorOptions } from "./sequence-config-constants";

const componentLabels: Record<LocationCodeComponent, string> = {
  name: "Name",
  city: "City",
  state: "State",
  postal_code: "Postal Code",
};

const componentIcons: Record<LocationCodeComponent, LucideIcon> = {
  name: TagIcon,
  city: Building2,
  state: MapIcon,
  postal_code: MailIcon,
};

const componentDescriptions: Record<LocationCodeComponent, string> = {
  name: "Location name",
  city: "City of address",
  state: "State of address",
  postal_code: "ZIP / postal code",
};

export function LocationCodeStrategySection({ index }: { index: number }) {
  const { control } = useFormContext<SequenceConfigDocument>();

  return (
    <Card>
      <CardHeader className="border-b pb-3">
        <CardTitle>Code Strategy</CardTitle>
        <CardDescription>
          Derive readable components from location attributes, then append a sequence. The combined
          length cannot exceed 32 characters.
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-4 pb-4">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <Controller
              control={control}
              name={`configs.${index}.locationCodeStrategy.components`}
              render={({ field, fieldState }) => {
                const selected = field.value ?? [];
                return (
                  <div className="space-y-2">
                    <Label className={fieldState.error ? "text-destructive" : ""}>
                      Components
                    </Label>
                    <div className="grid gap-2 sm:grid-cols-2">
                      {locationCodeComponents.map((component) => {
                        const Icon = componentIcons[component];
                        const isSelected = selected.includes(component);
                        return (
                          <button
                            key={component}
                            type="button"
                            onClick={() => {
                              field.onChange(
                                isSelected
                                  ? selected.filter((value) => value !== component)
                                  : [...selected, component],
                              );
                            }}
                            aria-pressed={isSelected}
                            className={cn(
                              "group flex items-center gap-3 rounded-md border px-3 py-2 text-left text-sm transition-colors",
                              isSelected
                                ? "border-blue-500/40 bg-blue-500/15 text-blue-700 dark:text-blue-300"
                                : "border-input bg-background text-foreground hover:bg-muted",
                            )}
                          >
                            <span
                              className={cn(
                                "flex size-7 shrink-0 items-center justify-center rounded-md border",
                                isSelected
                                  ? "border-blue-500/30 bg-blue-500/15 text-blue-600 dark:text-blue-300"
                                  : "border-border bg-muted/50 text-muted-foreground",
                              )}
                            >
                              <Icon className="size-4" aria-hidden />
                            </span>
                            <span className="flex min-w-0 flex-1 flex-col">
                              <span className="truncate font-medium leading-tight">
                                {componentLabels[component]}
                              </span>
                              <span className="truncate text-2xs text-muted-foreground">
                                {componentDescriptions[component]}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                    <p className="text-2xs text-muted-foreground">
                      Components are applied in the order they are selected.
                    </p>
                  </div>
                );
              }}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`configs.${index}.locationCodeStrategy.componentWidth`}
              label="Component Width"
              description="Characters drawn from each selected attribute (1–10)."
              min={1}
              max={10}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`configs.${index}.locationCodeStrategy.sequenceDigits`}
              label="Sequence Digits"
              description="Width of the zero-padded counter (1–10)."
              min={1}
              max={10}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name={`configs.${index}.locationCodeStrategy.separator`}
              label="Separator"
              description="Inserted between the prefix and the sequence."
              options={separatorOptions}
              placeholder="Select separator"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name={`configs.${index}.locationCodeStrategy.casing`}
              label="Casing"
              description="Applied to the derived prefix."
              options={casingOptions}
              placeholder="Select casing"
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name={`configs.${index}.locationCodeStrategy.fallbackPrefix`}
              label="Fallback Prefix"
              description="Used when the location name cannot produce a usable prefix."
              placeholder="LOC"
              maxLength={10}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
