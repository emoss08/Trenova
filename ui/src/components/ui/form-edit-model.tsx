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
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useUnsavedChanges } from "@/hooks/use-form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { formatToUserTimezone } from "@/lib/date";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/user-store";
import { type EditTableSheetProps } from "@/types/data-table";
import { APIError } from "@/types/errors";
import { type API_ENDPOINTS } from "@/types/server";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import {
  FormProvider,
  Path,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import { toast } from "sonner";
import { type ObjectSchema } from "yup";
import { ComponentLoader } from "./component-loader";
import { Form } from "./form";
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
  schema: ObjectSchema<T>;
  className?: string;
  fieldKey?: keyof T;
};

export function FormEditModal<T extends FieldValues>({
  open,
  onOpenChange,
  currentRecord,
  url,
  title,
  formComponent,
  queryKey,
  fieldKey,
  form,
  className,
  isLoading,
  error,
}: FormEditModalProps<T>) {
  const { isPopout, closePopout } = usePopoutWindow();
  const { user } = useAuthStore();

  const {
    setError,
    formState: { isDirty, isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const mutation = useMutation({
    mutationFn: async (values: T) => {
      const response = await http.put(`${url}${currentRecord?.id}`, values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Changes have been saved.", {
        description: `${title} updated successfully`,
      });
      onOpenChange(false);
      reset();

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
      // TODO(Wolfred): Add a standardized error handling system that catches all errors and displays them in a consistent way
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

  const {
    showWarning,
    handleClose: onClose,
    handleConfirmClose,
    handleCancelClose,
  } = useUnsavedChanges({
    isDirty,
    onClose: handleClose,
  });

  const onSubmit = useCallback(
    async (values: T) => {
      await mutation.mutateAsync(values);
    },
    [mutation.mutateAsync],
  );

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

  useEffect(() => {
    if (!isLoading && currentRecord) {
      reset(currentRecord);
    }
  }, [currentRecord, isLoading, reset]);

  // If there's an error, show a toast and close the modal
  useEffect(() => {
    if (error) {
      toast.error("Failed to load record", {
        description: "Please try again in a moment",
      });
      onOpenChange(false);
    }
  }, [error, onOpenChange]);

  const dialogContent = (
    <DialogContent
      withClose={!isLoading}
      className={cn("max-w-[450px]", className)}
    >
      <DialogHeader>
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
            {formatToUserTimezone(currentRecord.updatedAt, user?.timezone)}
          </DialogDescription>
        )}
      </DialogHeader>

      {isLoading ? (
        <DialogBody>
          <ComponentLoader />
        </DialogBody>
      ) : (
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <DialogBody>{formComponent}</DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={onClose}>
                Cancel
              </Button>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button type="submit" isLoading={isSubmitting}>
                      Save {isPopout ? "and Close" : "Changes"}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent className="flex items-center gap-2">
                    <kbd className="inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-background">
                      Ctrl
                    </kbd>
                    <kbd className="inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-background">
                      Enter
                    </kbd>
                    <p>to save and close the {title}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </DialogFooter>
          </Form>
        </FormProvider>
      )}
    </DialogContent>
  );

  return (
    <>
      <Dialog open={open} onOpenChange={onClose}>
        {dialogContent}
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
