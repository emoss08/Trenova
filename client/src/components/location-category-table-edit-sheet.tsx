/*
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { locationCategorySchema as formSchema } from "@/lib/validations/LocationSchema";
import { useTableStore } from "@/stores/TableStore";
import {
  LocationCategoryFormValues as FormValues,
  LocationCategory,
} from "@/types/location";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { LCForm } from "./location-category-table-sheet";

export function LCEditForm({
  locationCategory,
}: {
  locationCategory: LocationCategory;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(formSchema),
    defaultValues: {
      name: locationCategory.name,
      description: locationCategory.description,
      color: locationCategory.color,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/location-categories/${locationCategory.id}/`,
      successMessage: "Location Category updated successfully.",
      queryKeysToInvalidate: ["location-categories-table-data"],
      additionalInvalidateQueries: ["locationCategories"],
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
      <LCForm control={control} />
      <DialogFooter className="mt-6">
        <Button type="submit" isLoading={isSubmitting}>
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
  const [locationCategory] = useTableStore.use(
    "currentRecord",
  ) as LocationCategory[];

  if (!locationCategory) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{locationCategory && locationCategory.name}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {locationCategory && formatDate(locationCategory.updatedAt)}
        </DialogDescription>
        {locationCategory && <LCEditForm locationCategory={locationCategory} />}
      </DialogContent>
    </Dialog>
  );
}
