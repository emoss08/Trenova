/* eslint-disable react-hooks/exhaustive-deps */
import { Button, FormSaveButton } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { formatToUserTimezone } from "@/lib/date";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { useUser } from "@/stores/user-store";
import { type EditTableSheetProps } from "@/types/data-table";
import { type API_ENDPOINTS } from "@/types/server";
import {
  faChevronDown,
  faChevronUp,
  faX,
} from "@fortawesome/pro-solid-svg-icons";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import React, { useCallback, useEffect, useRef } from "react";
import {
  FormProvider,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";
import { useDataTable } from "../data-table/data-table-provider";
import { Kbd } from "../kbd";
import { ComponentLoader } from "./component-loader";
import { Form } from "./form";
import { Icon } from "./icons";
import { Separator } from "./separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

type FormEditModalProps<T extends FieldValues> = EditTableSheetProps<T> & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  form: UseFormReturn<T>;
  className?: string;
  fieldKey?: keyof T;
};

export function FormEditModal<T extends FieldValues>({
  currentRecord,
  url,
  title,
  formComponent,
  queryKey,
  fieldKey,
  form,
  className,
}: FormEditModalProps<T>) {
  const { table, rowSelection, isLoading } = useDataTable();
  const { isPopout, closePopout } = usePopoutWindow();
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [_, setSearchParams] = useQueryStates(searchParamsParser);
  const queryClient = useQueryClient();
  const user = useUser();

  const previousRecordIdRef = useRef<string | number | null>(null);
  const selectedRowKey = Object.keys(rowSelection)[0];

  const selectedRow = React.useMemo(() => {
    if (isLoading && !selectedRowKey) return;
    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [selectedRowKey, isLoading]);

  const index = table
    .getCoreRowModel()
    .flatRows.findIndex((row) => row.id === selectedRow?.id);

  const nextId = React.useMemo(
    () => table.getCoreRowModel().flatRows[index + 1]?.id,
    [index, isLoading],
  );

  const prevId = React.useMemo(
    () => table.getCoreRowModel().flatRows[index - 1]?.id,
    [index, isLoading],
  );

  const onPrev = React.useCallback(() => {
    if (prevId) table.setRowSelection({ [prevId]: true });
  }, [prevId, isLoading]);

  const onNext = React.useCallback(() => {
    if (nextId) table.setRowSelection({ [nextId]: true });
  }, [nextId, isLoading, table]);

  React.useEffect(() => {
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

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  // Update form values when currentRecord changes and is not loading
  useEffect(() => {
    if (
      !isLoading &&
      currentRecord &&
      currentRecord.id !== previousRecordIdRef.current
    ) {
      reset(currentRecord);
      previousRecordIdRef.current = currentRecord.id;
    }
  }, [currentRecord, isLoading, reset]);

  const handleClose = useCallback(() => {
    reset();
  }, [reset]);

  const { mutateAsync } = useApiMutation<
    T, // The response data type
    T, // The variables type
    unknown, // The context type
    T // The form values type for error handling
  >({
    mutationFn: async (values: T) => {
      const response = await http.put<T>(`${url}${currentRecord?.id}`, values);
      return response.data;
    },
    onMutate: async (newValues) => {
      // * Cancel any outgoing refetches so they don't overwrite our optmistic update
      await queryClient.cancelQueries({
        queryKey: [queryKey],
      });

      // * Snapshot the previous value
      const previousRecord = queryClient.getQueryData([queryKey]);

      // * Optimistically update to the new value
      queryClient.setQueryData([queryKey], newValues);

      return { previousRecord, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Changes have been saved", {
        description: `${title} updated successfully`,
      });

      // * Invalidate the query
      broadcastQueryInvalidation({
        queryKey: [queryKey],
        options: {
          correlationId: `update-${queryKey}-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      // * Reset the form to the new values
      reset(newValues);

      // * Reset row seleciton
      table.resetRowSelection();

      // * Close the modal
      setSearchParams({ modalType: null, entityId: null });

      // * If the page is a popout, close it
      if (isPopout) {
        closePopout();
      }
    },
    setFormError: setError,
    resourceName: title,
  });

  const onSubmit = useCallback(
    async (values: T) => {
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

  const dialogContent = (
    <DialogContent withClose={false} className={cn("max-w-[450px]", className)}>
      <DialogHeader>
        <div className="flex items-center justify-between gap-2">
          <div className="flex flex-col">
            <DialogTitle>
              <div>
                {isLoading
                  ? "Loading record..."
                  : fieldKey && currentRecord
                    ? currentRecord[fieldKey]
                    : title}
              </div>
            </DialogTitle>
            {!isLoading && currentRecord && (
              <DialogDescription>
                Last updated on{" "}
                {formatToUserTimezone(currentRecord.updatedAt, {
                  timezone: user?.timezone,
                  timeFormat: user?.timeFormat,
                })}
              </DialogDescription>
            )}
          </div>
          <div className="flex h-7 items-center gap-1">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7"
                    disabled={!prevId}
                    onClick={onPrev}
                  >
                    <Icon icon={faChevronUp} className="size-5" />
                    <span className="sr-only">Previous</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  <p>
                    Navigate <Kbd variant="outline">↑</Kbd>
                  </p>
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7"
                    disabled={!nextId}
                    onClick={onNext}
                  >
                    <Icon icon={faChevronDown} className="h-5 w-5" />
                    <span className="sr-only">Next</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  <p>
                    Navigate <Kbd variant="outline">↓</Kbd>
                  </p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <Separator orientation="vertical" className="mx-1" />
            <DialogClose autoFocus={true} asChild>
              <Button size="icon" variant="ghost" className="h-7 w-7">
                <Icon icon={faX} className="size-5" />
                <span className="sr-only">Close</span>
              </Button>
            </DialogClose>
          </div>
        </div>
      </DialogHeader>
      <FormProvider {...form}>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <DialogBody>
            {isLoading ? (
              <ComponentLoader message={`Loading ${title}...`} />
            ) : (
              formComponent
            )}
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <FormSaveButton
              isPopout={isPopout}
              isSubmitting={isSubmitting}
              title={title}
            />
          </DialogFooter>
        </Form>
      </FormProvider>
    </DialogContent>
  );

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
      {dialogContent}
    </Dialog>
  );
}
