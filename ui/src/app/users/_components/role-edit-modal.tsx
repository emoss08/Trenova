/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

/* eslint-disable react-hooks/exhaustive-deps */
import { useDataTable } from "@/components/data-table/data-table-provider";
import { FormSaveDock } from "@/components/form";
import { Form } from "@/components/ui/form";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { searchParamsParser } from "@/hooks/use-data-table-state";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import { queries } from "@/lib/queries";
import { roleSchema, type RoleSchema } from "@/lib/schemas/user-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { RoleForm } from "./role-form";

export function EditRoleSheet({
  currentRecord,
}: EditTableSheetProps<RoleSchema>) {
  const { table, rowSelection, isLoading } = useDataTable();
  const queryClient = useQueryClient();
  const sheetRef = useRef<HTMLDivElement>(null);
  const [, setSearchParams] = useQueryStates(searchParamsParser);
  const { isPopout, closePopout } = usePopoutWindow();
  const initialLoadRef = useRef(false);

  const previousRecordIdRef = useRef<string | number | null>(null);
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

  const {
    data: roleDetails,
    isLoading: isDetailsLoading,
    // isError: isDetailsError,
  } = useQuery({
    ...queries.role.getById(currentRecord?.id ?? ""),
  });

  const form = useForm({
    resolver: zodResolver(roleSchema),
    defaultValues: roleDetails,
    mode: "onChange",
  });

  const {
    setError,
    reset,
    handleSubmit,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: RoleSchema) => {
      const response = await http.put<RoleSchema>(
        `/roles/${currentRecord?.id}`,
        values,
      );
      return response.data;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["role", currentRecord?.id],
      });

      // * snapshot of the previous value
      const previousRole = queryClient.getQueryData([
        "role",
        currentRecord?.id,
      ]);

      // * optimistically update to the new value
      queryClient.setQueryData(["role", currentRecord?.id], newValues);

      return { previousRole, newValues };
    },
    onSuccess: (newValues) => {
      toast.success("Changes have been saved", {
        description: `Role updated successfully`,
      });

      broadcastQueryInvalidation({
        queryKey: ["role", "role-list"],
        options: {
          correlationId: `update-role-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      reset(newValues);

      // * Reset the row selection
      table.resetRowSelection();

      // * Close the sheet
      setSearchParams({ modalType: null, entityId: null });

      // * If the page is a popout, close it
      if (isPopout) {
        closePopout();
      }
    },
    setFormError: setError,
    resourceName: "Role",
  });

  const onSubmit = useCallback(
    async (values: RoleSchema) => {
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

  useEffect(() => {
    if (roleDetails && !isDetailsLoading && !initialLoadRef.current) {
      reset(roleDetails, {
        keepDirty: false, // Don't keep dirty state
        keepValues: false, // Don't keep current values
      });
      initialLoadRef.current = true;
    }
  }, [roleDetails, isDetailsLoading, reset]);

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
  }, [open, isSubmitting, handleSubmit, onSubmit]);

  return (
    <>
      <Sheet
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
        <SheetContent
          className="w-[1000px] sm:max-w-[1000px]"
          withClose={false}
          ref={sheetRef}
        >
          <VisuallyHidden>
            <SheetHeader>
              <SheetTitle>Shipment Details</SheetTitle>
            </SheetHeader>
            <SheetDescription>{roleDetails?.description}</SheetDescription>
          </VisuallyHidden>

          <FormProvider {...form}>
            <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
              <RoleSheetBody />
              <FormSaveDock position="right" />
            </Form>
          </FormProvider>
        </SheetContent>
      </Sheet>
    </>
  );
}

function RoleSheetBody() {
  return (
    <SheetBody>
      <RoleForm />
    </SheetBody>
  );
}
