import { CommodityForm } from "@/components/commodity-dialog";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { commoditySchema } from "@/lib/validations/CommoditiesSchema";
import { useTableStore } from "@/stores/TableStore";
import type { Commodity, CommodityFormValues } from "@/types/commodities";
import { yupResolver } from "@hookform/resolvers/yup";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
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

function CommodityEditForm({
  commodity,
  open,
}: {
  commodity: Commodity;
  open: boolean;
}) {
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const { control, reset, handleSubmit, watch, setValue } =
    useForm<CommodityFormValues>({
      resolver: yupResolver(commoditySchema),
      defaultValues: commodity,
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
        <CommodityForm control={control} open={open} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={isSubmitting}>
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
          <CredenzaTitle>{commodity && commodity.name} </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {commodity && formatDate(commodity.createdAt)}
        </CredenzaDescription>
        {commodity && <CommodityEditForm commodity={commodity} open={open} />}
      </CredenzaContent>
    </Credenza>
  );
}
