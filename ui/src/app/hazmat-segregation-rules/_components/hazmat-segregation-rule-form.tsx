/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { HazardousMaterialAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import {
  hazardousClassChoices,
  segregationDistanceUnitChoices,
  segregationTypeChoices,
  statusChoices,
} from "@/lib/choices";
import { type HazmatSegregationRuleSchema } from "@/lib/schemas/hazmat-segregation-rule-schema";
import { SegregationType } from "@/types/hazmat-segregation-rule";
import { useFormContext, useWatch } from "react-hook-form";

export function HazmatSegregationRuleForm() {
  const { control } = useFormContext<HazmatSegregationRuleSchema>();
  const [hasExceptions, segregationType] = useWatch({
    control,
    name: ["hasExceptions", "segregationType"],
  });

  const showDistanceOptions = segregationType === SegregationType.Distance;
  const showExceptionNotes = Boolean(hasExceptions);

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
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="Human-readable name for the segregation rule (e.g., 'Class 1 - Class 2 Separation')"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Detailed description explaining the purpose and application of the rule"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="classA"
          label="Class A"
          placeholder="Class A"
          description="The first hazardous material class in the segregation pair"
          options={hazardousClassChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="classB"
          label="Class B"
          placeholder="Class B"
          description="The second hazardous material class in the segregation pair"
          options={hazardousClassChoices}
        />
      </FormControl>
      <FormControl>
        <HazardousMaterialAutocompleteField<HazmatSegregationRuleSchema>
          name="hazmatAId"
          control={control}
          label="Hazardous Material A"
          clearable
          placeholder="Select Hazardous Material A"
          description="Optional specific hazardous material identifier (used when rule applies to specific materials rather than entire classes)"
        />
      </FormControl>
      <FormControl>
        <HazardousMaterialAutocompleteField<HazmatSegregationRuleSchema>
          name="hazmatBId"
          control={control}
          label="Hazardous Material B"
          clearable
          placeholder="Select Hazardous Material B"
          description="Optional specific hazardous material identifier (used when rule applies to specific materials rather than entire classes)"
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="segregationType"
          label="Segregation Type"
          placeholder="Segregation Type"
          description="The type of segregation required between materials"
          options={segregationTypeChoices}
        />
      </FormControl>
      {showDistanceOptions && (
        <>
          <FormControl>
            <NumberField
              rules={{ required: showDistanceOptions }}
              control={control}
              name="minimumDistance"
              label="Minimum Distance"
              placeholder="Minimum Distance"
              description="Required minimum distance between materials when segregation type is 'Distance'"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: showDistanceOptions }}
              name="distanceUnit"
              label="Distance Unit"
              placeholder="Distance Unit"
              description="Unit of measurement for minimum distance (e.g., 'ft', 'm', 'in', 'cm')"
              options={segregationDistanceUnitChoices}
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
        />
      </FormControl>
      {showExceptionNotes && (
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="exceptionNotes"
            label="Exception Notes"
            placeholder="Exception Notes"
            description="Documentation of any exceptions or special cases for this rule"
            rules={{ required: showExceptionNotes }}
          />
        </FormControl>
      )}
      <FormControl>
        <InputField
          control={control}
          name="referenceCode"
          label="Reference Code"
          placeholder="Reference Code"
          description="Regulatory code reference (e.g., '49 CFR 177.848')"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="regulationSource"
          label="Regulation Source"
          placeholder="Regulation Source"
          description="Source of the regulation (e.g., 'DOT', 'FMCSA', 'PHMSA')"
        />
      </FormControl>
    </FormGroup>
  );
}
