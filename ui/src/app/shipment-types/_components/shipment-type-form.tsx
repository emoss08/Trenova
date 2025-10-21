/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { useFormContext } from "react-hook-form";

export function ShipmentTypeForm() {
  const { control } = useFormContext<ShipmentTypeSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the shipment type"
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="The code of the shipment type"
          maxLength={10}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="The description of the shipment type"
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="The color of the shipment type"
        />
      </FormControl>
    </FormGroup>
  );
}
