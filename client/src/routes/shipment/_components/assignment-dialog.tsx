import {
  TractorAutocompleteField,
  TrailerAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { ApiRequestError } from "@/lib/api";
import { LocateTrailerDialog } from "@/routes/trailer/_components/locate-trailer-dialog";
import { apiService } from "@/services/api";
import type { Assignment, AssignmentPayload } from "@/types/shipment";
import { assignmentPayloadSchema } from "@/types/shipment";
import type { Tractor } from "@/types/tractor";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { TriangleAlertIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type AssignmentDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  moveId: string;
  existingAssignment?: Assignment | null;
  onAssigned?: (assignment: Assignment) => void;
};

export function AssignmentDialog({
  open,
  onOpenChange,
  moveId,
  existingAssignment,
  onAssigned,
}: AssignmentDialogProps) {
  const queryClient = useQueryClient();
  const isEditing = !!existingAssignment?.id;

  const [continuityError, setContinuityError] = useState<{
    message: string;
    trailerId: string;
    pickupLocationId: string;
  } | null>(null);
  const [locateDialogOpen, setLocateDialogOpen] = useState(false);

  const form = useForm({
    resolver: zodResolver(assignmentPayloadSchema),
    defaultValues: {
      tractorId: "",
      trailerId: null,
      primaryWorkerId: "",
      secondaryWorkerId: null,
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    setError,
    setValue,
    getValues,
    watch,
    formState: { isSubmitting },
  } = form;

  useEffect(() => {
    if (open && existingAssignment) {
      reset({
        tractorId: existingAssignment.tractorId ?? "",
        trailerId: existingAssignment.trailerId ?? null,
        primaryWorkerId: existingAssignment.primaryWorkerId ?? "",
        secondaryWorkerId: existingAssignment.secondaryWorkerId ?? null,
      });
    } else if (open) {
      reset({
        tractorId: "",
        trailerId: null,
        primaryWorkerId: "",
        secondaryWorkerId: null,
      });
    }
  }, [open, existingAssignment, reset]);

  const watchedTrailerId = watch("trailerId");
  useEffect(() => {
    setContinuityError(null);
  }, [watchedTrailerId]);

  const { mutateAsync } = useMutation({
    mutationFn: (payload: AssignmentPayload) =>
      isEditing
        ? apiService.assignmentService.reassign(moveId, payload)
        : apiService.assignmentService.assignToMove(moveId, payload),
    onSuccess: (data: Assignment) => {
      void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      onAssigned?.(data);
      toast.success(isEditing ? "Reassigned successfully" : "Assigned successfully");
    },
    onError: (error: ApiRequestError) => {
      if (error.isBusinessError()) {
        const params = error.getParams();
        if (params.trailerId && params.pickupLocationId) {
          setContinuityError({
            message: error.data.detail || error.data.title,
            trailerId: params.trailerId,
            pickupLocationId: params.pickupLocationId,
          });
          return;
        }
      }
      handleMutationError({ error, setFormError: setError, resourceName: "Assignment" });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
    setContinuityError(null);
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: AssignmentPayload) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  const handleTractorChange = useCallback(
    (tractor: Tractor | null) => {
      if (tractor) {
        const currentPrimary = getValues("primaryWorkerId");
        const currentSecondary = getValues("secondaryWorkerId");

        if (!currentPrimary && tractor.primaryWorkerId) {
          setValue("primaryWorkerId", tractor.primaryWorkerId);
        }
        if (!currentSecondary && tractor.secondaryWorkerId) {
          setValue("secondaryWorkerId", tractor.secondaryWorkerId);
        }
      }
    },
    [setValue, getValues],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Reassign Move" : "Assign Move"}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Update the tractor, trailer, and worker assignments for this move."
              : "Assign a tractor, trailer, and workers to this move."}
          </DialogDescription>
        </DialogHeader>
        {continuityError && (
          <Alert variant="default">
            <TriangleAlertIcon />
            <AlertTitle>Trailer Location Mismatch</AlertTitle>
            <AlertDescription>{continuityError.message}</AlertDescription>
            <AlertAction>
              <Button
                type="button"
                size="xs"
                variant="outline"
                onClick={() => setLocateDialogOpen(true)}
              >
                Locate Trailer
              </Button>
            </AlertAction>
          </Alert>
        )}
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={2} className="pb-4">
            <FormControl>
              <TractorAutocompleteField
                control={control}
                name="tractorId"
                label="Tractor"
                placeholder="Select tractor"
                rules={{ required: true }}
                onOptionChange={handleTractorChange}
              />
            </FormControl>
            <FormControl>
              <TrailerAutocompleteField
                control={control}
                name="trailerId"
                label="Trailer"
                placeholder="Select trailer"
                clearable
              />
            </FormControl>
            <FormControl>
              <WorkerAutocompleteField
                control={control}
                name="primaryWorkerId"
                label="Primary Worker"
                placeholder="Select primary worker"
                rules={{ required: true }}
                clearable
              />
            </FormControl>
            <FormControl>
              <WorkerAutocompleteField
                control={control}
                name="secondaryWorkerId"
                label="Secondary Worker"
                placeholder="Select secondary worker"
                clearable
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText="Saving...">
              {isEditing ? "Reassign" : "Assign"}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
      {continuityError && (
        <LocateTrailerDialog
          open={locateDialogOpen}
          onOpenChange={setLocateDialogOpen}
          trailerId={continuityError.trailerId}
          targetLocationId={continuityError.pickupLocationId}
          onLocated={() => setContinuityError(null)}
        />
      )}
    </Dialog>
  );
}
