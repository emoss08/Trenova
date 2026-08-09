import {
  AccessorialChargeAutocompleteField,
  CarrierAutocompleteField,
} from "@/components/autocomplete-fields";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SelectField } from "@/components/fields/select-field";
import { carrierRateMethodChoices } from "@/lib/choices";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";
import type { CarrierAssignmentPayload, CarrierEligibility } from "@trenova/shared/types/shipment";
import { OctagonXIcon, PlusIcon, TrashIcon, TriangleAlertIcon } from "lucide-react";
import { useFieldArray, type Control, type UseFormReturn } from "react-hook-form";

/**
 * Whether the current eligibility verdict should keep the submit button disabled.
 * Blockers always do; warnings only until the dispatcher explicitly overrides them.
 */
export function carrierEligibilityBlocksSubmit(
  eligibility: CarrierEligibility | undefined,
  overrideInsuranceWarning: boolean,
): boolean {
  if (!eligibility) return false;
  if (eligibility.blockers.length > 0) return true;
  return eligibility.warnings.length > 0 && !overrideInsuranceWarning;
}

export function CarrierEligibilityAlerts({
  control,
  eligibility,
  isLoading,
}: {
  control: Control<CarrierAssignmentPayload>;
  eligibility: CarrierEligibility | undefined;
  isLoading: boolean;
}) {
  if (isLoading) {
    return <Skeleton className="h-14 rounded-lg" />;
  }

  if (!eligibility || (eligibility.blockers.length === 0 && eligibility.warnings.length === 0)) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      {eligibility.blockers.length > 0 && (
        <Alert variant="destructive">
          <OctagonXIcon />
          <AlertTitle>Carrier cannot be assigned</AlertTitle>
          <AlertDescription>
            <ul className="list-disc pl-4">
              {eligibility.blockers.map((blocker) => (
                <li key={blocker}>{blocker}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}
      {eligibility.warnings.length > 0 && (
        <Alert variant="warning">
          <TriangleAlertIcon />
          <AlertTitle>Insurance warnings</AlertTitle>
          <AlertDescription>
            <ul className="list-disc pl-4">
              {eligibility.warnings.map((warning) => (
                <li key={warning}>{warning}</li>
              ))}
            </ul>
            {eligibility.blockers.length === 0 && (
              <div className="mt-2">
                <CheckboxField
                  control={control}
                  name="overrideInsuranceWarning"
                  label="Assign anyway"
                  description="Proceed despite the insurance warnings above."
                />
              </div>
            )}
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}

/**
 * The full brokered-coverage field set, shared by the dispatch console dialog and the
 * shipment panel's carrier tab so the two entry points cannot drift apart.
 */
export function CarrierAssignmentFields({
  form,
}: {
  form: UseFormReturn<CarrierAssignmentPayload>;
}) {
  const { control, getValues, setValue } = form;
  const { fields, append, remove } = useFieldArray({ control, name: "accessorials" });

  const handleAccessorialChargeChange = (index: number, option: AccessorialCharge | null) => {
    if (!option) return;
    if (!getValues(`accessorials.${index}.description`)) {
      setValue(`accessorials.${index}.description`, option.description ?? option.code, {
        shouldDirty: true,
        shouldValidate: true,
      });
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <CarrierAutocompleteField
            control={control}
            name="carrierId"
            label="Carrier"
            placeholder="Select carrier"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="rateMethod"
            label="Rate Method"
            placeholder="Rate Method"
            rules={{ required: true }}
            options={carrierRateMethodChoices}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="baseRate"
            label="Base Rate"
            placeholder="0.00"
            sideText="$"
            rules={{ required: true }}
            decimalScale={4}
            thousandSeparator
            description="Flat amount, or the per-mile rate multiplied by the move distance."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="fuelSurcharge"
            label="Fuel Surcharge"
            placeholder="0.00"
            sideText="$"
            decimalScale={4}
            thousandSeparator
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="proNumber"
            label="Carrier Pro Number"
            placeholder="Carrier's own reference"
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalDriverName"
            label="Driver Name"
            placeholder="Carrier driver"
            maxLength={255}
          />
        </FormControl>
        <FormControl>
          <PhoneNumberField
            control={control}
            name="externalDriverPhone"
            label="Driver Phone"
            placeholder="Phone"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalTractorNumber"
            label="Tractor Number"
            placeholder="Carrier tractor"
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalTrailerNumber"
            label="Trailer Number"
            placeholder="Carrier trailer"
            maxLength={50}
          />
        </FormControl>
      </FormGroup>

      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium">Accessorials</span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="h-6 px-2 text-[10px]"
            onClick={() => append({ accessorialChargeId: null, description: "", amount: 0 })}
          >
            <PlusIcon className="size-3" />
            Add accessorial
          </Button>
        </div>
        {fields.length === 0 && (
          <p className="text-xs text-muted-foreground">
            No accessorials. Add lumper fees, detention, or other pass-through charges the carrier
            bills for this move.
          </p>
        )}
        {fields.map((field, index) => (
          <div key={field.id} className="rounded-md border p-3">
            <div className="mb-2 flex items-center justify-between">
              <p className="text-xs font-medium">Accessorial {index + 1}</p>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className="h-6 px-2 text-[10px]"
                onClick={() => remove(index)}
              >
                <TrashIcon className="size-3" />
                Remove
              </Button>
            </div>
            <FormGroup cols={2}>
              <FormControl cols="full">
                <AccessorialChargeAutocompleteField
                  control={control}
                  name={`accessorials.${index}.accessorialChargeId`}
                  label="Accessorial Charge"
                  placeholder="Link a configured charge (optional)"
                  clearable
                  onOptionChange={(option) => handleAccessorialChargeChange(index, option)}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name={`accessorials.${index}.description`}
                  label="Description"
                  placeholder="e.g., Lumper fee"
                  rules={{ required: true }}
                  maxLength={255}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name={`accessorials.${index}.amount`}
                  label="Amount"
                  placeholder="0.00"
                  sideText="$"
                  rules={{ required: true }}
                  decimalScale={4}
                  thousandSeparator
                />
              </FormControl>
            </FormGroup>
          </div>
        ))}
      </div>
    </div>
  );
}
