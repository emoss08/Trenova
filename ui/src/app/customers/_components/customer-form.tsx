import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export function CustomerForm() {
  const { control } = useFormContext<CustomerSchema>();

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
          description="Defines the current operational status of the customer."
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
          description="A unique identifier for the customer."
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The official name of the customer."
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="Additional details or notes about the customer."
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="addressLine1"
          label="Address Line 1"
          placeholder="Address Line 1"
          description="The primary address for the customer."
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
          description="The city where the customer is situated."
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
          description="The U.S. state where the customer is situated."
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
          description="The ZIP code for the customer."
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          control={control}
          outlined
          name="autoMarkReadyToBill"
          label="Auto Mark Ready To Bill"
          description="Whether the shipments for this customer should automatically be marked as ready to bill"
        />
      </FormControl>
    </FormGroup>
  );
}
