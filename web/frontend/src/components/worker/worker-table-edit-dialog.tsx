/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

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
