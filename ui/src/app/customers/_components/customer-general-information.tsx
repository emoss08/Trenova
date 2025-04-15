import { AddressField } from "@/components/fields/address-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export default function CustomerForm() {
  const { control } = useFormContext<CustomerSchema>();

  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">General Information</h2>
        <p className="text-xs text-muted-foreground">
          Enter essential customer identification details including status,
          contact information, and physical address to establish the customer
          profile for shipment processing and billing.
        </p>
      </div>
      <Separator className="mt-2" />
      <div className="p-4">
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
              rules={{ required: true }}
              control={control}
              name="postalCode"
              label="Postal Code"
              placeholder="Postal Code"
              description="The ZIP code for the customer."
            />
          </FormControl>
        </FormGroup>
      </div>
    </div>
  );
}
