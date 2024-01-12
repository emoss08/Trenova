/*
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

import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl } from "@/components/ui/form";
import { useWorkers } from "@/hooks/useQueries";
import { cleanObject } from "@/lib/utils";
import { useShipmentStore } from "@/stores/ShipmentStore";
import { yupResolver } from "@hookform/resolvers/yup";
import { useState } from "react";
import { useForm } from "react-hook-form";
import * as Yup from "yup";

type SendMessageDialogProps = {
  onOpenChange: (open: boolean) => void;
  open: boolean;
};

type FormValues = {
  worker: string;
  message: string;
};

const schema: Yup.ObjectSchema<FormValues> = Yup.object().shape({
  worker: Yup.string().required("Please select a worker."),
  message: Yup.string().required("Please enter a message."),
});

export function SendMessageDialog({
  onOpenChange,
  open,
}: SendMessageDialogProps) {
  const [isSending, setIsSending] = useState(false);
  const [currentWorker] = useShipmentStore.use("currentWorker");

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(schema),
    defaultValues: {
      worker: currentWorker?.id || "",
      message: "",
    },
  });

  const {
    selectWorkers,
    isLoading: isWorkersLoading,
    isError: isWorkersError,
  } = useWorkers();

  const onSubmit = (values: FormValues) => {
    const cleanedValues = cleanObject(values);

    setIsSending(true);
    console.log(cleanedValues);
    // mutation.mutate(cleanedValues);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[500px]">
        <DialogHeader>
          <DialogTitle>Send Message</DialogTitle>
          <DialogDescription>
            Send a message to the worker to request more information.
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={handleSubmit(onSubmit)}
          className="flex h-full flex-col overflow-y-auto"
        >
          <Form>
            <FormControl>
              <SelectInput
                control={control}
                options={selectWorkers}
                isLoading={isWorkersLoading}
                isFetchError={isWorkersError}
                name="worker"
                placeholder="Select a worker"
                rules={{ required: true }}
                label="Worker"
                description="Select a worker to send a message to."
              />
            </FormControl>
            <FormControl className="mt-4">
              <TextareaField
                rules={{ required: true }}
                control={control}
                name="message"
                label="Message"
                description="Enter a message to send to the worker."
              />
            </FormControl>
          </Form>
          <DialogFooter>
            <Button
              type="submit"
              isLoading={isSending}
              loadingText="Send Message..."
            >
              Send Message
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
