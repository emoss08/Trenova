/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { UserAutocompleteField } from "@/components/ui/autocomplete-fields";
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
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import {
  transferOwnershipSchema,
  type TransferOwnershipSchema,
} from "@/lib/schemas/transfer-ownership-schema";
import type { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

type TransferOwnershipDialogProps = {
  shipmentId: ShipmentSchema["id"];
  currentOwnerId: ShipmentSchema["ownerId"];
} & TableSheetProps;

export function TransferOwnershipDialog({
  shipmentId,
  currentOwnerId,
  ...props
}: TransferOwnershipDialogProps) {
  const form = useForm<TransferOwnershipSchema>({
    resolver: zodResolver(transferOwnershipSchema),
    defaultValues: {
      ownerId: currentOwnerId ?? "",
      shipmentId,
    },
  });

  const {
    reset,
    setError,
    control,
    formState: { isSubmitSuccessful, isSubmitting, isDirty },
    handleSubmit,
  } = form;

  const { mutateAsync } = useApiMutation({
    setFormError: setError,
    resourceName: "Ownership Transfer",
    mutationFn: async (values: TransferOwnershipSchema) => {
      const response = await http.put<ShipmentSchema>(
        `/shipments/${shipmentId}/transfer-ownership/`,
        values,
      );

      return response.data;
    },
    onSuccess: (values) => {
      toast.success("Ownership transferred successfully", {
        description: values.ownerId
          ? "The shipment has been transferred to the new owner."
          : "Ownership has been removed.",
      });

      broadcastQueryInvalidation({
        queryKey: ["shipment", "shipment-list", "stop", "assignment", "user"],
        options: {
          correlationId: `transfer-shipment-ownership-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      props.onOpenChange(false);
    },
  });

  const onSubmit = useCallback(
    async (values: TransferOwnershipSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isSubmitSuccessful]);

  return (
    <Dialog {...props}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Transfer Ownership</DialogTitle>
          <DialogDescription>
            Transfer this shipment to a different user.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              handleSubmit(onSubmit)(e);
            }}
          >
            <DialogBody>
              <FormGroup cols={1}>
                <FormControl>
                  <UserAutocompleteField
                    clearable
                    control={control}
                    name="ownerId"
                    label="Owner"
                    placeholder="Select an owner"
                    description="The user who will be the owner of the shipment."
                  />
                </FormControl>
              </FormGroup>
            </DialogBody>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => props.onOpenChange(false)}
              >
                Cancel
              </Button>
              <FormSaveButton
                type="button"
                onClick={() => handleSubmit(onSubmit)()}
                isSubmitting={isSubmitting}
                disabled={!isDirty}
                title="transfer ownership"
                text="Transfer Ownership"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
