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
import { formatToUserTimezone } from "@/lib/date";
import { cleanObject, cn } from "@/lib/utils";
import { trailerSchema } from "@/lib/validations/EquipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  TrailerFormValues as FormValues,
  Trailer,
} from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { TrailerForm } from "./trailer-table-dialog";

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
    defaultValues: trailer,
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
      className="flex h-full flex-col overflow-y-auto"
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
        <Button type="submit" isLoading={isSubmitting} className="w-full">
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
            Last updated on {trailer && formatToUserTimezone(trailer.updatedAt)}
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
