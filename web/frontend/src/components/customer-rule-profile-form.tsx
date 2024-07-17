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
