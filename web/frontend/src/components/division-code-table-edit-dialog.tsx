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
import { useGLAccounts } from "@/hooks/useQueries";
import { formatToUserTimezone } from "@/lib/date";
import { divisionCodeSchema } from "@/lib/validations/AccountingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  DivisionCode,
  DivisionCodeFormValues as FormValues,
} from "@/types/accounting";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { DCForm } from "./division-code-table-dialog";
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

export function DCEditForm({
  divisionCode,
  open,
}: {
  divisionCode: DivisionCode;
  open: boolean;
}) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(divisionCodeSchema),
    defaultValues: divisionCode,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/division-codes/${divisionCode.id}/`,
    successMessage: "Division Code updated successfully.",
    queryKeysToInvalidate: "divisionCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to update division code.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);
  const { selectGLAccounts, isLoading, isError } = useGLAccounts(open);

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <DCForm
          control={control}
          glAccounts={selectGLAccounts}
          isError={isError}
          isLoading={isLoading}
        />
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

export function DivisionCodeEditDialog({
  open,
  onOpenChange,
}: TableSheetProps) {
  const [divisionCode] = useTableStore.use("currentRecord") as DivisionCode[];

  if (!divisionCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{divisionCode.code}</span>
            <Badge className="ml-5" variant="purple">
              {divisionCode.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(divisionCode.updatedAt)}
        </CredenzaDescription>
        <DCEditForm divisionCode={divisionCode} open={open} />
      </CredenzaContent>
    </Credenza>
  );
}
