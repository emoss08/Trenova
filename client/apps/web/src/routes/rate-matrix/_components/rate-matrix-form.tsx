import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { rateMatrixValueTypeChoices, statusChoices } from "@/lib/choices";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { RateMatrix } from "@trenova/shared/types/rate";
import { useFormContext } from "react-hook-form";

export function RateMatrixForm() {
  const t = useT();

  const { control } = useFormContext<RateMatrix>();

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title={t("General information")}
        description={t("How this tariff grid is identified and whether it is live.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t(
                "An inactive matrix stops pricing, and every lane pointing at it stops with it",
              )}
              options={statusChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label={t("Code")}
              placeholder={t("LTL-2025-Q3")}
              description={t("The short name lanes refer to")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("LTL base tariff, Q3 2025")}
              description={t("What this tariff is called out loud")}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("Zone-to-zone base rates by weight break, published July 2025")}
              description={t(
                "Where this tariff came from, so the next person knows what they are amending",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Values")}
        description={t("What the numbers in the grid mean when a tariff reads them.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="valueType"
              label={t("Value type")}
              placeholder={t("Select value type")}
              description={t(
                "Each cell's number is read as this: a flat amount, a rate per unit, a percent or a plain number. Money is in the contract's currency.",
              )}
              options={rateMatrixValueTypeChoices}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
