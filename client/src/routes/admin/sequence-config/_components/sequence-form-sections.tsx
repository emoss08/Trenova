import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import type { SequenceConfigDocument } from "@/types/sequence-config";
import { ChevronDownIcon } from "lucide-react";
import { useState, type ReactNode } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { separatorOptions, yearDigitsOptions } from "./sequence-config-constants";

type SectionProps = { index: number };

function SectionCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <Card>
      <CardHeader className="border-b pb-3">
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="pt-4 pb-4">
        <FormGroup cols={2}>{children}</FormGroup>
      </CardContent>
    </Card>
  );
}

export function CoreStructureSection({ index }: SectionProps) {
  const { control } = useFormContext<SequenceConfigDocument>();
  const useSeparators = useWatch({
    control,
    name: `configs.${index}.useSeparators`,
  });

  return (
    <SectionCard
      title="Core Structure"
      description="Primary sequence components and delimiter behavior."
    >
      <FormControl>
        <InputField
          control={control}
          name={`configs.${index}.prefix`}
          label="Prefix"
          placeholder="e.g. PRO"
          description="Leading literal text included in every generated value."
          maxLength={20}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name={`configs.${index}.sequenceDigits`}
          label="Sequence Digits"
          description="Width of the zero-padded counter (1–10)."
          min={1}
          max={10}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name={`configs.${index}.useSeparators`}
          label="Use Separators"
          description="Insert a delimiter between each token segment."
          position="left"
          outlined
        />
      </FormControl>
      {useSeparators ? (
        <FormControl>
          <SelectField
            control={control}
            name={`configs.${index}.separatorChar`}
            label="Separator Character"
            options={separatorOptions}
            placeholder="Select separator"
          />
        </FormControl>
      ) : null}
    </SectionCard>
  );
}

export function DateComponentsSection({ index }: SectionProps) {
  const { control } = useFormContext<SequenceConfigDocument>();
  const includeYear = useWatch({
    control,
    name: `configs.${index}.includeYear`,
  });

  return (
    <SectionCard
      title="Date Components"
      description="Embed period context to make values self-describing."
    >
      <FormControl cols="full">
        <div className="grid gap-3 sm:grid-cols-2">
          <SwitchField
            control={control}
            name={`configs.${index}.includeYear`}
            label="Include Year"
            description="Embed a 2- or 4-digit year token."
            position="left"
            outlined
          />
          <SwitchField
            control={control}
            name={`configs.${index}.includeMonth`}
            label="Include Month"
            description="Append the current month as a 2-digit number."
            position="left"
            outlined
          />
          <SwitchField
            control={control}
            name={`configs.${index}.includeWeekNumber`}
            label="Include ISO Week Number"
            description="Append the ISO week number for weekly grouping."
            position="left"
            outlined
          />
          <SwitchField
            control={control}
            name={`configs.${index}.includeDay`}
            label="Include Day"
            description="Append the day of the month as 2 digits."
            position="left"
            outlined
          />
        </div>
      </FormControl>
      {includeYear ? (
        <FormControl cols="full">
          <Controller
            control={control}
            name={`configs.${index}.yearDigits`}
            render={({ field, fieldState }) => (
              <div className="space-y-1.5">
                <Label className={fieldState.error ? "text-destructive" : ""}>Year Digits</Label>
                <div className="grid grid-cols-2 gap-2">
                  {yearDigitsOptions.map((option) => {
                    const isActive = field.value === option.value;
                    return (
                      <button
                        key={option.value}
                        type="button"
                        onClick={() => field.onChange(option.value)}
                        aria-pressed={isActive}
                        className={cn(
                          "flex flex-col items-start gap-0.5 rounded-md border px-3 py-2 text-left text-sm transition-colors",
                          isActive
                            ? "border-blue-500/40 bg-blue-500/15 text-blue-700 dark:text-blue-300"
                            : "border-input bg-background text-foreground hover:bg-muted",
                        )}
                      >
                        <span className="font-medium leading-none">{option.label}</span>
                        <span className="text-2xs text-muted-foreground">{option.example}</span>
                      </button>
                    );
                  })}
                </div>
                {fieldState.error?.message ? (
                  <p className="text-xs text-destructive">{fieldState.error.message}</p>
                ) : (
                  <p className="text-2xs text-muted-foreground">
                    How many digits of the year to embed in the code.
                  </p>
                )}
              </div>
            )}
          />
        </FormControl>
      ) : null}
    </SectionCard>
  );
}

export function ContextComponentsSection({ index }: SectionProps) {
  const { control } = useFormContext<SequenceConfigDocument>();

  return (
    <SectionCard
      title="Context Components"
      description="Embed operational identity fields resolved at generation time."
    >
      <FormControl>
        <SwitchField
          control={control}
          name={`configs.${index}.includeLocationCode`}
          label="Include Location Code"
          description="Embed the origin location's code, resolved from the shipment."
          position="left"
          outlined
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name={`configs.${index}.includeBusinessUnitCode`}
          label="Include Business Unit Code"
          description="Embed the business unit's code, resolved from the organization."
          position="left"
          outlined
        />
      </FormControl>
    </SectionCard>
  );
}

export function AdvancedSection({ index }: SectionProps) {
  const { control } = useFormContext<SequenceConfigDocument>();
  const includeRandomDigits = useWatch({
    control,
    name: `configs.${index}.includeRandomDigits`,
  });
  const includeCheckDigit = useWatch({
    control,
    name: `configs.${index}.includeCheckDigit`,
  });
  const allowCustomFormat = useWatch({
    control,
    name: `configs.${index}.allowCustomFormat`,
  });

  const enabledCount =
    Number(Boolean(includeRandomDigits)) +
    Number(Boolean(includeCheckDigit)) +
    Number(Boolean(allowCustomFormat));

  const [open, setOpen] = useState(enabledCount > 0);

  return (
    <Card>
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger
          render={(props) => (
            <button
              {...props}
              type="button"
              className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left"
            >
              <div className="flex flex-col gap-0.5">
                <span className="text-base leading-snug font-medium">Advanced</span>
                <span className="text-sm text-muted-foreground">
                  Validation helpers and custom formatting overrides.
                </span>
              </div>
              <div className="flex items-center gap-2">
                {enabledCount > 0 ? (
                  <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-foreground">
                    {enabledCount} enabled
                  </span>
                ) : null}
                <ChevronDownIcon
                  className={cn(
                    "size-4 text-muted-foreground transition-transform",
                    open && "rotate-180",
                  )}
                />
              </div>
            </button>
          )}
        />
        <CollapsibleContent>
          <div className="border-t px-4 pt-4 pb-4">
            <FormGroup cols={2}>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`configs.${index}.includeRandomDigits`}
                  label="Include Random Digits"
                  description="Append random digits for collision avoidance."
                  position="left"
                  outlined
                />
              </FormControl>
              {includeRandomDigits ? (
                <FormControl>
                  <NumberField
                    control={control}
                    name={`configs.${index}.randomDigitsCount`}
                    label="Random Digits Count"
                    description="Number of random digits to append (1–10)."
                    min={1}
                    max={10}
                  />
                </FormControl>
              ) : null}
              <FormControl>
                <SwitchField
                  control={control}
                  name={`configs.${index}.includeCheckDigit`}
                  label="Include Check Digit"
                  description="Append a Luhn-computed check digit for validation."
                  position="left"
                  outlined
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`configs.${index}.allowCustomFormat`}
                  label="Allow Custom Format"
                  description="Override auto-composition with a token template."
                  position="left"
                  outlined
                />
              </FormControl>
              {allowCustomFormat ? (
                <FormControl cols="full">
                  <InputField
                    control={control}
                    name={`configs.${index}.customFormat`}
                    label="Custom Format Template"
                    placeholder="{P}-{Y}{M}-{S}"
                    description="Use tokens like {P}, {Y}, {M}, {S}. See Tokens reference above."
                  />
                </FormControl>
              ) : null}
            </FormGroup>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </Card>
  );
}
