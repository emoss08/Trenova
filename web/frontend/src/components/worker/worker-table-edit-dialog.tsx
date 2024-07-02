import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { cleanObject } from "@/lib/utils";
import { useTableStore } from "@/stores/TableStore";
import { type TableSheetProps } from "@/types/tables";
import type { WorkerFormValues as FormValues, Worker } from "@/types/worker";
import { FormProvider, useForm } from "react-hook-form";
import { Dialog, DialogContent } from "../ui/dialog";
import { WorkerForm } from "./worker-table-dialog";

function WorkerEditForm({ worker }: { worker: Worker; open: boolean }) {
  const methods = useForm<FormValues>({
    // resolver: yupResolver(trailerSchema),
    defaultValues: worker,
  });

  const { control, handleSubmit, reset } = methods;

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/workers/${worker.id}/`,
    successMessage: "Worker updated successfully.",
    queryKeysToInvalidate: "workers",
    closeModal: true,
    reset,
    errorMessage: "Failed to update worker.",
  });

  const onSubmit = (values: FormValues) => {
    const cleanedValues = cleanObject(values);
    mutation.mutate(cleanedValues);
  };

  return (
    <FormProvider {...methods}>
      <form onSubmit={handleSubmit(onSubmit)}>
        <WorkerForm />
        <div className="flex justify-end gap-x-2">
          <Button variant="outline" type="button">
            Cancel
          </Button>
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}

export function WorkerEditDialog({ onOpenChange, open }: TableSheetProps) {
  const [worker] = useTableStore.use("currentRecord") as Worker[];

  if (!worker) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[90em]">
        <WorkerEditForm worker={worker} open={open} />
      </DialogContent>
    </Dialog>
  );
}
