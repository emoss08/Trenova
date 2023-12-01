/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { useUSStates, useUsers } from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { CustomerFormValues as FormValues } from "@/types/customer";
import { Control } from "react-hook-form";
import { CheckboxInput } from "../common/fields/checkbox";

export function CustomerInfoForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const {
    selectUsersData,
    isError: isUserError,
    isLoading: isUsersLoading,
  } = useUsers(open);

  const {
    selectUSStates,
    isError: isUSStatesError,
    isLoading: isUsStatesLoading,
  } = useUSStates(open);

  return (
    <>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 my-4">
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
            <SelectInput
              name="status"
              rules={{ required: true }}
              control={control}
              label="Status"
              options={statusChoices}
              placeholder="Select Status"
              description="Identify the current operational status of the customer."
              isClearable={false}
            />
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label="Code"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Code"
              description="Enter a unique identifier or code for the customer."
              maxLength={10}
            />
          </div>
        </div>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 my-4">
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Name"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Name"
              description="Specify the official name of the customer."
              maxLength={10}
            />
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
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
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
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
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              rules={{ required: true }}
              name="city"
              label="City"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="City"
              description="Enter the city where the customer is situated."
            />
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
            <SelectInput
              name="state"
              control={control}
              rules={{ required: true }}
              label="State"
              options={selectUSStates}
              isFetchError={isUSStatesError}
              isLoading={isUsStatesLoading}
              placeholder="Select State"
              description="Select the state or region for the customer."
            />
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              rules={{ required: true }}
              name="zipCode"
              label="Zip Code"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Zip Code"
              description="Input the postal code associated with the customer's address."
            />
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
          <div className="min-h-[4em]">
            <SelectInput
              name="advocate"
              control={control}
              rules={{ required: true }}
              label="Customer Advocate"
              options={selectUsersData}
              isFetchError={isUserError}
              isLoading={isUsersLoading}
              placeholder="Select Customer Advocate"
              description="Assign a customer advocate from your team."
            />
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5 mt-6">
          <div className="min-h-[4em]">
            <CheckboxInput
              control={control}
              label="Has Customer Protal?"
              name="hasCustomerPortal"
              description="Indicate whether the customer has access to the online portal for managing their account and services."
            />
          </div>
        </div>
        <div className="flex flex-col justify-between w-full max-w-sm gap-0.5 mt-6">
          <div className="min-h-[4em]">
            <CheckboxInput
              control={control}
              label="Automatic Billing Readiness"
              name="autoMarkReadyToBill"
              description="Enable automatic marking of billing readiness."
            />
          </div>
        </div>
      </div>
    </>
  );
}
