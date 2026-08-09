import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import {
  carrierComplianceStatusChoices,
  carrierInsurancePolicyTypeChoices,
  carrierSafetyRatingChoices,
} from "@/lib/choices";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

function CarrierInsurancePolicyEditor() {
  const { control } = useFormContext<Carrier>();
  const { fields, append, remove } = useFieldArray({ control, name: "insurancePolicies" });

  const appendPolicy = () => {
    append({
      policyType: "AutoLiability",
      policyNumber: "",
      providerName: "",
      coverageAmount: 0,
      effectiveDate: 0,
      expirationDate: 0,
      isVerified: false,
    });
  };

  return (
    <div className="flex flex-col gap-3">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No insurance policies on file. Add the carrier&apos;s active policies to track coverage
          and expirations.
        </p>
      )}

      {fields.map((field, index) => (
        <div key={field.id} className="rounded-md border p-3">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs font-medium">Policy {index + 1}</p>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              className="h-6 text-xs"
              onClick={() => remove(index)}
            >
              Remove
            </Button>
          </div>

          <FormGroup cols={2}>
            <FormControl>
              <SelectField
                control={control}
                name={`insurancePolicies.${index}.policyType`}
                label="Policy Type"
                placeholder="Select policy type"
                rules={{ required: true }}
                options={carrierInsurancePolicyTypeChoices}
                description="Coverage line this policy provides."
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name={`insurancePolicies.${index}.policyNumber`}
                label="Policy Number"
                placeholder="e.g., AL-1234567"
                rules={{ required: true }}
                description="Policy number as issued by the insurance provider."
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name={`insurancePolicies.${index}.providerName`}
                label="Provider"
                placeholder="e.g., Progressive Commercial"
                rules={{ required: true }}
                description="Insurance company underwriting this policy."
                maxLength={255}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`insurancePolicies.${index}.coverageAmount`}
                label="Coverage Amount"
                placeholder="1,000,000"
                sideText="$"
                rules={{ required: true }}
                description="Coverage limit in dollars."
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name={`insurancePolicies.${index}.effectiveDate`}
                label="Effective Date"
                placeholder="Effective Date"
                rules={{ required: true }}
                description="Date coverage under this policy begins."
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name={`insurancePolicies.${index}.expirationDate`}
                label="Expiration Date"
                placeholder="Expiration Date"
                rules={{ required: true }}
                description="Date coverage under this policy ends."
              />
            </FormControl>
            <FormControl cols="full">
              <SwitchField
                control={control}
                name={`insurancePolicies.${index}.isVerified`}
                label="Verified"
                description="A certificate of insurance has been received and verified for this policy."
              />
            </FormControl>
          </FormGroup>
        </div>
      ))}

      <div>
        <Button type="button" size="sm" variant="outline" onClick={appendPolicy}>
          Add policy
        </Button>
      </div>
    </div>
  );
}

export function CarrierComplianceForm() {
  const { control } = useFormContext<Carrier>();
  const complianceStatus = useWatch({ control, name: "complianceStatus" });

  return (
    <div className="space-y-6">
      <FormSection
        title="Compliance Status"
        description="Qualification standing and FMCSA safety rating for this carrier."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="complianceStatus"
              label="Compliance Status"
              placeholder="Compliance Status"
              description="Whether the carrier is qualified to haul freight for your organization."
              options={carrierComplianceStatusChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="safetyRating"
              label="Safety Rating"
              placeholder="Safety Rating"
              description="Most recent FMCSA safety rating on record for this carrier."
              options={carrierSafetyRatingChoices}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="qualifiedAt"
              label="Qualified At"
              placeholder="Qualified At"
              description="Date the carrier most recently passed qualification."
            />
          </FormControl>
          {complianceStatus === "Disqualified" && (
            <FormControl cols="full">
              <TextareaField
                control={control}
                rules={{ required: true }}
                name="disqualifiedReason"
                label="Disqualification Reason"
                placeholder="Reason the carrier was disqualified"
                description="Required when a carrier is disqualified. Recorded for audit purposes."
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>

      <FormSection
        title="Insurance Policies"
        description="Active insurance coverage on file for this carrier."
      >
        <CarrierInsurancePolicyEditor />
      </FormSection>
    </div>
  );
}
