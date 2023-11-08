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
import {
  LocationCategory,
  LocationCategoryFormValues as FormValues,
} from "@/types/location";
import { useTableStore } from "@/stores/TableStore";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { formatDate } from "@/lib/date";
import React from "react";
import { useForm } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { toast } from "@/components/ui/use-toast";
import { Button } from "@/components/ui/button";
import { LCForm } from "@/components/location-categories/lc-table-sheet";
import { locationCategorySchema as formSchema } from "@/lib/validations/location";

export function LCEditForm({
  locationCategory,
}: {
  locationCategory: LocationCategory;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit, watch, setValue } = useForm<FormValues>(
    {
      resolver: yupResolver(formSchema),
      defaultValues: {
        name: locationCategory.name,
        description: locationCategory.description,
        color: locationCategory.color,
      },
    },
  );

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "PUT",
      path: `/location_categories/${locationCategory.id}/`,
      successMessage: "Location Category updated successfully.",
      queryKeysToInvalidate: ["location-categories-table-data"],
      closeModal: true,
      errorMessage: "Failed to update location category.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <LCForm control={control} setValue={setValue} watch={watch} />
      <DialogFooter className="mt-6">
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
        >
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function LCTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [locationCategory] = useTableStore.use("currentRecord");

  if (!locationCategory) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{locationCategory && locationCategory.name}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {locationCategory && formatDate(locationCategory.modified)}
        </DialogDescription>
        {locationCategory && <LCEditForm locationCategory={locationCategory} />}
      </DialogContent>
    </Dialog>
  );
}
