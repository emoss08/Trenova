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

import { CommodityForm } from "@/components/commodity-dialog";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { commoditySchema } from "@/lib/validations/CommoditiesSchema";
import { useTableStore } from "@/stores/TableStore";
import type { Commodity, CommodityFormValues } from "@/types/commodities";
import { yupResolver } from "@hookform/resolvers/yup";
import { useEffect } from "react";
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

function CommodityEditForm({ commodity }: { commodity: Commodity }) {
  const { control, handleSubmit, watch, reset, setValue } =
    useForm<CommodityFormValues>({
      resolver: yupResolver(commoditySchema),
      defaultValues: commodity,
    });

  const mutation = useCustomMutation<CommodityFormValues>(control, {
    method: "PUT",
    path: `/commodities/${commodity.id}/`,
    successMessage: "Commodity updated successfully.",
    queryKeysToInvalidate: "commodities",
    closeModal: true,
    reset,
    errorMessage: "Failed to update commodity.",
  });

  const onSubmit = (values: CommodityFormValues) => mutation.mutate(values);

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name === "hazardousMaterialId" && value.hazardousMaterialId) {
        setValue("isHazmat", true);
      } else if (name === "hazardousMaterialId" && !value.hazardousMaterialId) {
        setValue("isHazmat", false);
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, setValue]);

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <CommodityForm control={control} />
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

export function CommodityEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [commodity] = useTableStore.use("currentRecord") as Commodity[];

  if (!commodity) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{commodity.name}</span>
            <Badge className="ml-5" variant="purple">
              {commodity.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {formatToUserTimezone(commodity.updatedAt)}
        </CredenzaDescription>
        <CommodityEditForm commodity={commodity} />
      </CredenzaContent>
    </Credenza>
  );
}
