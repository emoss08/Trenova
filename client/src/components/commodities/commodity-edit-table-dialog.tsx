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

import { CommodityForm } from "@/components/commodities/commodity-dialog";
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
import { useHazardousMaterial } from "@/hooks/useQueries";
import { formatDate } from "@/lib/date";
import { commoditySchema } from "@/lib/validations/CommoditiesSchema";
import { useTableStore } from "@/stores/TableStore";
import { Commodity, CommodityFormValues } from "@/types/commodities";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";

function CommodityEditForm({
  commodity,
  open,
}: {
  commodity: Commodity;
  open: boolean;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { selectHazardousMaterials, isLoading, isError } =
    useHazardousMaterial(open);

  const { control, reset, handleSubmit, watch, setValue } =
    useForm<CommodityFormValues>({
      resolver: yupResolver(commoditySchema),
      defaultValues: {
        status: commodity.status,
        name: commodity.name,
        description: commodity?.description || "",
        minTemp: commodity?.minTemp || "",
        maxTemp: commodity?.maxTemp || "",
        setPointTemp: commodity?.setPointTemp,
        unitOfMeasure: commodity?.unitOfMeasure,
        hazardousMaterial: commodity.hazardousMaterial || "",
        isHazmat: commodity?.isHazmat || "",
      },
    });

  const mutation = useCustomMutation<CommodityFormValues>(
    control,
    {
      method: "PUT",
      path: `/commodities/${commodity.id}/`,
      successMessage: "Commodity updated successfully.",
      queryKeysToInvalidate: ["commodity-table-data"],
      closeModal: true,
      errorMessage: "Failed to update commodity.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: CommodityFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  React.useEffect(() => {
    const hazardousMaterial = watch("hazardousMaterial");

    if (hazardousMaterial) {
      setValue("isHazmat", "Y");
    } else {
      setValue("isHazmat", "N");
    }
  }, [watch("hazardousMaterial"), setValue]);

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <CommodityForm
        control={control}
        hazardousMaterials={selectHazardousMaterials}
        isLoading={isLoading}
        isError={isError}
      />
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

export function CommodityEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [commodity] = useTableStore.use("currentRecord");

  if (!commodity) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{commodity && commodity.name}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {commodity && formatDate(commodity.modified)}
        </DialogDescription>
        {commodity && <CommodityEditForm commodity={commodity} open={open} />}
      </DialogContent>
    </Dialog>
  );
}
