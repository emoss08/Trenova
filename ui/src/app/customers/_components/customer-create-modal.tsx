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
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  type CustomerSchema,
  customerSchema,
} from "@/lib/schemas/customer-schema";
import { BillingCycleType, PaymentTerm } from "@/types/billing";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { CustomerForm } from "./customer-form";

export function CreateCustomerModal({ open, onOpenChange }: TableSheetProps) {
  const { isPopout, closePopout } = usePopoutWindow();

  const form = useFormWithSave({
    resourceName: "Customer",
    formOptions: {
      resolver: zodResolver(customerSchema),
      defaultValues: {
        status: Status.Active,
        name: "",
        code: "",
        description: "",
        addressLine1: "",
        addressLine2: "",
        city: "",
        postalCode: "",
        stateId: "",
        isGeocoded: false,
        placeId: "",
        longitude: 0,
        latitude: 0,
        consolidationPriority: 1,
        allowConsolidation: true,
        exclusiveConsolidation: false,
        billingProfile: {
          billingCycleType: BillingCycleType.Immediate,
          hasOverrides: false,
          enforceCustomerBillingReq: false,
          validateCustomerRates: false,
          paymentTerm: PaymentTerm.Net30,
          autoTransfer: false,
          autoMarkReadyToBill: false,
          autoBill: false,
          specialInstructions: "",
          documentTypes: [],
        },
        emailProfile: {
          subject: "",
          comment: "",
          fromEmail: "",
          blindCopy: "",
          attachmentName: "",
          readReceipt: false,
        },
      },
    },
    mutationFn: async (values: CustomerSchema) => {
      const response = await http.post("/customers/", values);
      return response.data;
    },
    onSuccess: () => {
      onOpenChange(false);

      broadcastQueryInvalidation({
        queryKey: ["customer", "customer-list"],
        options: {
          correlationId: `create-customer-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

  const {
    formState: { isSubmitting, isSubmitSuccessful },
    handleSubmit,
    onSubmit,
    reset,
  } = form;

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    if (isSubmitSuccessful) {
      reset();
    }
  }, [isSubmitSuccessful, reset]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="md:max-w-[700px] lg:max-w-[900px]">
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Create Customer</DialogTitle>
            <DialogDescription>
              Create a new customer to manage their billing and other
              information.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <FormProvider {...form}>
          <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
            <DialogBody className="p-0">
              <CustomerForm />
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={isSubmitting}
                title="Customer"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
