import { CheckboxField } from "@/components/fields/checkbox-field";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  facilityTypeChoices,
  locationCategoryTypeChoices,
} from "@/lib/choices";
import { type LocationCategorySchema } from "@/lib/schemas/location-category-schema";
import { useFormContext } from "react-hook-form";

export function LocationCategoryForm() {
  const { control } = useFormContext<LocationCategorySchema>();

  return (
    <FormGroup cols={2}>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the location category"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="The description of the location category"
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="The color of the location category"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="type"
          label="Type"
          placeholder="Type"
          description="The type of the location category"
          options={locationCategoryTypeChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="facilityType"
          label="Facility Type"
          placeholder="Facility Type"
          description="The type of the facility"
          options={facilityTypeChoices}
        />
      </FormControl>
      <FormControl>
        <CheckboxField
          name="allowsOvernight"
          control={control}
          label="Allows Overnight"
          outlined
          description="Specifies whether the location allows overnight parking."
        />
      </FormControl>
      <FormControl>
        <CheckboxField
          name="hasSecureParking"
          control={control}
          label="Has Secure Parking"
          outlined
          description="Specifies whether the location has secure parking."
        />
      </FormControl>
      <FormControl>
        <CheckboxField
          name="requiresAppointment"
          control={control}
          label="Requires Appointment"
          outlined
          description="Specifies whether the location requires an appointment."
        />
      </FormControl>
      <FormControl>
        <CheckboxField
          name="hasRestroom"
          control={control}
          label="Has Restroom"
          outlined
          description="Specifies whether the location has a restroom."
        />
      </FormControl>
    </FormGroup>
  );
}
