import { HazardousMaterialAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  hazardousClassChoices,
  segregationDistanceUnitChoices,
  segregationTypeChoices,
  statusChoices,
} from "@/lib/choices";
import type { HazmatSegregationRule } from "@/types/hazmat-segregation-rule";
import { useFormContext, useWatch } from "react-hook-form";

export function HazmatSegregationRuleForm({
  disabled,
}: {
  disabled?: boolean;
}) {
  const { control } = useFormContext<HazmatSegregationRule>();

  const [hasExceptions, segregationType] = useWatch({
    control,
    name: ["hasExceptions", "segregationType"],
  });

  const showDistanceOptions = segregationType === "Distance";

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the segregation rule"
          options={statusChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="Human-readable name for the segregation rule"
          disabled={disabled}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Detailed description for this rule"
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="classA"
          label="Class A"
          placeholder="Class A"
          description="First hazardous material class"
          options={hazardousClassChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="classB"
          label="Class B"
          placeholder="Class B"
          description="Second hazardous material class"
          options={hazardousClassChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <HazardousMaterialAutocompleteField<HazmatSegregationRule>
          name="hazmatAId"
          control={control}
          label="Hazardous Material A"
          clearable
          placeholder="Select Hazardous Material A"
          description="Optional specific hazardous material"
        />
      </FormControl>
      <FormControl>
        <HazardousMaterialAutocompleteField<HazmatSegregationRule>
          name="hazmatBId"
          control={control}
          label="Hazardous Material B"
          clearable
          placeholder="Select Hazardous Material B"
          description="Optional specific hazardous material"
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="segregationType"
          label="Segregation Type"
          placeholder="Segregation Type"
          description="Type of segregation required"
          options={segregationTypeChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      {showDistanceOptions && (
        <>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: showDistanceOptions }}
              name="minimumDistance"
              label="Minimum Distance"
              placeholder="Minimum Distance"
              description="Minimum required distance"
              min={0}
              step={0.1}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: showDistanceOptions }}
              name="distanceUnit"
              label="Distance Unit"
              placeholder="Distance Unit"
              description="Measurement unit for minimum distance"
              options={segregationDistanceUnitChoices}
              isReadOnly={disabled}
            />
          </FormControl>
        </>
      )}
      <FormControl cols="full">
        <SwitchField
          control={control}
          outlined
          name="hasExceptions"
          label="Has Exceptions"
          description="Indicates whether exceptions to this rule exist"
          disabled={disabled}
        />
      </FormControl>
      {hasExceptions && (
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="exceptionNotes"
            label="Exception Notes"
            placeholder="Exception Notes"
            description="Document exceptions or special cases"
            rules={{ required: hasExceptions }}
            disabled={disabled}
          />
        </FormControl>
      )}
      <FormControl>
        <InputField
          control={control}
          name="referenceCode"
          label="Reference Code"
          placeholder="49 CFR 177.848"
          description="Regulatory code reference"
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="regulationSource"
          label="Regulation Source"
          placeholder="DOT"
          description="Source of the regulation"
          disabled={disabled}
        />
      </FormControl>
    </FormGroup>
  );
}
