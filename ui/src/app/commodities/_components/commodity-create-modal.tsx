/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import { commoditySchema } from "@/lib/schemas/commodity-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { CommodityForm } from "./commodity-form";

export function CreateCommodityModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(commoditySchema),
    defaultValues: {
      status: Status.Active,
      name: "",
      description: "",
      minTemperature: undefined,
      maxTemperature: undefined,
      weightPerUnit: undefined,
      freightClass: "",
      dotClassification: "",
      stackable: false,
      fragile: false,
      hazardousMaterialId: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Commodity"
      formComponent={<CommodityForm />}
      form={form}
      url="/commodities/"
      queryKey="commodity-list"
      className="max-w-[500px]"
    />
  );
}
