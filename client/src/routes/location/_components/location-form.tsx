import {
  LocationCategoryAutocompleteField,
  UsStateAutocompleteField,
} from "@/components/autocomplete-fields";
import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { Location } from "@/types/location";
import { useFormContext } from "react-hook-form";

export function LocationForm() {
  const { control } = useFormContext<Location>();

  return (
    <div className="space-y-6">
      <FormSection
        title="General Information"
        description="Basic identification for this location."
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label="Status"
              placeholder="Status"
              description="The current status of the location."
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
              description="A unique code for this location."
              maxLength={10}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Name"
              placeholder="Name"
              description="The name of the location."
              maxLength={255}
            />
          </FormControl>
          <FormControl>
            <LocationCategoryAutocompleteField
              control={control}
              rules={{ required: true }}
              name="locationCategoryId"
              label="Location Category"
              placeholder="Location Category"
              description="The category this location belongs to."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="description"
              label="Description"
              placeholder="Description"
              description="A brief description of this location."
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection title="Address" description="Location address and geographic details.">
        <FormGroup cols={2}>
          <FormControl cols="full" id="address-field-container">
            <AddressField control={control} />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label="Address Line 2"
              placeholder="Address Line 2"
              description="Additional address details, if applicable."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="city"
              rules={{ required: true }}
              label="City"
              placeholder="City"
              description="The city where the location is situated."
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label="State"
              placeholder="State"
              description="The U.S. state where the location is situated."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              rules={{ required: true }}
              control={control}
              name="postalCode"
              label="Postal Code"
              placeholder="Postal Code"
              description="The ZIP code for the location."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
