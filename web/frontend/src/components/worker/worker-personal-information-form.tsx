import { useUSStates } from "@/hooks/useQueries";
import { statusChoices, workerTypeChoices } from "@/lib/choices";
import {
  clearProfilePicture,
  postUserProfilePicture,
} from "@/services/UserRequestService";
import { Worker } from "@/types/worker";
import { useFormContext } from "react-hook-form";
import { InputField } from "../common/fields/input";
import { SelectInput } from "../common/fields/select-input";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  ImageUploader,
} from "../ui/avatar";
import { FormControl, FormGroup } from "../ui/form";

export default function WorkerPersonalInformation() {
  const { control, getValues } = useFormContext();

  const {
    selectUSStates,
    isLoading: isUsStatesLoading,
    isError: isUSStatesError,
  } = useUSStates();

  const initials = `${getValues("firstName")?.charAt(0)}${getValues(
    "lastName",
  )?.charAt(0)}`;

  return (
    <>
      <div className="col-span-full my-5 flex items-center gap-x-8">
        <Avatar className="size-24 flex-none rounded-lg">
          <AvatarImage src={getValues("profilePictureUrl") || ""} />
          <AvatarFallback className="size-24 flex-none rounded-lg">
            {initials}
          </AvatarFallback>
        </Avatar>
        <ImageUploader
          callback={postUserProfilePicture}
          successCallback={(data: Worker) => {
            console.info(data);
            return "What";
          }}
          removeFileCallback={clearProfilePicture}
          removeSuccessCallback={() => {
            return "what";
          }}
        />
      </div>
      <FormGroup className="grid gap-x-6 md:grid-cols-3 lg:grid-cols-2">
        <FormControl className="col-span-1">
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Indicates the current operational status of the worker."
            isClearable={false}
          />
        </FormControl>
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="code"
            label="Code"
            rules={{ required: true }}
            placeholder="Enter Code"
            description="A unique code assigned to the worker."
          />
        </FormControl>
      </FormGroup>
      <FormGroup className="grid gap-x-6 md:grid-cols-3 lg:grid-cols-3">
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="firstName"
            label="First Name"
            rules={{ required: true }}
            placeholder="Enter First Name"
            description="The first name of the worker."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="lastName"
            label="Last Name"
            rules={{ required: true }}
            placeholder="Enter Last Name"
            description="The last name of the worker."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <SelectInput
            name="workerType"
            rules={{ required: true }}
            control={control}
            label="Worker Type"
            options={workerTypeChoices}
            placeholder="Select Worker Type"
            description="The type of employment relationship with the worker."
            isClearable={false}
          />
        </FormControl>
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="addressLine1"
            label="Address Line 1"
            placeholder="Enter Address Line 1"
            description="The primary street address of the worker."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="addressLine2"
            label="Address Line 2"
            placeholder="Enter Address Line 2"
            description="An additional street address line for the worker."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="city"
            label="City"
            placeholder="Enter City"
            description="The city where the worker resides."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="postalCode"
            label="Postal Code"
            placeholder="Enter Postal Code"
            description="The postal code for the worker's address."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="stateId"
            control={control}
            label="State"
            options={selectUSStates}
            isFetchError={isUSStatesError}
            isLoading={isUsStatesLoading}
            placeholder="Select State"
            description="The state or region where the worker is located."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <SelectInput
            name="managerId"
            control={control}
            label="Manager"
            options={selectUSStates}
            isFetchError={isUSStatesError}
            isLoading={isUsStatesLoading}
            placeholder="Select Manager"
            description="The manager responsible for the worker."
          />
        </FormControl>
      </FormGroup>
    </>
  );
}
