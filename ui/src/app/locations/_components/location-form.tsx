import { AsyncSelectField } from "@/components/fields/async-select";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
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
        />
      </FormControl>
      <FormControl cols="full">
        <AsyncSelectField
          name="locationCategoryId"
          control={control}
          rules={{ required: true }}
          link="/location-categories"
          label="Location Category"
          placeholder="Select Location Category"
          description="Select the location category for the location."
          // TODO(wolfred): We need to change this to include the actual user permissions
          hasPermission
          hasPopoutWindow
          popoutLink="/dispatch/configurations/location-categories/"
          popoutLinkLabel="Location Category"
        />
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
        <InputField
          control={control}
          rules={{ required: true }}
          name="addressLine1"
          label="Address Line 1"
          placeholder="Address Line 1"
          description="The primary address for the location."
        />
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
        <SelectField
          control={control}
          rules={{ required: true }}
          name="stateId"
          label="State"
          placeholder="State"
          menuPlacement="top"
          description="The U.S. state where the location is situated."
          options={usStateOptions}
          isLoading={usStates.isLoading}
          isFetchError={usStates.isError}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="postalCode"
          label="Postal Code"
          placeholder="Postal Code"
          description="The ZIP code for the location."
        />
      </FormControl>
    </FormGroup>
  );
}
