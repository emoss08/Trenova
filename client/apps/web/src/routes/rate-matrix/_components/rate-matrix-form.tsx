import { useT } from "@trenova/shared/i18n/use-t";
import { FormulaTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { currencyChoices, rateRoundingModeChoices, statusChoices } from "@/lib/choices";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { RateMatrix } from "@trenova/shared/types/rate";
import { useFormContext } from "react-hook-form";

export function RateMatrixForm() {
  const t = useT();

  const { control } = useFormContext<RateMatrix>();

  return (
    <div className="space-y-6">
      <FormSection
        title={t("General Information")}
        description={t("How this tariff grid is identified and whether it is live.")}
        className="border-b pb-4"
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
        title={t("Pricing")}
        description={t(
          "What the numbers in the grid mean and how a looked-up rate becomes a charge.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <FormulaTemplateAutocompleteField<RateMatrix>
              control={control}
              rules={{ required: true }}
              name="formulaTemplateId"
              label={t("Rating Method")}
              placeholder={t("Select rating method")}
              description={t(
                "The formula template that says what each number in the grid means — the same grid is a per-mile tariff or a flat table depending on which template prices it",
              )}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="currency"
              label={t("Currency")}
              placeholder={t("Select currency")}
              description={t(
                "The currency the numbers in the grid are in, converted at rating time when it differs from the contract's",
              )}
              options={currencyChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="roundingMode"
              label={t("Rounding Mode")}
              placeholder={t("Select rounding")}
              description={t("How a looked-up rate is rounded before it becomes a charge")}
              options={rateRoundingModeChoices}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name="roundingPrecision"
              label={t("Rounding Precision")}
              placeholder="2"
              description={t("How many decimals survive the rounding")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
