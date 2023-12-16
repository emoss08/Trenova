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

import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { cn } from "@/lib/utils";
import { tractorSchema } from "@/lib/validations/EquipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import { TractorFormValues as FormValues, Tractor } from "@/types/equipment";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Button } from "../ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "../ui/sheet";
import { TractorForm } from "./tractor-table-dialog";

export function TractorEditForm({
  tractor,
  open,
  onOpenChange,
}: {
  tractor: Tractor;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  if (!tractor) return null;

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(tractorSchema),
    defaultValues: {
      status: tractor.status,
      code: tractor.code,
      equipmentType: tractor.equipmentType,
      manufacturer: tractor?.manufacturer,
      vinNumber: tractor?.vinNumber,
      model: tractor?.model,
      year: tractor?.year,
      state: tractor?.state,
      fleetCode: tractor?.fleetCode,
      primaryWorker: tractor?.primaryWorker,
      secondaryWorker: tractor?.secondaryWorker,
      licensePlateNumber: tractor?.licensePlateNumber,
      leasedDate: tractor?.leasedDate,
      hosExempt: tractor.hosExempt,
      ownerOperated: tractor.ownerOperated,
      leased: tractor.leased,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/tractors/${tractor.id}/`,
      successMessage: "Tractor updated successfully.",
      queryKeysToInvalidate: ["tractor-table-data"],
      additionalInvalidateQueries: ["tractors"],
      closeModal: true,
      errorMessage: "Failed to update existing tractor.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  function onSubmit(values: FormValues) {
    setIsSubmitting(true);
    mutation.mutate(values);
  }

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col h-full overflow-y-auto"
    >
      <TractorForm control={control} open={open} />
      <SheetFooter className="mb-12">
        <Button
          type="reset"
          variant="secondary"
          onClick={() => onOpenChange(false)}
          className="w-full"
        >
          Cancel
        </Button>
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
          className="w-full"
        >
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
          <SheetTitle>{tractor && tractor.code}</SheetTitle>
          <SheetDescription>
            Last updated on {tractor && formatDate(tractor.modified)}
          </SheetDescription>
        </SheetHeader>
        {tractor && (
          <TractorEditForm
            tractor={tractor}
            open={open}
            onOpenChange={onOpenChange}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
