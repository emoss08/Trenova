/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { useUSStates } from "@/hooks/useQueries";
import { statusChoices } from "@/lib/choices";
import { type CustomerFormValues as FormValues } from "@/types/customer";
import { useFormContext } from "react-hook-form";
import { CheckboxInput } from "./common/fields/checkbox";
import { LocationAutoComplete } from "./ui/autocomplete";
import { FormControl, FormGroup } from "./ui/form";

export function CustomerInfoForm({ open }: { open: boolean }) {
  const { control } = useFormContext<FormValues>();
  const {
    selectUSStates,
    isLoading: isUsStatesLoading,
    isError: isUSStatesError,
  } = useUSStates(open);

  return (
    <>
      <FormGroup>
        <FormControl>
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
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            readOnly
            placeholder="Code"
            description="Unique identifier for the customer."
            maxLength={10}
          />
        </FormControl>
      </FormGroup>
      <FormGroup>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Name"
            description="Specify the official name of the customer."
          />
        </FormControl>
        <FormControl>
          <LocationAutoComplete
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
            description="Enter the city where the customer is situated."
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="stateId"
            control={control}
            rules={{ required: true }}
            label="State"
            options={selectUSStates}
            isFetchError={isUSStatesError}
            isLoading={isUsStatesLoading}
            placeholder="Select State"
            description="Select the state or region for the customer."
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
            placeholder="Postal Code"
            description="Input the postal code associated with the customer's address."
          />
        </FormControl>
        <FormControl>
          <CheckboxInput
            control={control}
            label="Has Customer Protal?"
            disabled
            name="hasCustomerPortal"
            description="Indicate whether the customer has access to the online portal for managing their account and services."
          />
        </FormControl>
        <FormControl>
          <CheckboxInput
            control={control}
            label="Automatic Billing Readiness"
            name="autoMarkReadyToBill"
            description="Enable automatic marking of billing readiness."
          />
        </FormControl>
      </FormGroup>
    </>
  );
}
