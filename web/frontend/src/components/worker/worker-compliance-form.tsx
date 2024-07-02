import { useUSStates } from "@/hooks/useQueries";
import { workerEndorsementChoices } from "@/lib/choices";
import { useFormContext } from "react-hook-form";
import { DatepickerField } from "../common/fields/date-picker";
import { InputField } from "../common/fields/input";
import { SelectInput } from "../common/fields/select-input";
import { FormControl, FormGroup } from "../ui/form";

export default function WorkerComplianceInformation() {
  const { control } = useFormContext();
  const {
    selectUSStates,
    isLoading: isUsStatesLoading,
    isError: isUSStatesError,
  } = useUSStates();

  return (
    <>
      <FormGroup className="grid gap-x-6 md:grid-cols-3 lg:grid-cols-3">
        <FormControl className="col-span-1">
          <InputField
            control={control}
            name="workerProfile.licenseNumber"
            label="License Number"
            rules={{ required: true }}
            placeholder="Enter License Number"
            description="The worker's driver's license number."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <SelectInput
            name="workerProfile.stateId"
            control={control}
            label="License State"
            rules={{ required: true }}
            maxOptions={selectUSStates.length}
            options={selectUSStates}
            isFetchError={isUSStatesError}
            isLoading={isUsStatesLoading}
            placeholder="Select State"
            description="The state in which the worker's driver's license was issued."
          />
        </FormControl>
        <FormControl className="col-span-1">
          <SelectInput
            name="workerProfile.endorsements"
            control={control}
            rules={{ required: true }}
            label="Endorsements"
            options={workerEndorsementChoices}
            placeholder="Select Endorsements"
            description="The endorsements on the worker's driver's license."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="workerProfile.dateOfBirth"
            control={control}
            label="Date of Birth"
            placeholder="Select Date"
            description="The worker's date of birth."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="workerProfile.hazmatExpirationDate"
            control={control}
            label="Hazmat Expiration Date"
            placeholder="Select Date"
            description="The expiration date of the worker's hazmat certification."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="workerProfile.hireDate"
            control={control}
            label="Hire Date"
            placeholder="Select Date"
            description="The date the worker was hired."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="workerProfile.terminationDate"
            control={control}
            label="Termination Date"
            placeholder="Select Date"
            description="The date the worker was terminated."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="workerProfile.physicalDueDate"
            control={control}
            label="Physical Due Date"
            placeholder="Select Date"
            description="The date the worker's physical is due."
          />
        </FormControl>
        <FormControl>
          <DatepickerField
            name="workerProfile.mvrDueDate"
            control={control}
            label="MVR Due Date"
            placeholder="Select Date"
            description="The date the worker's MVR is due."
          />
        </FormControl>
      </FormGroup>
    </>
  );
}
