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
import { Button } from "@/components/ui/button";
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
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useUnsavedChanges } from "@/hooks/use-form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { type TableSheetProps } from "@/types/data-table";
import { type APIError } from "@/types/errors";
import { type API_ENDPOINTS } from "@/types/server";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import {
  type FieldValues,
  FormProvider,
  type Path,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";
import { type ObjectSchema } from "yup";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./tooltip";

type FormCreateModalProps<T extends FieldValues> = TableSheetProps & {
  url: API_ENDPOINTS;
  title: string;
  queryKey: string;
  formComponent: React.ReactNode;
  form: UseFormReturn<T>;
  schema: ObjectSchema<T>;
  className?: string;
  notice?: React.ReactNode;
};

export function FormCreateModal<T extends FieldValues>({
  open,
  onOpenChange,
  title,
  formComponent,
  form,
  className,
  url,
  queryKey,
  notice,
}: FormCreateModalProps<T>) {
  const { isPopout, closePopout } = usePopoutWindow();

  const {
    setError,
    formState: { isDirty, isSubmitting, isSubmitSuccessful },
    handleSubmit,
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

  const { mutateAsync } = useMutation({
    mutationFn: async (values: T) => {
      const response = await http.post(url, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Changes have been saved.", {
        description: `${title} created successfully`,
      });
      onOpenChange(false);

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: [queryKey],
        options: { correlationId: `create-${queryKey}-${Date.now()}` },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as Path<T>, {
            message: fieldError.reason,
          });
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
    onSettled: () => {
      if (isPopout) {
        closePopout();
      }
    },
  });

  const onSubmit = useCallback(
    async (values: T) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, reset]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        open &&
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
      <Dialog open={open} onOpenChange={onClose}>
        <DialogContent className={cn("max-w-[450px]", className)}>
          <DialogHeader>
            <DialogTitle>Add New {title}</DialogTitle>
            <DialogDescription>
              Please fill out the form below to create a new {title}.
            </DialogDescription>
          </DialogHeader>
          {notice ? notice : null}
          <FormProvider {...form}>
            <Form onSubmit={handleSubmit(onSubmit)}>
              <DialogBody>{formComponent}</DialogBody>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={onClose}>
                  Cancel
                </Button>
                <TooltipProvider delayDuration={0}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button type="submit" isLoading={isSubmitting}>
                        Save {isPopout ? "and Close" : "Changes"}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent className="flex items-center gap-2">
                      <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                        Ctrl
                      </kbd>
                      <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                        Enter
                      </kbd>
                      <p>to save and close the {title}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
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
