import { useT } from "@trenova/shared/i18n/use-t";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { facilityTypeChoices, locationCategoryTypeChoices } from "@/lib/choices";
import type { LocationCategory } from "@/types/location-category";
import { useFormContext } from "react-hook-form";

export function LocationCategoryForm() {
  const t = useT();

  const { control } = useFormContext<LocationCategory>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Name")}
          placeholder={t("Name")}
          description={t("The name of the location category")}
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="type"
          label={t("Type")}
          placeholder={t("Type")}
          description={t("The type of location category")}
          options={locationCategoryTypeChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="facilityType"
          label={t("Facility Type")}
          placeholder={t("Facility Type")}
          description={t("The facility type of the location category")}
          options={facilityTypeChoices}
          isClearable
        />
      </FormControl>
      <FormControl>
        <ColorField
          control={control}
          name="color"
          label={t("Color")}
          description={t("The color of the location category")}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("The description of the location category")}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="hasSecureParking"
          label={t("Secure Parking")}
          description={t("Whether this location has secure parking")}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="requiresAppointment"
          label={t("Requires Appointment")}
          description={t("Whether this location requires an appointment")}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="allowsOvernight"
          label={t("Allows Overnight")}
          description={t("Whether this location allows overnight stays")}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="hasRestroom"
          label={t("Has Restroom")}
          description={t("Whether this location has restroom facilities")}
        />
      </FormControl>
    </FormGroup>
  );
}
