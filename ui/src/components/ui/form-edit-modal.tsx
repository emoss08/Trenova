/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

"use no memo";
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
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import React, { useCallback, useEffect, useRef, useTransition } from "react";
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
  titleComponent?: (currentRecord: T) => React.ReactNode;
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
  titleComponent,
}: FormEditModalProps<T>) {
  const { table, rowSelection, isLoading } = useDataTable();
  const { isPopout, closePopout } = usePopoutWindow();
  const [isPending, startTransition] = useTransition();
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser, {
    // Use replace to avoid history stacking and reduce throttling
    history: "replace",
    throttleMs: 50, // Minimum allowed value
  });
  const queryClient = useQueryClient();
  const user = useUser();

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const previousRecordIdRef = useRef<string | number | null>(null);
  const navigationQueueRef = useRef<string | null>(null);
  const isNavigatingRef = useRef(false);
  const selectedRowKey = Object.keys(rowSelection)?.[0];

  const selectedRow = React.useMemo(() => {
    if (isLoading && !selectedRowKey) return;
    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [selectedRowKey, isLoading, table]);

  const index = table
    .getCoreRowModel()
    .flatRows.findIndex((row) => row.id === selectedRow?.id);

  const nextId = React.useMemo(
    () => table.getCoreRowModel().flatRows[index + 1]?.id,
    [index, table],
  );

  const prevId = React.useMemo(
    () => table.getCoreRowModel().flatRows[index - 1]?.id,
    [index, table],
  );

  // Fetch individual record when currentRecord is undefined but entityId exists in URL
  const {
    data: fetchedRecord,
    isLoading: isFetchingRecord,
    error: fetchError,
  } = useQuery<T>({
    queryKey: [queryKey, "single", searchParams.entityId, url],
    queryFn: async () => {
      if (!searchParams.entityId) {
        throw new Error("No entity ID provided");
      }
      const response = await http.get<T>(`${url}${searchParams.entityId}`);
      return response.data;
    },
    enabled: !currentRecord && !!searchParams.entityId && !isLoading,
    staleTime: 5 * 60 * 1000, // Consider data fresh for 5 minutes
    retry: (failureCount, error) => {
      // Don't retry on 404 errors
      if (error instanceof Error && error.message.includes("404")) {
        return false;
      }
      return failureCount < 3;
    },
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000), // Exponential backoff
  });

  // Use either the currentRecord from table or the fetched record
  const effectiveRecord = currentRecord || fetchedRecord;
  const isLoadingRecord = isLoading || isFetchingRecord;

  // Check if we're using a fetched record (no table context for navigation)
  const isFetchedRecord = !currentRecord && !!fetchedRecord;

  // Clean up entityId from URL if it doesn't match any record and we're not fetching
  React.useEffect(() => {
    if (
      searchParams.entityId &&
      !currentRecord &&
      !fetchedRecord &&
      !isFetchingRecord &&
      !isLoading &&
      fetchError
    ) {
      // Record not found, clean up URL
      console.warn(`Record with ID ${searchParams.entityId} not found`);
    }
  }, [
    searchParams.entityId,
    currentRecord,
    fetchedRecord,
    isFetchingRecord,
    isLoading,
    fetchError,
  ]);

  // Process navigation with useTransition to prevent blinking
  const processNavigation = React.useCallback(
    async (targetId: string) => {
      if (isNavigatingRef.current) {
        navigationQueueRef.current = targetId;
        return;
      }

      isNavigatingRef.current = true;

      try {
        // Use startTransition to mark this update as non-urgent
        startTransition(() => {
          setSearchParams({ entityId: targetId, modalType: "edit" }).then(
            () => {
              // Process any queued navigation
              if (navigationQueueRef.current) {
                const queuedId = navigationQueueRef.current;
                navigationQueueRef.current = null;
                // Use a small delay to prevent visual blinking
                setTimeout(() => {
                  isNavigatingRef.current = false;
                  processNavigation(queuedId);
                }, 50);
              } else {
                isNavigatingRef.current = false;
              }
            },
          );
        });
      } catch (error) {
        console.error("Navigation error:", error);
        isNavigatingRef.current = false;
      }
    },
    [setSearchParams, startTransition],
  );

  const onPrev = React.useCallback(() => {
    if (prevId) {
      processNavigation(prevId);
    }
  }, [prevId, processNavigation]);

  const onNext = React.useCallback(() => {
    if (nextId) {
      processNavigation(nextId);
    }
  }, [nextId, processNavigation]);

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

  // Update form values when effectiveRecord changes and is not loading
  useEffect(() => {
    if (
      !isLoadingRecord &&
      effectiveRecord &&
      effectiveRecord.id !== previousRecordIdRef.current
    ) {
      // Ensure all form fields have explicit values, including empty arrays for missing fields
      const formData = {
        ...effectiveRecord,
        roles: effectiveRecord.roles || [], // Ensure roles is always an array
      };

      // Use setTimeout to ensure reset happens after any potential race conditions
      setTimeout(() => {
        reset(formData, { keepDefaultValues: false });
      }, 0);

      previousRecordIdRef.current = effectiveRecord.id;
    }
  }, [effectiveRecord, isLoadingRecord, reset]);

  const handleClose = useCallback(() => {
    reset();
    // Just clear the URL - no need to reset row selection separately
    setSearchParams({ modalType: null, entityId: null });
  }, [reset, setSearchParams]);

  const { mutateAsync } = useApiMutation<
    T, // The response data type
    T, // The variables type
    unknown, // The context type
    T // The form values type for error handling
  >({
    mutationFn: async (values: T) => {
      const response = await http.put<T>(
        `${url}${effectiveRecord?.id}`,
        values,
      );
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

      // Also invalidate the single record query if it was fetched
      if (isFetchedRecord) {
        queryClient.invalidateQueries({
          queryKey: [queryKey, "single", searchParams.entityId, url],
        });
      }

      // * Reset the form to the new values
      reset(newValues);

      // * Close the modal (which also clears row selection via URL)
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
              {titleComponent ? (
                titleComponent(effectiveRecord as T)
              ) : (
                <div>
                  {isLoadingRecord
                    ? "Loading record..."
                    : fieldKey && effectiveRecord
                      ? effectiveRecord[fieldKey]
                      : title}
                </div>
              )}
            </DialogTitle>
            {!isLoadingRecord && effectiveRecord && (
              <DialogDescription>
                Last updated on{" "}
                {formatToUserTimezone(effectiveRecord.updatedAt, {
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
                    disabled={!prevId || isPending || isFetchedRecord}
                    onClick={onPrev}
                  >
                    <Icon
                      icon={faChevronUp}
                      className={cn(
                        "size-5",
                        (isPending || isFetchedRecord) && "opacity-50",
                      )}
                    />
                    <span className="sr-only">Previous</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  {isFetchedRecord ? (
                    <p>Navigation unavailable when viewing record directly</p>
                  ) : (
                    <p>
                      Navigate <Kbd variant="outline">↑</Kbd>
                    </p>
                  )}
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-7 w-7"
                    disabled={!nextId || isPending || isFetchedRecord}
                    onClick={onNext}
                  >
                    <Icon
                      icon={faChevronDown}
                      className={cn(
                        "h-5 w-5",
                        (isPending || isFetchedRecord) && "opacity-50",
                      )}
                    />
                    <span className="sr-only">Next</span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  {isFetchedRecord ? (
                    <p>Navigation unavailable when viewing record directly</p>
                  ) : (
                    <p>
                      Navigate <Kbd variant="outline">↓</Kbd>
                    </p>
                  )}
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
            {isLoadingRecord ? (
              <ComponentLoader message={`Loading ${title}...`} />
            ) : fetchError ? (
              <div className="flex flex-col items-center justify-center space-y-3 py-8">
                <p className="text-sm text-muted-foreground">
                  {fetchError instanceof Error &&
                  fetchError.message.includes("404")
                    ? "Record not found. It may have been deleted."
                    : "Failed to load record. Please try again."}
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    // Close modal and navigate back
                    setSearchParams({ modalType: null, entityId: null });
                  }}
                >
                  Close
                </Button>
              </div>
            ) : (
              formComponent
            )}
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            {!fetchError && (
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={isSubmitting}
                title={title}
              />
            )}
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
          // When closing, clear the URL selection
          const el = selectedRowKey
            ? document.getElementById(selectedRowKey)
            : null;
          setSearchParams({ modalType: null, entityId: null });

          // Focus back to the row after closing
          setTimeout(() => el?.focus(), 0);
        }
      }}
    >
      {dialogContent}
    </Dialog>
  );
}
