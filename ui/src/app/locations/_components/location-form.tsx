import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { LocationCategoryAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export function LocationForm() {
  const { control } = useFormContext<LocationSchema>();

  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Defines the current operational status of the location."
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
          description="A unique identifier for the location."
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
          description="The official name of the location."
          maxLength={100}
        />
      </FormControl>
      <FormControl cols="full">
        <FormControl>
          <LocationCategoryAutocompleteField<LocationSchema>
            name="locationCategoryId"
            control={control}
            rules={{ required: true }}
            label="Location Category"
            placeholder="Select Location Category"
            description="Select the location category for the location."
          />
        </FormControl>
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Additional details or notes about the location."
        />
      </FormControl>
      <FormControl cols="full">
        <AddressField control={control} rules={{ required: true }} />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="addressLine2"
          label="Address Line 2"
          placeholder="Address Line 2"
          description="Additional address details, if applicable."
          maxLength={150}
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
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="stateId"
          label="State"
          placeholder="State"
          description="The U.S. state where the location is situated."
          options={usStateOptions}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="postalCode"
          label="Postal Code"
          placeholder="Postal Code"
          description="The ZIP code for the location."
          rules={{ required: true }}
          maxLength={150}
        />
      </FormControl>
    </FormGroup>
  );
}
