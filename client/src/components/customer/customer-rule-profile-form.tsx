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
import { useDocumentClass } from "@/hooks/useQueries";
import { CustomerFormValues as FormValues } from "@/types/customer";
import { Control } from "react-hook-form";
import { SelectInput } from "../common/fields/select-input";

export function CustomerRuleProfileForm({
  control,
  open,
}: {
  control: Control<FormValues>;
  open: boolean;
}) {
  const {
    selectDocumentClassData,
    isError: isDocumentClassError,
    isLoading: isDocumentClassLoading,
  } = useDocumentClass(open);

  return (
    <>
      <div className="my-4 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              rules={{ required: true }}
              name="ruleProfile.name"
              label="Name"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Name"
              description="Specify the official name of the customer."
              maxLength={50}
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <SelectInput
              name="ruleProfile.documentClass"
              control={control}
              isMulti
              rules={{ required: true }}
              label="Document Classification"
              options={selectDocumentClassData}
              isFetchError={isDocumentClassError}
              isLoading={isDocumentClassLoading}
              placeholder="Select Document Classification"
              description="Select the state or region for the customer."
              hasPopoutWindow
              popoutLink="#" // TODO: Change once Document Classification is added.
              popoutLinkLabel="Document Classification"
            />
          </div>
        </div>
      </div>
    </>
  );
}
