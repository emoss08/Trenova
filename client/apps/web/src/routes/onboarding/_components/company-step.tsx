import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SectionPanel } from "@/components/section-panel";
import { timezoneGroupedChoices } from "@/lib/choices";
import type { OnboardingFormValues } from "@/types/onboarding";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Control } from "react-hook-form";

export function CompanyStep({
  control,
  onStateLabelChange,
}: {
  control: Control<OnboardingFormValues>;
  onStateLabelChange: (label: string) => void;
}) {
  const t = useT();

  return (
    <SectionPanel
      title={t("Company profile")}
      help={t(
        "Appears on invoices, rate confirmations and other documents, and sets the clock the organization's reports use.",
      )}
    >
      <div className="p-4">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              name="organization.name"
              rules={{ required: true }}
              label={t("Company name")}
              placeholder={t("Enter company name")}
              autoComplete="organization"
            />
          </FormControl>
          <FormControl cols="full">
            <SelectField
              control={control}
              rules={{ required: true }}
              name="organization.timezone"
              label={t("Timezone")}
              placeholder={t("Select timezone")}
              groups={timezoneGroupedChoices}
              renderOption={(option) => (
                <span className="flex w-full items-center justify-between gap-3">
                  <span>{t(option.label)}</span>
                  {option.description && (
                    <span className="text-muted-foreground text-xs">{t(option.description)}</span>
                  )}
                </span>
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="organization.addressLine1"
              rules={{ required: true }}
              label={t("Street address")}
              placeholder={t("Enter address")}
              autoComplete="address-line1"
            />
          </FormControl>
          <FormControl cols={1}>
            <InputField
              control={control}
              name="organization.city"
              rules={{ required: true }}
              label={t("City")}
              placeholder={t("Enter city")}
              autoComplete="address-level2"
            />
          </FormControl>
          <FormControl cols={1}>
            <UsStateAutocompleteField
              control={control}
              name="organization.stateId"
              rules={{ required: true }}
              label={t("State")}
              placeholder={t("State")}
              onOptionChange={(option) => onStateLabelChange(option?.label ?? "")}
            />
          </FormControl>
          <FormControl cols={1}>
            <InputField
              control={control}
              name="organization.postalCode"
              rules={{ required: true }}
              label={t("ZIP code")}
              placeholder={t("Enter ZIP code")}
              autoComplete="postal-code"
              inputMode="numeric"
            />
          </FormControl>
          <FormControl cols={1} className="hidden md:block" aria-hidden="true">
            {null}
          </FormControl>
          <FormControl cols={1}>
            <InputField
              control={control}
              name="organization.scacCode"
              label={t("SCAC code")}
              placeholder={t("Optional")}
              description={t("Your Standard Carrier Alpha Code, if you have one.")}
              maxLength={4}
            />
          </FormControl>
          <FormControl cols={1}>
            <InputField
              control={control}
              name="organization.dotNumber"
              label={t("DOT number")}
              placeholder={t("Optional")}
              description={t("Your USDOT number, if you have one.")}
              inputMode="numeric"
              maxLength={8}
            />
          </FormControl>
        </FormGroup>
      </div>
    </SectionPanel>
  );
}
