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



import { ACForm } from "@/components/accessorial-charge-table-dialog";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { accessorialChargeSchema } from "@/lib/validations/BillingSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
    AccessorialCharge,
    AccessorialChargeFormValues as FormValues,
} from "@/types/billing";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
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

function ACEditForm({
  accessorialCharge,
}: {
  accessorialCharge: AccessorialCharge;
}) {
  const { control, handleSubmit, reset } = useForm<FormValues>({
    resolver: yupResolver(accessorialChargeSchema),
    defaultValues: accessorialCharge,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/accessorial-charges/${accessorialCharge.id}/`,
    successMessage: "Accesorial Charge updated successfully.",
    queryKeysToInvalidate: "accessorialCharges",
    closeModal: true,
    reset,
    errorMessage: "Failed to update accesorial charge.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <ACForm control={control} />
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

export function AccessorialChargeTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [accessorialCharge] = useTableStore.use(
    "currentRecord",
  ) as AccessorialCharge[];

  if (!accessorialCharge) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{accessorialCharge.code}</span>
            <Badge className="ml-5" variant="purple">
              {accessorialCharge.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(accessorialCharge.updatedAt)}
        </CredenzaDescription>
        <ACEditForm accessorialCharge={accessorialCharge} />
      </CredenzaContent>
    </Credenza>
  );
}
