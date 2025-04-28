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
import { Icon } from "@/components/ui/icons";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import {
  assignmentSchema,
  type AssignmentSchema,
} from "@/lib/schemas/assignment-schema";
import { AssignmentStatus } from "@/types/assignment";
import { faLoader } from "@fortawesome/pro-solid-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { AssignmentForm } from "./assignment-form";

export function AssignmentDialog({
  open,
  onOpenChange,
  shipmentMoveId,
  assignmentId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentMoveId: string;
  assignmentId?: string;
}) {
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

  const form = useForm({
    resolver: zodResolver(assignmentSchema),
    defaultValues: {
      status: AssignmentStatus.New,
      shipmentMoveId: shipmentMoveId,
      primaryWorkerId: "",
      secondaryWorkerId: "",
      tractorId: "",
      trailerId: "",
      createdAt: undefined,
      updatedAt: undefined,
      id: undefined,
      version: undefined,
    },
  });

  const {
    setError,
    formState: { isSubmitting, isSubmitSuccessful },
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

  const queryClient = useQueryClient();

  const { mutateAsync: createAssignment } = useApiMutation({
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

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["shipment", "shipment-list", "stop", "assignment", "move"],
        options: {
          correlationId: `${isEditing ? "update" : "create"}-shipment-move-assignment-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      // Also directly invalidate the specific move query
      queryClient.invalidateQueries({
        queryKey: ["moves", shipmentMoveId],
      });
    },
    setFormError: setError,
    resourceName: "Assignment",
  });

  // Reset the form when the mutation is successful
  useEffect(() => {
    if (isSubmitSuccessful) {
      reset();
    }
  }, [isSubmitSuccessful, reset]);

  const onSubmit = useCallback(
    async (values: AssignmentSchema) => {
      await createAssignment(values);
    },
    [createAssignment],
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
                  <AssignmentLoading />
                ) : (
                  <AssignmentForm />
                )}
              </DialogBody>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={handleClose}>
                  Cancel
                </Button>
                <FormSaveButton
                  type="button"
                  onClick={() => handleSubmit(onSubmit)()}
                  isSubmitting={isSubmitting}
                  title={isEditing ? "Reassign" : "Assign"}
                  text={isEditing ? "Reassign" : "Assign"}
                />
              </DialogFooter>
            </Form>
          </FormProvider>
        </DialogContent>
      </Dialog>
    </>
  );
}

function AssignmentLoading() {
  return (
    <div className="w-full px-6 py-10">
      <div className="flex flex-col gap-2 items-center justify-center text-center">
        <Icon icon={faLoader} className="animate-spin size-8 text-blue-500" />
        <div className="flex flex-col">
          <p className="mt-2 text-sm text-foreground">
            Loading Assignment details...
          </p>
          <p className="mt-2 text-2xs text-muted-foreground">
            If this takes longer than a few seconds, please check your internet
            connection.
          </p>
        </div>
      </div>
    </div>
  );
}
