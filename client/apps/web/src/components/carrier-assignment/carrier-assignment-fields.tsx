import { useT } from "@trenova/shared/i18n/use-t";
import {
  AccessorialChargeAutocompleteField,
  CarrierAutocompleteField,
} from "@/components/autocomplete-fields";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SelectField } from "@/components/fields/select-field";
import { groupCarrierEligibility, hasEligibilityContent } from "@/lib/carrier-eligibility";
import { carrierRateMethodChoices } from "@/lib/choices";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import type {
  CarrierAssignmentPayload,
  CarrierAssignmentPayloadInput,
  CarrierEligibility,
} from "@trenova/shared/types/shipment";
import { PlusIcon, TrashIcon } from "lucide-react";
import { useMemo } from "react";
import { useFieldArray, type Control, type UseFormReturn } from "react-hook-form";
import {
  EligibilityGroup,
  IntelAdvisoriesCallout,
  IntelBlockersAlert,
} from "./intel-eligibility-alerts";

/**
 * The payload schema coerces GraphQL decimal strings to numbers, so the form's
 * field values (input) and submitted values (output) differ — both entry points
 * type their forms with these aliases so the pair cannot drift apart.
 */
export type CarrierAssignmentFormReturn = UseFormReturn<
  CarrierAssignmentPayloadInput,
  unknown,
  CarrierAssignmentPayload
>;
export type CarrierAssignmentFormControl = Control<
  CarrierAssignmentPayloadInput,
  unknown,
  CarrierAssignmentPayload
>;

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
  carrierId,
  onEligibilityChanged,
}: {
  control: CarrierAssignmentFormControl;
  eligibility: CarrierEligibility | undefined;
  isLoading: boolean;
  carrierId?: string;
  onEligibilityChanged?: () => void;
}) {
  const t = useT();

  const grouped = useMemo(
    () => (eligibility ? groupCarrierEligibility(eligibility) : null),
    [eligibility],
  );

  if (isLoading) {
    return <Skeleton className="h-14 rounded-lg" />;
  }

  if (!grouped || !hasEligibilityContent(grouped)) {
    return null;
  }

  const hasBlockers = grouped.recordBlockers.length > 0 || grouped.intelBlockers.length > 0;

  return (
    <div className="flex flex-col gap-3 rounded-lg border px-3 py-3">
      {grouped.recordBlockers.length > 0 && (
        <EligibilityGroup
          group="record-blockers"
          tone="blocker"
          title={t("Carrier cannot be assigned")}
          rows={grouped.recordBlockers}
        />
      )}
      <IntelBlockersAlert
        items={grouped.intelBlockers}
        carrierId={carrierId}
        onEligibilityChanged={onEligibilityChanged}
      />
      {grouped.warnings.length > 0 && (
        <EligibilityGroup
          group="warnings"
          tone="warning"
          title={t("Insurance warnings")}
          rows={grouped.warnings}
        >
          {!hasBlockers && (
            <div className="pt-1">
              <CheckboxField
                control={control}
                name="overrideInsuranceWarning"
                label={t("Assign anyway")}
                description={t("Proceed despite the insurance warnings above.")}
              />
            </div>
          )}
        </EligibilityGroup>
      )}
      <IntelAdvisoriesCallout items={grouped.advisories} carrierId={carrierId} />
    </div>
  );
}

/**
 * The full brokered-coverage field set, shared by the dispatch console dialog and the
 * shipment panel's carrier tab so the two entry points cannot drift apart.
 */
export function CarrierAssignmentFields({ form }: { form: CarrierAssignmentFormReturn }) {
  const t = useT();

  const { control, getValues, setValue } = form;
  const { fields, append, remove } = useFieldArray({ control, name: "accessorials" });

  const handleAccessorialChargeChange = (index: number, option: GraphQLSelectOption | null) => {
    if (!option) return;
    if (!getValues(`accessorials.${index}.description`)) {
      setValue(`accessorials.${index}.description`, option.description ?? option.label, {
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
            label={t("Carrier")}
            placeholder={t("Select carrier")}
            rules={{ required: true }}
            description={t("External carrier that will run this move.")}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="rateMethod"
            label={t("Rate method")}
            placeholder={t("Select rate method")}
            rules={{ required: true }}
            options={carrierRateMethodChoices}
            description={t("How the carrier's pay is calculated for this move.")}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="baseRate"
            label={t("Base rate")}
            placeholder="0.00"
            sideText="$"
            rules={{ required: true }}
            decimalScale={4}
            thousandSeparator
            description={t("Flat amount, or the per-mile rate multiplied by the move distance.")}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="fuelSurcharge"
            label={t("Fuel surcharge")}
            placeholder="0.00"
            sideText="$"
            decimalScale={4}
            thousandSeparator
            description={t("Fuel surcharge paid to the carrier on top of the base rate.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="proNumber"
            label={t("Carrier pro number")}
            placeholder={t("e.g., PRO-482910")}
            maxLength={50}
            description={t("The carrier's own tracking reference for this move.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalDriverName"
            label={t("Driver name")}
            placeholder={t("e.g., John Smith")}
            maxLength={255}
            description={t("Name of the carrier's driver running this move.")}
          />
        </FormControl>
        <FormControl>
          <PhoneNumberField
            control={control}
            name="externalDriverPhone"
            label={t("Driver phone")}
            placeholder="(555) 555-5555"
            description={t("Contact number for the carrier's driver while in transit.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalTractorNumber"
            label={t("Tractor number")}
            placeholder={t("e.g., T-4521")}
            maxLength={50}
            description={t("Unit number of the carrier's tractor on this move.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalTrailerNumber"
            label={t("Trailer number")}
            placeholder={t("e.g., TR-8834")}
            maxLength={50}
            description={t("Unit number of the carrier's trailer on this move.")}
          />
        </FormControl>
      </FormGroup>

      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium">{t("Accessorials")}</span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="h-6 px-2 text-2xs"
            onClick={() => append({ accessorialChargeId: null, description: "", amount: 0 })}
          >
            <PlusIcon className="size-3" />
            {t("Add accessorial")}
          </Button>
        </div>
        {fields.length === 0 && (
          <p className="text-muted-foreground text-xs">
            {t(
              "No accessorials. Add lumper fees, detention, or other pass-through charges the carrier bills for this move.",
            )}
          </p>
        )}
        {fields.map((field, index) => (
          <div key={field.id} className="rounded-md border p-3">
            <div className="mb-2 flex items-center justify-between">
              <p className="text-xs font-medium">{t("Accessorial {0}", index + 1)}</p>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                className="h-6 px-2 text-2xs"
                onClick={() => remove(index)}
              >
                <TrashIcon className="size-3" />
                {t("Remove")}
              </Button>
            </div>
            <FormGroup cols={2}>
              <FormControl cols="full">
                <AccessorialChargeAutocompleteField
                  control={control}
                  name={`accessorials.${index}.accessorialChargeId`}
                  label={t("Accessorial charge")}
                  placeholder={t("Link a configured charge (optional)")}
                  clearable
                  description={t("Optionally link a configured charge to prefill the description.")}
                  onOptionChange={(option) => handleAccessorialChargeChange(index, option)}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name={`accessorials.${index}.description`}
                  label={t("Description")}
                  placeholder={t("e.g., Lumper fee")}
                  rules={{ required: true }}
                  maxLength={255}
                  description={t("What the carrier is billing this charge for.")}
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name={`accessorials.${index}.amount`}
                  label={t("Amount")}
                  placeholder="0.00"
                  sideText="$"
                  rules={{ required: true }}
                  decimalScale={4}
                  thousandSeparator
                  description={t("Amount the carrier bills for this charge.")}
                />
              </FormControl>
            </FormGroup>
          </div>
        ))}
      </div>
    </div>
  );
}
