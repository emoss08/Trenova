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
import { useTableStore } from "@/stores/TableStore";
import { type TableSheetProps } from "@/types/tables";
import type { WorkerFormValues as FormValues, Worker } from "@/types/worker";
import React from "react";
import { useForm } from "react-hook-form";

function WorkerEditForm({
  worker,
  onOpenChange,
}: {
  worker: Worker;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    // resolver: yupResolver(trailerSchema),
    defaultValues: worker,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/worker/${worker.id}/`,
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
      {/* <TrailerForm control={control} open={open} /> */}
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

export function WorkerEditDialog({ onOpenChange, open }: TableSheetProps) {
  const [worker] = useTableStore.use("currentRecord") as Worker[];

  if (!worker) {
    return null;
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{worker && worker.code}</SheetTitle>
          <SheetDescription>
            Last updated on {worker && formatToUserTimezone(worker.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        {worker && (
          <WorkerEditForm
            worker={worker}
            open={open}
            onOpenChange={onOpenChange}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
