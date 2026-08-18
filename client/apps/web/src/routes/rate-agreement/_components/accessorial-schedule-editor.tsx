import { AccessorialChargeAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { accessorialChargeMethodChoices, rateUnitChoices } from "@/lib/choices";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { RateAgreement, RateAgreementAccessorial } from "@trenova/shared/types/rate";
import { PlusIcon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

const NEW_ACCESSORIAL = {
  accessorialChargeId: "",
  method: "Flat",
  amount: null,
  waived: false,
  autoApply: false,
  applyCondition: "",
} as unknown as RateAgreementAccessorial;

/**
 * The contract's own prices for accessorial services.
 *
 * This is what makes a rate confirmation and an invoice agree: both read the
 * price from here rather than one reading the organization default and the
 * other whatever a clerk typed. A row marked to apply automatically becomes a
 * system charge on every shipment the contract prices, so it never depends on
 * somebody remembering.
 */
export function AccessorialScheduleEditor() {
  const { control } = useFormContext<RateAgreement>();
  const { fields, append, remove } = useFieldArray({ control, name: "accessorials" });
  const accessorials = (useWatch({ control, name: "accessorials" }) ??
    []) as RateAgreementAccessorial[];

  return (
    <div className="flex flex-col gap-3">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No negotiated accessorials. Without one, every accessorial is priced at the organization
          default instead of at what the contract says.
        </p>
      )}

      {fields.map((field, index) => {
        const row = accessorials[index];
        const isPerUnit = row?.method === "PerUnit";
        const autoApplies = Boolean(row?.autoApply);

        return (
          <div key={field.id} className="rounded-md border bg-card p-4">
            <div className="mb-3 flex items-center justify-between">
              <p className="text-sm font-medium">
                Accessorial {index + 1}
                {row?.waived && (
                  <span className="text-muted-foreground ml-2 text-xs font-normal">Waived</span>
                )}
              </p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-6 text-xs"
                onClick={() => remove(index)}
              >
                Remove
              </Button>
            </div>

            <FormGroup cols={3}>
              <FormControl>
                <AccessorialChargeAutocompleteField
                  control={control}
                  rules={{ required: true }}
                  name={`accessorials.${index}.accessorialChargeId` as never}
                  label="Accessorial"
                  placeholder="Charge"
                  description="Which service this price is for"
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  rules={{ required: true }}
                  name={`accessorials.${index}.method` as never}
                  label="Method"
                  placeholder="Method"
                  description="Flat, per unit, or a percentage"
                  options={accessorialChargeMethodChoices}
                />
              </FormControl>
              {isPerUnit ? (
                <FormControl>
                  <SelectField
                    control={control}
                    rules={{ required: true }}
                    name={`accessorials.${index}.rateUnit` as never}
                    label="Rate Unit"
                    placeholder="Unit"
                    description="What a unit is on this charge"
                    options={rateUnitChoices}
                  />
                </FormControl>
              ) : (
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.amount` as never}
                    label="Amount"
                    placeholder="75.00"
                    sideText="$"
                    decimalScale={2}
                    thousandSeparator
                    description="What the contract charges"
                  />
                </FormControl>
              )}
            </FormGroup>

            {isPerUnit && (
              <FormGroup cols={3} className="mt-3">
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.amount` as never}
                    label="Amount Per Unit"
                    placeholder="65.00"
                    sideText="$"
                    decimalScale={2}
                    thousandSeparator
                    description="What the contract charges for each unit"
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.freeUnits` as never}
                    label="Free Units"
                    placeholder="2"
                    description="Units the contract gives away before charging"
                  />
                </FormControl>
                <FormControl>
                  <NumberField
                    control={control}
                    name={`accessorials.${index}.maxAmount` as never}
                    label="Maximum Amount"
                    placeholder="0.00"
                    sideText="$"
                    decimalScale={2}
                    thousandSeparator
                    description="A ceiling on what this charge can reach"
                  />
                </FormControl>
              </FormGroup>
            )}

            <FormGroup cols={2} className="mt-3">
              <FormControl>
                <SwitchField
                  control={control}
                  name={`accessorials.${index}.autoApply` as never}
                  label="Auto Apply"
                  description="Added to every shipment this contract prices, so it never depends on somebody remembering"
                  outlined
                />
              </FormControl>
              <FormControl>
                <SwitchField
                  control={control}
                  name={`accessorials.${index}.waived` as never}
                  label="Waived"
                  description="The contract gives this service away, which is a stated term rather than an omission"
                  outlined
                />
              </FormControl>
            </FormGroup>

            {autoApplies && (
              <FormGroup cols={1} className="mt-3">
                <FormControl cols="full">
                  <InputField
                    control={control}
                    name={`accessorials.${index}.applyCondition` as never}
                    label="Apply Condition"
                    placeholder="totalStops > 2"
                    description="An expression in the same language the rating formulas use. Leave empty to apply to every shipment."
                  />
                </FormControl>
              </FormGroup>
            )}
          </div>
        );
      })}

      <Button type="button" variant="outline" size="sm" onClick={() => append(NEW_ACCESSORIAL)}>
        <PlusIcon className="mr-1 size-3.5" />
        Add accessorial
      </Button>
    </div>
  );
}
