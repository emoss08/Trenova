import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { LocationCategorySchema as formSchema } from "@/lib/validations/LocationSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  LocationCategoryFormValues as FormValues,
  LocationCategory,
} from "@/types/location";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { LCForm } from "./location-category-table-dialog";
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

export function LCEditForm({
  locationCategory,
}: {
  locationCategory: LocationCategory;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(formSchema),
    defaultValues: locationCategory,
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
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <LCForm control={control} />
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

export function LocationCategoryEditDialog({
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
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>
            {locationCategory && locationCategory.name}
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {locationCategory && formatDate(locationCategory.updatedAt)}
        </CredenzaDescription>
        {locationCategory && <LCEditForm locationCategory={locationCategory} />}
      </CredenzaContent>
    </Credenza>
  );
}
