import { useT } from "@trenova/shared/i18n/use-t";
import { HazardousMaterialAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  hazardousClassChoices,
  segregationDistanceUnitChoices,
  segregationTypeChoices,
  statusChoices,
} from "@/lib/choices";
import type { HazmatSegregationRule } from "@/types/hazmat-segregation-rule";
import { useFormContext, useWatch } from "react-hook-form";

export function HazmatSegregationRuleForm({ disabled }: { disabled?: boolean }) {
  const t = useT();

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
          label={t("Status")}
          placeholder={t("Status")}
          description={t("The status of the segregation rule")}
          options={statusChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Name")}
          placeholder={t("Name")}
          description={t("Human-readable name for the segregation rule")}
          disabled={disabled}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("Detailed description for this rule")}
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="classA"
          label={t("Class A")}
          placeholder={t("Class A")}
          description={t("First hazardous material class")}
          options={hazardousClassChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="classB"
          label={t("Class B")}
          placeholder={t("Class B")}
          description={t("Second hazardous material class")}
          options={hazardousClassChoices}
          isReadOnly={disabled}
        />
      </FormControl>
      <FormControl>
        <HazardousMaterialAutocompleteField<HazmatSegregationRule>
          name="hazmatAId"
          control={control}
          label={t("Hazardous Material A")}
          clearable
          placeholder={t("Select Hazardous Material A")}
          description={t("Optional specific hazardous material")}
        />
      </FormControl>
      <FormControl>
        <HazardousMaterialAutocompleteField<HazmatSegregationRule>
          name="hazmatBId"
          control={control}
          label={t("Hazardous Material B")}
          clearable
          placeholder={t("Select Hazardous Material B")}
          description={t("Optional specific hazardous material")}
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="segregationType"
          label={t("Segregation Type")}
          placeholder={t("Segregation Type")}
          description={t("Type of segregation required")}
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
              label={t("Minimum Distance")}
              placeholder={t("Minimum Distance")}
              description={t("Minimum required distance")}
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
              label={t("Distance Unit")}
              placeholder={t("Distance Unit")}
              description={t("Measurement unit for minimum distance")}
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
          label={t("Has Exceptions")}
          description={t("Indicates whether exceptions to this rule exist")}
          disabled={disabled}
        />
      </FormControl>
      {hasExceptions && (
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="exceptionNotes"
            label={t("Exception Notes")}
            placeholder={t("Exception Notes")}
            description={t("Document exceptions or special cases")}
            rules={{ required: hasExceptions }}
            disabled={disabled}
          />
        </FormControl>
      )}
      <FormControl>
        <InputField
          control={control}
          name="referenceCode"
          label={t("Reference Code")}
          placeholder={t("49 CFR 177.848")}
          description={t("Regulatory code reference")}
          disabled={disabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="regulationSource"
          label={t("Regulation Source")}
          placeholder={t("DOT")}
          description={t("Source of the regulation")}
          disabled={disabled}
        />
      </FormControl>
    </FormGroup>
  );
}
