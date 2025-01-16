import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { queries } from "@/lib/queries";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export default function WorkersPersonalInformationForm() {
  const { control } = useFormContext<WorkerSchema>();
  const usStates = useQuery({
    ...queries.usState.options(),
  });

  const usStateOptions = usStates.data?.results ?? [];

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">Personal Information</h2>
        <p className="text-xs text-muted-foreground">
          The following information is required for the worker to be able to
          work in the United States.
        </p>
      </div>
      <Separator className="mt-2" />
      <div className="p-4">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="firstName"
              label="First Name"
              placeholder="First Name"
              description="The first name of the worker"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="lastName"
              label="Last Name"
              placeholder="Last Name"
              description="The last name of the worker"
            />
          </FormControl>
        </FormGroup>
        <FormGroup className="mt-2" cols={2}>
          <FormControl cols={2}>
            <InputField
              control={control}
              rules={{ required: true }}
              name="addressLine1"
              label="Address Line 1"
              placeholder="Address Line 1"
              description="The first line of the worker's address"
            />
          </FormControl>
          <FormControl cols={2}>
            <InputField
              control={control}
              name="addressLine2"
              label="Address Line 2"
              placeholder="Address Line 2"
              description="The second line of the worker's address"
            />
          </FormControl>
        </FormGroup>
        <FormGroup className="mt-2" cols={4}>
          <FormControl cols={2}>
            <InputField
              control={control}
              name="city"
              label="City"
              placeholder="City"
              description="The city of the worker's address"
            />
          </FormControl>
          <FormControl cols={2}>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="stateId"
              label="State"
              placeholder="State"
              description="The state of the worker"
              options={usStateOptions}
              isLoading={usStates.isLoading}
              isFetchError={usStates.isError}
            />
          </FormControl>
          <FormControl cols={4}>
            <InputField
              control={control}
              rules={{ required: true }}
              name="postalCode"
              label="Postal Code"
              placeholder="Postal Code"
              description="The postal code of the worker's address"
            />
          </FormControl>
        </FormGroup>
      </div>
    </div>
  );
}
