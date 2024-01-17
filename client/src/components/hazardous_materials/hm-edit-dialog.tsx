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
import { hazardousMaterialSchema } from "@/lib/validations/CommoditiesSchema";
import { useTableStore } from "@/stores/TableStore";
import {
  HazardousMaterialFormValues as FormValues,
  HazardousMaterial,
  HazardousMaterialFormValues,
} from "@/types/commodities";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hm-dialog";

function HazardousMaterialEditForm({
  hazardousMaterial,
}: {
  hazardousMaterial: HazardousMaterial;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(hazardousMaterialSchema),
    defaultValues: hazardousMaterial,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/hazardous_materials/${hazardousMaterial.id}/`,
      successMessage: "Hazardous Material updated successfully.",
      queryKeysToInvalidate: ["hazardous-material-table-data"],
      closeModal: true,
      errorMessage: "Failed to update Hazardous Material.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: HazardousMaterialFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <HazardousMaterialForm control={control} />
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

export function HazardousMaterialEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [hazardousMaterial] = useTableStore.use(
    "currentRecord",
  ) as HazardousMaterial[];

  if (!hazardousMaterial) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[600px]">
        <DialogHeader>
          <DialogTitle>
            {hazardousMaterial && hazardousMaterial.name}
          </DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp
          {hazardousMaterial && formatDate(hazardousMaterial.modified)}
        </DialogDescription>
        {hazardousMaterial && (
          <HazardousMaterialEditForm hazardousMaterial={hazardousMaterial} />
        )}
      </DialogContent>
    </Dialog>
  );
}
