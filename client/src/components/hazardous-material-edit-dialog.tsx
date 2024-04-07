import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { hazardousMaterialSchema } from "@/lib/validations/CommoditiesSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  HazardousMaterialFormValues as FormValues,
  HazardousMaterial,
} from "@/types/commodities";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hazardous-material-dialog";
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
      path: `/hazardous-materials/${hazardousMaterial.id}/`,
      successMessage: "Hazardous Material updated successfully.",
      queryKeysToInvalidate: ["hazardous-material-table-data"],
      closeModal: true,
      errorMessage: "Failed to update Hazardous Material.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <HazardousMaterialForm control={control} />
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
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>
            {hazardousMaterial && hazardousMaterial.name}
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {hazardousMaterial && formatDate(hazardousMaterial.updatedAt)}
        </CredenzaDescription>
        {hazardousMaterial && (
          <HazardousMaterialEditForm hazardousMaterial={hazardousMaterial} />
        )}
      </CredenzaContent>
    </Credenza>
  );
}
