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

import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useGLAccounts } from "@/hooks/useQueries";
import { formatToUserTimezone } from "@/lib/date";
import { revenueCodeSchema } from "@/lib/validations/AccountingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  RevenueCodeFormValues as FormValues,
  RevenueCode,
} from "@/types/accounting";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { RCForm } from "./revenue-code-table-dialog";
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

function RCEditForm({
  revenueCode,
  open,
}: {
  revenueCode: RevenueCode;
  open: boolean;
}) {
  const { selectGLAccounts, isLoading, isError } = useGLAccounts(open);

  const { handleSubmit, reset, control } = useForm<FormValues>({
    resolver: yupResolver(revenueCodeSchema),
    defaultValues: revenueCode,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/revenue-codes/${revenueCode.id}/`,
    successMessage: "Revenue Code updated successfully.",
    queryKeysToInvalidate: "revenueCodes",
    closeModal: true,
    reset,
    errorMessage: "Failed to update revenue code.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <RCForm
          control={control}
          glAccounts={selectGLAccounts}
          isLoading={isLoading}
          isError={isError}
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

export function RevenueCodeTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [revenueCode] = useTableStore.use("currentRecord") as RevenueCode[];

  if (!revenueCode) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{revenueCode.code}</span>
            <Badge className="ml-5" variant="purple">
              {revenueCode.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(revenueCode.updatedAt)}
        </CredenzaDescription>
        <RCEditForm revenueCode={revenueCode} open={open} />
      </CredenzaContent>
    </Credenza>
  );
}
