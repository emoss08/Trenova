/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  hazardousClassChoices,
  packingGroupChoices,
  statusChoices,
} from "@/lib/choices";
import { HazardousMaterialSchema } from "@/lib/schemas/hazardous-material-schema";
import { useFormContext } from "react-hook-form";

export function HazardousMaterialForm() {
  const { control } = useFormContext<HazardousMaterialSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the hazardous material"
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
          description="The code of the hazardous material"
          maxLength={10}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the hazardous material"
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
          description="The description of the hazardous material"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="unNumber"
          label="UN Number"
          placeholder="UN Number"
          description="The UN number of the hazardous material"
          maxLength={4}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="casNumber"
          label="CAS Number"
          placeholder="CAS Number"
          description="The CAS number of the hazardous material"
          maxLength={10}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="packingGroup"
          label="Packing Group"
          placeholder="Packing Group"
          description="The packing group of the hazardous material"
          options={packingGroupChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="class"
          label="Class"
          placeholder="Class"
          description="The class of the hazardous material"
          options={hazardousClassChoices}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="properShippingName"
          label="Proper Shipping Name"
          placeholder="Proper Shipping Name"
          description="The proper shipping name of the hazardous material"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="handlingInstructions"
          label="Handling Instructions"
          placeholder="Handling Instructions"
          description="The handling instructions of the hazardous material"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="emergencyContact"
          label="Emergency Contact"
          placeholder="Emergency Contact"
          description="The emergency contact of the hazardous material"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="emergencyContactPhoneNumber"
          label="Emergency Contact Phone Number"
          placeholder="Emergency Contact Phone Number"
          description="The emergency contact phone number of the hazardous material"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          outlined
          size="sm"
          control={control}
          name="placardRequired"
          label="Placard Required"
          description="Whether the hazardous material requires a placard"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          outlined
          size="sm"
          control={control}
          name="isReportableQuantity"
          label="Is Reportable Quantity"
          description="Whether the hazardous material is a reportable quantity"
        />
      </FormControl>
    </FormGroup>
  );
}
