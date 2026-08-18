import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { rateMatrixValueKindChoices, rateRoundingModeChoices, statusChoices } from "@/lib/choices";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { RateMatrix } from "@trenova/shared/types/rate";
import { useFormContext } from "react-hook-form";

export function RateMatrixForm() {
  const { control } = useFormContext<RateMatrix>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="An inactive matrix stops pricing, and every lane pointing at it stops with it"
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="valueKind"
          label="Rates are"
          placeholder="Rates are"
          description="What each number in the grid means — the same grid is a per-mile tariff or a discount table depending on this"
          options={rateMatrixValueKindChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="LTL-2025-Q3"
          description="The short name lanes refer to"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="LTL base tariff, Q3 2025"
          description="What this tariff is called out loud"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Zone-to-zone base rates by weight break, published July 2025"
          description="Where this tariff came from, so the next person knows what they are amending"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="currency"
          label="Currency"
          placeholder="USD"
          description="The currency the numbers in the grid are in, converted at rating time when it differs from the contract's"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="roundingMode"
          label="Rounding"
          placeholder="Rounding"
          description="How a looked-up rate is rounded before it becomes a charge"
          options={rateRoundingModeChoices}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          rules={{ required: true }}
          name="roundingPrecision"
          label="Decimal places"
          placeholder="2"
          description="How many decimals survive the rounding"
        />
      </FormControl>
    </FormGroup>
  );
}
