/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { HazardousMaterialAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { statusChoices } from "@/lib/choices";
import { type CommoditySchema } from "@/lib/schemas/commodity-schema";
import { useFormContext } from "react-hook-form";

export function CommodityForm() {
  const { control } = useFormContext<CommoditySchema>();

  return (
    <FormGroup className="gap-y-3" cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Indicates whether the commodity is Active, Inactive, or Archived."
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
          description="The official name used to identify the commodity."
          maxLength={100}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          rules={{ required: true }}
          name="description"
          label="Description"
          placeholder="Description"
          description="Detailed information about the commodity's characteristics and handling requirements."
        />
      </FormControl>
      <FormControl>
        <HazardousMaterialAutocompleteField<CommoditySchema>
          name="hazardousMaterialId"
          control={control}
          label="Hazardous Material"
          clearable
          description="Select the hazardous material classification if this commodity contains regulated substances."
        />
      </FormControl>
      <FormControl>
        <InputField
          name="freightClass"
          control={control}
          label="Freight Class"
          placeholder="Freight Class"
          description="The NMFC code used for pricing and handling in LTL shipping."
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <NumberField
          name="minTemperature"
          control={control}
          label="Min Temperature"
          placeholder="Min Temperature"
          description="The lowest temperature (°F) at which the commodity can be safely transported."
          sideText="°F"
        />
      </FormControl>
      <FormControl>
        <NumberField
          name="maxTemperature"
          control={control}
          label="Max Temperature"
          placeholder="Max Temperature"
          description="The highest temperature (°F) at which the commodity can be safely transported."
          sideText="°F"
        />
      </FormControl>
      <FormControl>
        <NumberField
          name="weightPerUnit"
          control={control}
          label="Weight Per Unit"
          placeholder="Weight Per Unit"
          description="The weight (in pounds) of a single unit of this commodity. Used for calculating total load weight."
        />
      </FormControl>
      <FormControl>
        <NumberField
          name="linearFeetPerUnit"
          control={control}
          label="Linear Feet Per Unit"
          placeholder="Linear Feet Per Unit"
          description="The linear feet (in feet) of a single unit of this commodity. Used for calculating total load linear feet."
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          name="dotClassification"
          control={control}
          label="DOT Classification"
          placeholder="DOT Classification"
          description="The U.S. Department of Transportation classification used for regulatory compliance."
          maxLength={100}
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          name="stackable"
          control={control}
          label="Stackable"
          outlined
          size="sm"
          description="Indicates if the commodity can be safely stacked during transport or storage."
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          name="fragile"
          control={control}
          label="Fragile"
          outlined
          size="sm"
          description="Specifies whether the commodity is fragile and requires special handling."
        />
      </FormControl>
    </FormGroup>
  );
}
