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
