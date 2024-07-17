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



import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { tractorSchema } from "@/lib/validations/EquipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  TractorFormValues as FormValues,
  Tractor,
} from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-table-dialog";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "./ui/sheet";

export function TractorEditForm({
  tractor,
  onOpenChange,
}: {
  tractor: Tractor;
  onOpenChange: (open: boolean) => void;
}) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(tractorSchema),
    defaultValues: tractor,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/tractors/${tractor.id}/`,
    successMessage: "Tractor updated successfully.",
    queryKeysToInvalidate: "tractors",
    closeModal: true,
    reset,
    errorMessage: "Failed to update existing tractor.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex h-full flex-col overflow-y-auto"
    >
      <TractorForm control={control} />
      <SheetFooter className="mb-12">
        <Button
          type="reset"
          variant="secondary"
          onClick={() => onOpenChange(false)}
          className="w-full"
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={mutation.isPending} className="w-full">
          Save
        </Button>
      </SheetFooter>
    </form>
  );
}

export function TractorTableEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [tractor] = useTableStore.use("currentRecord") as Tractor[];

  if (!tractor) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle className="flex">
            <span>{tractor.code}</span>
            <Badge className="ml-5" variant="purple">
              {tractor.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(tractor.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <TractorEditForm tractor={tractor} onOpenChange={onOpenChange} />
      </SheetContent>
    </Sheet>
  );
}
