/* eslint-disable react-hooks/exhaustive-deps */
import { useDataTable } from "@/components/data-table/data-table-provider";
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
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import { workerSchema, WorkerSchema } from "@/lib/schemas/worker-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { WorkerForm } from "./workers-form";

function WorkerEditForm({ currentRecord }: EditTableSheetProps<WorkerSchema>) {
  const { table, isLoading } = useDataTable();
  const queryClient = useQueryClient();
  const [, setSearchParams] = useQueryStates(searchParamsParser);
  const { isPopout, closePopout } = usePopoutWindow();
  const previousRecordIdRef = useRef<string | number | null>(null);

  const form = useForm({
    resolver: zodResolver(workerSchema),
    defaultValues: currentRecord,
    mode: "onChange",
  });

  const {
    setError,
    reset,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: WorkerSchema) => {
      const response = await http.put<WorkerSchema>(
        `/workers/${currentRecord?.id}`,
        values,
      );
      return response.data;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["worker", currentRecord?.id],
      });

      const previousCustomer = queryClient.getQueryData([
        "worker",
        currentRecord?.id,
      ]);

      queryClient.setQueryData(["worker", currentRecord?.id], newValues);

      return { previousCustomer, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Changes have been saved", {
        description: `Worker updated successfully`,
      });

      broadcastQueryInvalidation({
        queryKey: ["worker", "worker-list"],
        options: {
          correlationId: `update-worker-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      reset(newValues);

      table.resetRowSelection();

      setSearchParams({ modalType: null, entityId: null });

      if (isPopout) {
        closePopout();
      }
    },
    resourceName: "Worker",
    setFormError: setError,
  });

  const onSubmit = useCallback(
    async (values: WorkerSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  // Update form values when currentRecord changes and is not loading
  useEffect(() => {
    if (
      !isLoading &&
      currentRecord &&
      currentRecord.id !== previousRecordIdRef.current
    ) {
      reset(currentRecord);
      previousRecordIdRef.current = currentRecord.id ?? null;
    }
  }, [currentRecord, isLoading, reset]);

  // Make sure we populate the form with the current record
  useEffect(() => {
    if (currentRecord) {
      reset(currentRecord);
    }
  }, [currentRecord, reset]);

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

  const onClose = useCallback(() => {
    table.resetRowSelection();
  }, [table]);

  return (
    <FormProvider {...form}>
      <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
        <DialogBody className="p-0">
          <WorkerForm />
        </DialogBody>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <FormSaveButton
            isPopout={isPopout}
            isSubmitting={isSubmitting}
            title="Worker"
          />
        </DialogFooter>
      </Form>
    </FormProvider>
  );
}

export function EditWorkerModal({
  currentRecord,
}: EditTableSheetProps<WorkerSchema>) {
  const { table, rowSelection, isLoading } = useDataTable();

  const selectedRowKey = Object.keys(rowSelection)[0];

  const selectedRow = useMemo(() => {
    if (isLoading && !selectedRowKey) return;
    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [selectedRowKey, isLoading]);

  const index = table
    .getCoreRowModel()
    .flatRows.findIndex((row) => row.id === selectedRow?.id);

  const nextId = useMemo(
    () => table.getCoreRowModel().flatRows[index + 1]?.id,
    [index, isLoading],
  );

  const prevId = useMemo(
    () => table.getCoreRowModel().flatRows[index - 1]?.id,
    [index, isLoading],
  );

  const onPrev = useCallback(() => {
    if (prevId) table.setRowSelection({ [prevId]: true });
  }, [prevId, isLoading]);

  const onNext = useCallback(() => {
    if (nextId) table.setRowSelection({ [nextId]: true });
  }, [nextId, isLoading, table]);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (!selectedRowKey) return;

      // REMINDER: prevent dropdown navigation inside of sheet to change row selection
      const activeElement = document.activeElement;
      const isMenuActive = activeElement?.closest('[role="menu"]');

      if (isMenuActive) return;

      if (e.key === "ArrowUp") {
        e.preventDefault();
        onPrev();
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        onNext();
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [selectedRowKey, onNext, onPrev]);

  return (
    <Dialog
      open={!!selectedRowKey}
      onOpenChange={(open) => {
        if (!open) {
          const el = selectedRowKey
            ? document.getElementById(selectedRowKey)
            : null;
          table.resetRowSelection();

          setTimeout(() => el?.focus(), 0);
        }
      }}
    >
      <DialogContent className="md:max-w-[700px] lg:max-w-[800px]">
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Edit Worker</DialogTitle>
            <DialogDescription>
              Edit the details of the worker.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <WorkerEditForm currentRecord={currentRecord} />
      </DialogContent>
    </Dialog>
  );
}
