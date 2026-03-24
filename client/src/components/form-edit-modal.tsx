import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useDataTable } from "@/contexts/data-table-context";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { api } from "@/lib/api";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { type EditTableSheetProps } from "@/types/data-table";
import { type API_ENDPOINTS } from "@/types/server";
import { TimeFormat } from "@/types/user";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronDownIcon, ChevronUpIcon, XIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import React, { useCallback, useEffect, useRef, useTransition } from "react";
import {
  FormProvider,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";
import { ComponentLoader } from "./component-loader";
import { Form } from "./ui/form";
import { Kbd } from "./ui/kbd";
import { Separator } from "./ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";

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
  const [isPending, startTransition] = useTransition();
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser, {
    history: "replace",
    throttleMs: 50,
  });
  const queryClient = useQueryClient();

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const previousRecordIdRef = useRef<string | number | null>(null);
  const navigationQueueRef = useRef<string | null>(null);
  const isNavigatingRef = useRef(false);
  const [pendingNavigationId, setPendingNavigationId] = React.useState<
    string | null
  >(null);
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
      return api.get<T>(`${url}${searchParams.entityId}/`);
    },
    enabled: !currentRecord && !!searchParams.entityId && !isLoading,
    staleTime: 5 * 60 * 1000,
    retry: (failureCount, error) => {
      if (error instanceof Error && error.message.includes("404")) {
        return false;
      }
      return failureCount < 3;
    },
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });

  const effectiveRecord = currentRecord || fetchedRecord;
  const isLoadingRecord = isLoading || isFetchingRecord;

  const isFetchedRecord = !currentRecord && !!fetchedRecord;

  React.useEffect(() => {
    if (
      searchParams.entityId &&
      !currentRecord &&
      !fetchedRecord &&
      !isFetchingRecord &&
      !isLoading &&
      fetchError
    ) {
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

  const processNavigation = useCallback(
    async (targetId: string) => {
      if (isNavigatingRef.current) {
        navigationQueueRef.current = targetId;
        return;
      }

      isNavigatingRef.current = true;

      try {
        startTransition(() => {
          void setSearchParams({
            entityId: targetId,
            modalType: "edit",
          }).then(() => {
            if (navigationQueueRef.current) {
              const queuedId = navigationQueueRef.current;
              navigationQueueRef.current = null;
              setTimeout(() => {
                isNavigatingRef.current = false;
                setPendingNavigationId(queuedId);
              }, 50);
            } else {
              isNavigatingRef.current = false;
            }
          });
        });
      } catch (error) {
        console.error("Navigation error:", error);
        isNavigatingRef.current = false;
      }
    },
    [setSearchParams, startTransition],
  );

  React.useEffect(() => {
    if (!pendingNavigationId) return;
    const nextId = pendingNavigationId;
    setPendingNavigationId(null);
    void processNavigation(nextId);
  }, [pendingNavigationId, processNavigation]);

  const onPrev = React.useCallback(() => {
    if (prevId) {
      void processNavigation(prevId);
    }
  }, [prevId, processNavigation]);

  const onNext = React.useCallback(() => {
    if (nextId) {
      void processNavigation(nextId);
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

  useEffect(() => {
    if (
      !isLoadingRecord &&
      effectiveRecord &&
      effectiveRecord.id !== previousRecordIdRef.current
    ) {
      const formData = {
        ...effectiveRecord,
        roles: effectiveRecord.roles || [], // Ensure roles is always an array
      };

      setTimeout(() => {
        reset(formData, { keepDefaultValues: false });
      }, 0);

      previousRecordIdRef.current = effectiveRecord.id;
    }
  }, [effectiveRecord, isLoadingRecord, reset]);

  const handleClose = useCallback(() => {
    reset();
    void setSearchParams({ modalType: null, entityId: null });
  }, [reset, setSearchParams]);

  const { mutateAsync } = useApiMutation<T, T, unknown, T>({
    mutationFn: async (values: T) => {
      return api.put<T>(`${url}${effectiveRecord?.id}/`, values);
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: [queryKey],
      });

      const previousRecord = queryClient.getQueryData([queryKey]);

      queryClient.setQueryData([queryKey], newValues);

      return { previousRecord, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Changes have been saved", {
        description: `${title} updated successfully`,
      });

      if (isFetchedRecord) {
        void queryClient.invalidateQueries({
          queryKey: [queryKey, "single", searchParams.entityId, url],
        });
      }

      reset(newValues);
      void setSearchParams({ modalType: null, entityId: null });
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
        void handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isSubmitting, handleSubmit, onSubmit]);

  return (
    <Dialog
      open={!!selectedRowKey}
      onOpenChange={(open) => {
        if (!open) {
          const el = selectedRowKey
            ? document.getElementById(selectedRowKey)
            : null;
          void setSearchParams({ modalType: null, entityId: null });

          setTimeout(() => el?.focus(), 0);
        }
      }}
    >
      <DialogContent
        showCloseButton={false}
        className={cn("max-w-[450px]", className)}
      >
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
                    timeFormat: TimeFormat.enum["24-hour"],
                  })}
                </DialogDescription>
              )}
            </div>
            <div className="flex h-7 items-center gap-1">
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      size="icon"
                      variant="ghost"
                      className="size-7 [&_svg]:size-5"
                      disabled={!prevId || isPending || isFetchedRecord}
                      onClick={onPrev}
                    >
                      <ChevronUpIcon
                        className={cn(
                          (isPending || isFetchedRecord) && "opacity-50",
                        )}
                      />
                      <span className="sr-only">Previous</span>
                    </Button>
                  }
                />
                <TooltipContent>
                  {isFetchedRecord ? (
                    <p>Navigation unavailable when viewing record directly</p>
                  ) : (
                    <p>
                      Navigate <Kbd>↑</Kbd>
                    </p>
                  )}
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      size="icon"
                      variant="ghost"
                      className="size-7 [&_svg]:size-5"
                      disabled={!nextId || isPending || isFetchedRecord}
                      onClick={onNext}
                    >
                      <ChevronDownIcon
                        className={cn(
                          (isPending || isFetchedRecord) && "opacity-50",
                        )}
                      />
                      <span className="sr-only">Next</span>
                    </Button>
                  }
                ></TooltipTrigger>
                <TooltipContent>
                  {isFetchedRecord ? (
                    <p>Navigation unavailable when viewing record directly</p>
                  ) : (
                    <p>
                      Navigate <Kbd>↓</Kbd>
                    </p>
                  )}
                </TooltipContent>
              </Tooltip>
              <Separator orientation="vertical" className="mx-1" />
              <DialogClose
                render={
                  <Button
                    size="icon"
                    variant="ghost"
                    className="size-7 [&_svg]:size-4"
                  >
                    <XIcon />
                    <span className="sr-only">Close</span>
                  </Button>
                }
              ></DialogClose>
            </div>
          </div>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <div>
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
                      void setSearchParams({
                        modalType: null,
                        entityId: null,
                      });
                    }}
                  >
                    Close
                  </Button>
                </div>
              ) : (
                formComponent
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              {!fetchError && (
                <Button
                  type="submit"
                  isLoading={isSubmitting}
                  loadingText="Saving..."
                >
                  Save and Close
                </Button>
              )}
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
