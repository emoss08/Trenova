/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



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
