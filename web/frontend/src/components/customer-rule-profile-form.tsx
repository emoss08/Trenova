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

import { useDocumentClass } from "@/hooks/useQueries";
import { BillingCycleChoices, type CustomerFormValues } from "@/types/customer";
import { useFormContext } from "react-hook-form";
import { SelectInput } from "./common/fields/select-input";
import { FormControl, FormGroup } from "./ui/form";

export function CustomerRuleProfileForm({ open }: { open: boolean }) {
  const { control } = useFormContext<CustomerFormValues>();

  const {
    selectDocumentClassData,
    isError: isDocumentClassError,
    isLoading: isDocumentClassLoading,
  } = useDocumentClass(open);

  return (
    <FormGroup className="my-4 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <FormControl>
        <SelectInput
          control={control}
          rules={{ required: true }}
          name="ruleProfile.billingCycle"
          label="Billing Cycle"
          options={BillingCycleChoices}
          placeholder="Billing Cycle"
          description="Specify the frequency of which the customer will be billed."
        />
      </FormControl>
      <FormControl>
        <SelectInput
          name="ruleProfile.docClassIds"
          control={control}
          isMulti
          rules={{ required: true }}
          label="Required Documents"
          options={selectDocumentClassData}
          isFetchError={isDocumentClassError}
          isLoading={isDocumentClassLoading}
          placeholder="Select Required Document Class."
          description="Specify the document classes that are required for this customer."
          hasPopoutWindow
          popoutLink="/billing/document-classes/"
          popoutLinkLabel="Document Classification"
        />
      </FormControl>
    </FormGroup>
  );
}
