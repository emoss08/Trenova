/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import {
  consolidationGroupSchema,
  ConsolidationStatus,
  type ConsolidationGroupSchema,
} from "@/lib/schemas/consolidation-schema";
import { api } from "@/services/api";
import { TableSheetProps } from "@/types/data-table";
import { APIError } from "@/types/errors";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { FormProvider, useForm, type Path } from "react-hook-form";
import { toast } from "sonner";
import { ConsolidationForm } from "./consolidation-form";

export function ConsolidationCreateSheet({
  open,
  onOpenChange,
}: TableSheetProps) {
  const queryClient = useQueryClient();
  const [shipmentErrors, setShipmentErrors] = useState<string | null>(null);

  const form = useForm({
    resolver: zodResolver(consolidationGroupSchema),
    defaultValues: {
      status: ConsolidationStatus.New,
      shipments: [],
    },
  });

  const { reset } = form;

  const createMutation = useApiMutation({
    mutationFn: (values: ConsolidationGroupSchema) =>
      api.consolidations.create(values),
    onSuccess: async () => {
      await broadcastQueryInvalidation({ queryKey: ["consolidation-list"] });
      await queryClient.invalidateQueries({
        queryKey: ["consolidation-list"],
      });

      toast.success("Consolidation created successfully");
      onOpenChange(false);
      reset();
    },
    onError: (error) => {
      const apiError = error instanceof APIError ? error : null;

      apiError?.getFieldErrors().forEach((fieldError) => {
        form.setError(fieldError.name as Path<ConsolidationGroupSchema>, {
          message: fieldError.reason,
        });
        setShipmentErrors(fieldError.reason);
      });
    },
  });

  const handleSubmit = useCallback(
    async (values: ConsolidationGroupSchema) => {
      await createMutation.mutateAsync(values);
    },
    [createMutation],
  );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-[780px]">
        <SheetHeader className="px-4 py-2 space-y-0">
          <SheetTitle>Create New Consolidation</SheetTitle>
          <SheetDescription>
            Create a new consolidation group by selecting shipments to
            consolidate together.
          </SheetDescription>
        </SheetHeader>
        <FormProvider {...form}>
          <Form onSubmit={form.handleSubmit(handleSubmit)}>
            <SheetBody>
              <ConsolidationForm shipmentErrors={shipmentErrors} />
            </SheetBody>
            <FormSaveDock position="right" />
          </Form>
        </FormProvider>
      </SheetContent>
    </Sheet>
  );
}
