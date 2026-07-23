import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  facilityTypeChoices,
  locationCategoryTypeChoices,
} from "@/lib/choices";
import type { LocationCategory } from "@/types/location-category";
import { useFormContext } from "react-hook-form";

export function LocationCategoryForm() {
  const { control } = useFormContext<LocationCategory>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the location category"
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="type"
          label="Type"
          placeholder="Type"
          description="The type of location category"
          options={locationCategoryTypeChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="facilityType"
          label="Facility Type"
          placeholder="Facility Type"
          description="The facility type of the location category"
          options={facilityTypeChoices}
          isClearable
        />
      </FormControl>
      <FormControl>
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="The color of the location category"
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
      <FormControl>
        <SwitchField
          control={control}
          name="hasSecureParking"
          label="Secure Parking"
          description="Whether this location has secure parking"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="requiresAppointment"
          label="Requires Appointment"
          description="Whether this location requires an appointment"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="allowsOvernight"
          label="Allows Overnight"
          description="Whether this location allows overnight stays"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="hasRestroom"
          label="Has Restroom"
          description="Whether this location has restroom facilities"
        />
      </FormControl>
    </FormGroup>
  );
}
