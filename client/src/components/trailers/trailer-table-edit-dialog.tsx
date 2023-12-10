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

import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { cleanObject, cn } from "@/lib/utils";
import { useTableStore } from "@/stores/TableStore";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { Trailer, TrailerFormValues as FormValues } from "@/types/equipment";
import { TrailerForm } from "@/components/trailers/trailer-table-dialog";
import { trailerSchema } from "@/lib/validations/EquipmentSchema";

function TrailerEditForm({
  trailer,
  open,
  onOpenChange,
}: {
  trailer: Trailer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(trailerSchema),
    defaultValues: {
      code: trailer.code,
      status: trailer.status,
      equipmentType: trailer.equipmentType,
      make: trailer.make,
      model: trailer.model,
      year: trailer.year,
      vinNumber: trailer.vinNumber,
      fleetCode: trailer.fleetCode,
      licensePlateNumber: trailer.licensePlateNumber,
      lastInspection: trailer.lastInspection,
      state: trailer.state,
      isLeased: trailer.isLeased,
      registrationNumber: trailer.registrationNumber,
      registrationState: trailer.registrationState,
      registrationExpiration: trailer.registrationExpiration,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/trailers/${trailer.id}/`,
      successMessage: "Trailer updated successfully.",
      queryKeysToInvalidate: ["trailer-table-data"],
      additionalInvalidateQueries: ["trailers"],
      closeModal: true,
      errorMessage: "Failed to update trailers.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: FormValues) => {
    const cleanedValues = cleanObject(values);

    setIsSubmitting(true);
    mutation.mutate(cleanedValues);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col h-full overflow-y-auto"
    >
      <TrailerForm control={control} open={open} />
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
          Save Changes
        </Button>
      </SheetFooter>
    </form>
  );
}

export function TrailerEditDialog({ onOpenChange, open }: TableSheetProps) {
  const [trailer] = useTableStore.use("currentRecord") as Trailer[];

  if (!trailer) {
    return null;
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{trailer && trailer.code}</SheetTitle>
          <SheetDescription>
            Last updated on {trailer && formatDate(trailer.modified)}
          </SheetDescription>
        </SheetHeader>
        {trailer && (
          <TrailerEditForm
            trailer={trailer}
            open={open}
            onOpenChange={onOpenChange}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
