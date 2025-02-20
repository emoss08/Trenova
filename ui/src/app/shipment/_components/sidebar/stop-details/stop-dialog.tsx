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
import { ComponentLoader } from "@/components/ui/component-loader";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState } from "@/components/ui/empty-state";
import { Form } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useUnsavedChanges } from "@/hooks/use-form";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { http } from "@/lib/http-client";
import { stopSchema, type StopSchema } from "@/lib/schemas/stop-schema";
import { type TableSheetProps } from "@/types/data-table";
import { type APIError } from "@/types/errors";
import { StopStatus } from "@/types/stop";
import {
  faExclamation,
  faExclamationCircle,
  faExclamationTriangle,
  faInfoCircle,
  faXmark,
} from "@fortawesome/pro-solid-svg-icons";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useLocalStorage } from "@uidotdev/usehooks";
import { useCallback, useEffect } from "react";
import { FormProvider, type Path, useForm } from "react-hook-form";
import { toast } from "sonner";
import { StopDialogForm } from "./stop-dialog-form";

type StopDialogProps = TableSheetProps & {
  stopId: string;
  isEditing: boolean;
};

export function StopDialog({
  open,
  onOpenChange,
  stopId,
  isEditing,
}: StopDialogProps) {
  const {
    data: existingStop,
    isLoading: isLoadingStop,
    isError: isStopError,
  } = useQuery({
    queryKey: ["stop", stopId],
    queryFn: async () => {
      const response = await http.get<StopSchema>(`/stops/${stopId}/`);
      return response.data;
    },
    enabled: isEditing && open,
  });

  const form = useForm<StopSchema>({
    resolver: yupResolver(stopSchema),
    defaultValues: {
      status: StopStatus.New,
      sequence: 0,
      pieces: 0,
      weight: 0,
      plannedArrival: 0,
      plannedDeparture: 0,
      actualArrival: 0,
      actualDeparture: 0,
      addressLine: "",
    },
  });

  const {
    setError,
    formState: { isDirty, isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (existingStop && !isLoadingStop) {
      reset(existingStop);
    }
  }, [existingStop, isLoadingStop, reset]);

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
    mutationFn: async (values: StopSchema) => {
      if (isEditing) {
        const response = await http.put(`/stops/${stopId}/`, values);
        return response.data;
      } else {
        const response = await http.post("/stops/", values);
        return response.data;
      }
    },
    onSuccess: () => {
      toast.success(
        isEditing ? "Stop updated successfully" : "Stop added successfully",
        {
          description: isEditing
            ? "The stop has been updated with the new details."
            : "The stop has been added to the shipment.",
        },
      );
      onOpenChange(false);
      reset();

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["stop", "shipment", "assignment"],
        options: {
          correlationId: `${isEditing ? "update" : "create"}-shipment-stop-${Date.now()}`,
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
          setError(fieldError.name as Path<StopSchema>, {
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
    async (values: StopSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
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
            <DialogTitle>{isEditing ? "Edit Stop" : "Add Stop"}</DialogTitle>
            <DialogDescription>
              {isEditing
                ? "Edit the stop details for this shipment."
                : "Add a new stop to the shipment."}
            </DialogDescription>
          </DialogHeader>
          <StopDialogNotice />
          <FormProvider {...form}>
            <Form className="space-y-0 p-0" onSubmit={handleSubmit(onSubmit)}>
              <DialogBody>
                {isLoadingStop ? (
                  <ComponentLoader message="Loading stop details..." />
                ) : isStopError ? (
                  <EmptyState
                    className="border-none hover:bg-transparent"
                    icons={[
                      faExclamationTriangle,
                      faExclamationCircle,
                      faExclamation,
                    ]}
                    title="Error loading stop details"
                    description="Please try again later."
                  />
                ) : (
                  <StopDialogForm />
                )}
              </DialogBody>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={onClose}>
                  Cancel
                </Button>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button type="submit" isLoading={isSubmitting}>
                        {isEditing ? "Update" : "Add"}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent className="flex items-center gap-2">
                      <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                        Ctrl
                      </kbd>
                      <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-muted-foreground/60 px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                        Enter
                      </kbd>
                      <p>to save and close the stop</p>
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

function StopDialogNotice() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    "showStopDialogNotice",
    true,
  );

  const handleClose = () => {
    setNoticeVisible(false);
  };
  return noticeVisible ? (
    <div className="bg-muted px-4 py-3 text-foreground">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-foreground"
            aria-hidden="true"
          />
          <div className="flex grow flex-col justify-between gap-2 md:flex-row">
            <span className="text-sm">
              All times are displayed in your local timezone. Please ensure
              location details are accurate for proper routing.
            </span>
          </div>
        </div>
        <Button
          variant="secondary"
          className="group -my-1.5 -me-2 size-8 shrink-0 p-0 hover:bg-transparent"
          onClick={handleClose}
          aria-label="Close banner"
        >
          <Icon
            icon={faXmark}
            className="opacity-60 transition-opacity group-hover:opacity-100"
            aria-hidden="true"
          />
        </Button>
      </div>
    </div>
  ) : null;
}
