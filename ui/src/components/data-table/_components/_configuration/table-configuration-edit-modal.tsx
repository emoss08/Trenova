/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button, FormSaveButton } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import {
  tableConfigurationSchema,
  type TableConfigurationSchema,
} from "@/lib/schemas/table-configuration-schema";
import { api } from "@/services/api";
import { useUser } from "@/stores/user-store";
import type { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { TableConfigurationForm } from "./table-configuration-form";

export function TableConfigurationEditModal({
  open,
  onOpenChange,
  config,
}: TableSheetProps & { config: TableConfigurationSchema }) {
  const { isPopout, closePopout } = usePopoutWindow();
  const queryClient = useQueryClient();
  const user = useUser();

  const form = useForm({
    resolver: zodResolver(tableConfigurationSchema),
    defaultValues: config,
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (data: TableConfigurationSchema) => {
      if (!config.id) {
        throw new Error("Table configuration ID is required");
      }

      return await api.tableConfigurations.update(config.id, data);
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["table-configurations", config.resource],
      });

      const previousData = queryClient.getQueryData<TableConfigurationSchema>([
        "table-configurations",
        config.resource,
      ]);

      queryClient.setQueryData(
        [queries.tableConfiguration.getDefaultOrLatestConfiguration._def],
        newValues,
      );

      return { previousData, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Table configuration updated");

      broadcastQueryInvalidation({
        queryKey: ["table-configurations"],
        options: {
          correlationId: `update-table-configuration-${config.resource}-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      reset(newValues);

      onOpenChange(false);

      if (isPopout) {
        closePopout();
      }
    },
    setFormError: setError,
    resourceName: "Table Configuration",
  });

  const onSubmit = useCallback(
    async (values: TableConfigurationSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isSubmitting, handleSubmit, onSubmit]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Table Configuration</DialogTitle>
          <DialogDescription>
            Last updated on{" "}
            {formatToUserTimezone(config.updatedAt ?? 0, {
              timeFormat: user?.timeFormat,
            })}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <DialogBody>
              <TableConfigurationForm />
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={isSubmitting}
                title="Table Configuration"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
