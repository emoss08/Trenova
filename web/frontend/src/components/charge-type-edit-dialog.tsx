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

import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { chargeTypeSchema } from "@/lib/validations/BillingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  ChargeType,
  ChargeTypeFormValues as FormValues,
} from "@/types/billing";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ChargeTypeForm } from "./charge-type-dialog";
import { Badge } from "./ui/badge";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";

function ChargeTypeEditForm({ chargeType }: { chargeType: ChargeType }) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(chargeTypeSchema),
    defaultValues: chargeType,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/charge-types/${chargeType.id}/`,
    successMessage: "Charge Type updated successfully.",
    queryKeysToInvalidate: "chargeTypes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create update charge type.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <ChargeTypeForm control={control} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </CredenzaFooter>
      </form>
    </CredenzaBody>
  );
}

export function ChargeTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [chargeType] = useTableStore.use("currentRecord") as ChargeType[];

  if (!chargeType) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{chargeType.name}</span>
            <Badge className="ml-5" variant="purple">
              {chargeType.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(chargeType.updatedAt)}
        </CredenzaDescription>
        <ChargeTypeEditForm chargeType={chargeType} />
      </CredenzaContent>
    </Credenza>
  );
}
