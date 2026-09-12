import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

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
          {t("No insurance policies on file. Add the carrier's active policies to track coverage and expirations.")}
        </p>
      )}

      {fields.map((field, index) => (
        <div key={field.id} className="rounded-md border p-3">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs font-medium">{t("Policy {0}", index + 1)}</p>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              className="h-6 text-xs"
              onClick={() => remove(index)}
            >
              {t("Remove")}
            </Button>
          </div>

          <FormGroup cols={2}>
            <FormControl>
              <SelectField
                control={control}
                name={`insurancePolicies.${index}.policyType`}
                label={t("Policy Type")}
                placeholder={t("Select policy type")}
                rules={{ required: true }}
                options={carrierInsurancePolicyTypeChoices}
                description={t("Coverage line this policy provides.")}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name={`insurancePolicies.${index}.policyNumber`}
                label={t("Policy Number")}
                placeholder={t("e.g., AL-1234567")}
                rules={{ required: true }}
                description={t("Policy number as issued by the insurance provider.")}
                maxLength={100}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name={`insurancePolicies.${index}.providerName`}
                label={t("Provider")}
                placeholder={t("e.g., Progressive Commercial")}
                rules={{ required: true }}
                description={t("Insurance company underwriting this policy.")}
                maxLength={255}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`insurancePolicies.${index}.coverageAmount`}
                label={t("Coverage Amount")}
                placeholder="1,000,000"
                sideText="$"
                rules={{ required: true }}
                description={t("Coverage limit in dollars.")}
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name={`insurancePolicies.${index}.effectiveDate`}
                label={t("Effective Date")}
                placeholder={t("Effective Date")}
                rules={{ required: true }}
                description={t("Date coverage under this policy begins.")}
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={control}
                name={`insurancePolicies.${index}.expirationDate`}
                label={t("Expiration Date")}
                placeholder={t("Expiration Date")}
                rules={{ required: true }}
                description={t("Date coverage under this policy ends.")}
              />
            </FormControl>
            <FormControl cols="full">
              <SwitchField
                control={control}
                name={`insurancePolicies.${index}.isVerified`}
                label={t("Verified")}
                description={t("A certificate of insurance has been received and verified for this policy.")}
              />
            </FormControl>
          </FormGroup>
        </div>
      ))}

      <div>
        <Button type="button" size="sm" variant="outline" onClick={appendPolicy}>
          {t("Add policy")}
        </Button>
      </div>
    </div>
  );
}

export function CarrierComplianceForm() {
  const t = useT();

  const { control } = useFormContext<Carrier>();
  const complianceStatus = useWatch({ control, name: "complianceStatus" });

  return (
    <div className="space-y-6">
      <FormSection
        title={t("Compliance Status")}
        description={t("Qualification standing and FMCSA safety rating for this carrier.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="complianceStatus"
              label={t("Compliance Status")}
              placeholder={t("Compliance Status")}
              description={t("Whether the carrier is qualified to haul freight for your organization.")}
              options={carrierComplianceStatusChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="safetyRating"
              label={t("Safety Rating")}
              placeholder={t("Safety Rating")}
              description={t("Most recent FMCSA safety rating on record for this carrier.")}
              options={carrierSafetyRatingChoices}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="qualifiedAt"
              label={t("Qualified At")}
              placeholder={t("Qualified At")}
              description={t("Date the carrier most recently passed qualification.")}
            />
          </FormControl>
          {complianceStatus === "Disqualified" && (
            <FormControl cols="full">
              <TextareaField
                control={control}
                rules={{ required: true }}
                name="disqualifiedReason"
                label={t("Disqualification Reason")}
                placeholder={t("Reason the carrier was disqualified")}
                description={t("Required when a carrier is disqualified. Recorded for audit purposes.")}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Insurance Policies")}
        description={t("Active insurance coverage on file for this carrier.")}
      >
        <CarrierInsurancePolicyEditor />
      </FormSection>
    </div>
  );
}
