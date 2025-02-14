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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useUnsavedChanges } from "@/hooks/use-form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  assignmentSchema,
  type AssignmentSchema,
} from "@/lib/schemas/assignment-schema";
import { AssignmentStatus } from "@/types/assignment";
import { type APIError } from "@/types/errors";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, type Path, useForm } from "react-hook-form";
import { toast } from "sonner";
import { AssignmentForm } from "./assignment-form";

type AssignmentDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentMoveId: string;
  assignmentId?: string;
};

export function AssignmentDialog({
  open,
  onOpenChange,
  shipmentMoveId,
  assignmentId,
}: AssignmentDialogProps) {
  const isEditing = !!assignmentId;

  const { data: existingAssignment, isLoading: isLoadingAssignment } = useQuery(
    {
      queryKey: ["assignment", assignmentId],
      queryFn: async () => {
        const response = await http.get<AssignmentSchema>(
          `/assignments/${assignmentId}/`,
        );
        return response.data;
      },
      enabled: isEditing && open,
    },
  );

  const form = useForm<AssignmentSchema>({
    resolver: yupResolver(assignmentSchema),
    defaultValues: {
      status: AssignmentStatus.New,
      shipmentMoveId: shipmentMoveId,
      primaryWorkerId: "",
      secondaryWorkerId: "",
      tractorId: "",
      trailerId: "",
    },
  });

  const {
    setError,
    formState: { isDirty, isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (existingAssignment && !isLoadingAssignment) {
      reset(existingAssignment);
    }
  }, [existingAssignment, isLoadingAssignment, reset]);

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

  const mutation = useMutation({
    mutationFn: async (values: AssignmentSchema) => {
      if (isEditing) {
        const response = await http.put(
          `/assignments/${assignmentId}/`,
          values,
        );
        return response.data;
      } else {
        const response = await http.post("/assignments/single/", values);
        return response.data;
      }
    },
    onSuccess: () => {
      toast.success(
        isEditing
          ? "Assignment updated successfully"
          : "Movement assigned successfully",
        {
          description: isEditing
            ? "The assignment has been updated with the new equipment and worker(s)"
            : "The movement has been assigned to the selected equipment and worker(s)",
        },
      );
      onOpenChange(false);
      reset();

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["assignment", "shipment"],
        options: {
          correlationId: `${isEditing ? "update" : "create"}-shipment-move-assignment-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as Path<AssignmentSchema>, {
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
  });

  const onSubmit = useCallback(
    async (values: AssignmentSchema) => {
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

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {isEditing ? "Edit Assignment" : "Assign"} {shipmentMoveId}
            </DialogTitle>
            <DialogDescription>
              {isEditing
                ? "Update equipment and worker(s) for this assignment."
                : "Assign equipment and worker(s) to the selected movement."}
            </DialogDescription>
          </DialogHeader>
          <FormProvider {...form}>
            <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
              <DialogBody>
                {isLoadingAssignment ? (
                  <div className="flex items-center justify-center p-4">
                    Loading assignment details...
                  </div>
                ) : (
                  <AssignmentForm />
                )}
              </DialogBody>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={onClose}>
                  Cancel
                </Button>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        type="submit"
                        isLoading={isSubmitting}
                        loadingText={
                          isEditing ? "Reassigning..." : "Assigning..."
                        }
                      >
                        {isEditing ? "Reassign" : "Assign"}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent className="flex items-center gap-2">
                      <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                        Ctrl
                      </kbd>
                      <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                        Enter
                      </kbd>
                      <p>to save and close the assignment</p>
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
