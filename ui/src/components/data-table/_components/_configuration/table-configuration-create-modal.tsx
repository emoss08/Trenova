import { Button, FormSaveButton } from "@/components/ui/button";
import {
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
import {
  tableConfigurationSchema,
  type FilterStateSchema,
  type TableConfigurationSchema,
} from "@/lib/schemas/table-configuration-schema";
import { api } from "@/services/api";
import type { Resource } from "@/types/audit-entry";
import { TableSheetProps } from "@/types/data-table";
import { Visibility } from "@/types/table-configuration";
import { zodResolver } from "@hookform/resolvers/zod";
import { Dialog } from "@radix-ui/react-dialog";
import { useQueryClient } from "@tanstack/react-query";
import type { VisibilityState } from "@tanstack/react-table";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { TableConfigurationForm } from "./table-configuration-form";

type CreateTableConfigurationModalProps = TableSheetProps & {
  resource: Resource;
  visiblityState: VisibilityState;
  tableFilters: FilterStateSchema;
  columnOrder?: string[];
};

export function CreateTableConfigurationModal({
  open,
  onOpenChange,
  resource,
  visiblityState,
  tableFilters,
  columnOrder,
}: CreateTableConfigurationModalProps) {
  const { isPopout, closePopout } = usePopoutWindow();
  const queryClient = useQueryClient();

  const form = useForm({
    resolver: zodResolver(tableConfigurationSchema),
    defaultValues: {
      name: "",
      description: "",
      visibility: Visibility.Private,
      isDefault: false,
      resource: resource,
      tableConfig: {
        columnVisibility: visiblityState,
        filters: tableFilters.filters,
        sort: tableFilters.sort,
        columnOrder: columnOrder,
      },
    },
  });

  const {
    register,
    setError,
    formState: { isSubmitting, isSubmitSuccessful },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: TableConfigurationSchema) => {
      return await api.tableConfigurations.create(values);
    },
    onSuccess: (data) => {
      toast.success("Table Configuration Created", {
        description: `Table configuration ${data.name} created successfully`,
      });
      handleClose();

      broadcastQueryInvalidation({
        queryKey: ["table-configurations"],
        options: { correlationId: `create-${resource}-${Date.now()}` },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      queryClient.setQueryData(["table-configurations", resource], data);
      reset({
        name: "",
        description: "",
        visibility: Visibility.Private,
        isDefault: false,
        resource: resource,
        tableConfig: {
          columnVisibility: visiblityState,
          filters: tableFilters.filters,
          sort: tableFilters.sort,
          columnOrder: columnOrder,
        },
      });

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
      console.log("Values", values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    register("tableConfig");
    if (visiblityState) {
      form.setValue("tableConfig.columnVisibility", visiblityState, {
        shouldValidate: true,
      });
    }
  }, [register, visiblityState, form]);

  useEffect(() => {
    if (visiblityState) {
      form.setValue("tableConfig.columnVisibility", visiblityState, {
        shouldValidate: true,
        shouldDirty: true,
      });
    }
  }, [visiblityState, form]);

  useEffect(() => {
    if (isSubmitSuccessful || !open) {
      reset({
        name: "",
        description: "",
        visibility: Visibility.Private,
        isDefault: false,
        resource: resource,
        tableConfig: {
          columnVisibility: visiblityState,
          filters: tableFilters.filters,
          sort: tableFilters.sort,
          columnOrder: columnOrder,
        },
      });
    }
  }, [isSubmitSuccessful, reset, open, resource, visiblityState, tableFilters, columnOrder]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create Table Configuration</DialogTitle>
          <DialogDescription>
            Create a new table configuration to customize the columns and
            visibility of the table.
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
