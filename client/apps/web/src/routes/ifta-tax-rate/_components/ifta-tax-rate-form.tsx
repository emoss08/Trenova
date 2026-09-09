import { IftaJurisdictionSelectField } from "@/components/fields/ifta-jurisdiction-select-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { InfoPopover } from "@/components/info-popover";
import { iftaFuelTypeChoices, iftaQuarterChoices } from "@/lib/choices";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { IftaTaxRateFormValues } from "@trenova/shared/types/ifta-tax-rate";
import { IFTA_MAX_YEAR, IFTA_MIN_YEAR } from "@trenova/shared/types/ifta-tax-rate";
import { useFormContext } from "react-hook-form";

export function IftaTaxRateForm({ isEdit }: { isEdit: boolean }) {
  const { control } = useFormContext<IftaTaxRateFormValues>();

  return (
    <div className="flex flex-col gap-4">
      <FormSection
        title="Period & product"
        description={
          isEdit
            ? "The period, jurisdiction and fuel type identify the rate. Changing them replaces a different rate rather than moving this one."
            : "One rate per jurisdiction, quarter and fuel type. Saving over an existing combination replaces it."
        }
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="year"
              label="Year"
              placeholder="2026"
              min={IFTA_MIN_YEAR}
              max={IFTA_MAX_YEAR}
              rules={{ required: true }}
              description="The calendar year of the quarter the rate was published for."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="quarter"
              label="Quarter"
              options={iftaQuarterChoices}
              rules={{ required: true }}
              placeholder="Select a quarter"
              description="Rates change every quarter; the matrix is published shortly before each one."
            />
          </FormControl>
          <FormControl>
            <IftaJurisdictionSelectField<IftaTaxRateFormValues>
              control={control}
              name="jurisdictionId"
              label="Jurisdiction"
              rules={{ required: true }}
              placeholder="Select a jurisdiction"
              description="The state or province that levies the tax."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="fuelType"
              label="Fuel type"
              options={iftaFuelTypeChoices}
              rules={{ required: true }}
              placeholder="Select a fuel type"
              description="The matrix lists a separate rate for each fuel; DEF, reefer and other never carry one."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection title="Rate" description="USD per US gallon, as printed in the matrix.">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="ratePerGallon"
              label="Rate per gallon"
              placeholder="0.2000"
              inputMode="decimal"
              rules={{ required: true }}
              description="Up to four decimals. Zero is a published rate, not a missing one."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="surchargeRatePerGallon"
              label={
                <span className="inline-flex items-center gap-1">
                  Surcharge per gallon
                  <InfoPopover title="Surcharge">
                    Indiana, Kentucky and Virginia levy a surcharge on fuel consumed in the
                    jurisdiction, on top of the tax on fuel bought there. It is charged on taxable
                    gallons and is never a credit. Leave it empty for jurisdictions that publish no
                    surcharge.
                  </InfoPopover>
                </span>
              }
              placeholder="0.1100"
              inputMode="decimal"
              description="Only for jurisdictions that publish one. Empty is stored as zero."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title="Source"
        description="Where the figure came from, so a reviewer can check it against the published matrix."
      >
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="sourceUrl"
              label="Source URL"
              placeholder="https://www.iftach.org/taxmatrix4/"
              maxLength={500}
              description="A link to the matrix page or bulletin the rate was taken from."
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="sourceNote"
              label="Source note"
              placeholder="e.g. IFTA Inc. tax rate matrix, Q3 2026, downloaded 12 Jul"
              maxLength={500}
              description="The edition or bulletin, and anything odd about how the figure was read."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
