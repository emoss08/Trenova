import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
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
import { useUnsavedChanges } from "@/hooks/use-form";
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  type CustomerSchema,
  customerSchema,
} from "@/lib/schemas/customer-schema";
import { BillingCycleType } from "@/types/billing";
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
        billingProfile: {
          billingCycleType: BillingCycleType.Immediate,
          hasOverrides: false,
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
      const response = await http.post("/customers", values);
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
    formState: { isDirty, isSubmitting, isSubmitSuccessful },
    handleSubmit,
    onSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const {
    showWarning,
    handleClose: onClose,
    handleConfirmClose,
    handleCancelClose,
  } = useUnsavedChanges({
    isDirty,
    onClose: handleClose,
  });

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, reset, onOpenChange]);

  return (
    <>
      <Dialog open={open} onOpenChange={onClose}>
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
                <Button type="button" variant="outline" onClick={onClose}>
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

      {showWarning && (
        <AlertDialog open={showWarning} onOpenChange={handleCancelClose}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
              <AlertDialogDescription>
                You have unsaved changes. Are you sure you want to close this
                form? All changes will be lost.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel onClick={handleCancelClose}>
                Continue Editing
              </AlertDialogCancel>
              <AlertDialogAction onClick={handleConfirmClose}>
                Discard Changes
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </>
  );
}
