import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { useUSStates } from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { type LocationFormValues as FormValues } from "@/types/location";
import { type Control } from "react-hook-form";
import { AsyncSelectInput } from "../common/fields/async-select-input";
import { TextareaField } from "../common/fields/textarea";
import { Form, FormControl, FormGroup } from "../ui/form";

export function LocationInfoForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const {
    selectUSStates,
    isError: isUSStatesError,
    isLoading: isUsStatesLoading,
  } = useUSStates(open);

  return (
    <Form>
      <FormGroup>
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Identify the current operational status of the location."
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            readOnly
            label="Code"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Code"
            description="Enter a unique identifier or code for the location."
            maxLength={10}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            description="Specify the official name of the location."
          />
        </FormControl>
        <FormControl>
          <AsyncSelectInput
            name="locationCategoryId"
            rules={{ required: true }}
            control={control}
            link="/location-categories/"
            valueKey="name"
            label="Location Category"
            placeholder="Select Location Category"
            description="Choose the category that best describes the location's function or type."
            hasPopoutWindow
            popoutLink="/dispatch/location-categories/"
            popoutLinkLabel="Location Category"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="addressLine1"
            label="Address Line 1"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Address Line 1"
            description="Provide the primary street address or location detail."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="addressLine2"
            label="Address Line 2"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Address Line 2"
            description="Include any additional address information, such as suite or building number."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="city"
            label="City"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="City"
            description="Enter the city where the location is situated."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="stateId"
            control={control}
            rules={{ required: true }}
            label="State"
            maxOptions={selectUSStates.length}
            options={selectUSStates}
            isFetchError={isUSStatesError}
            isLoading={isUsStatesLoading}
            placeholder="Select State"
            description="Select the state or region for the location."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="postalCode"
            label="Postal Code"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Zip Code"
            description="Input the postal code associated with the location's address."
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Additional notes or comments for the location"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}
