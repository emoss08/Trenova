import { LocationAutocompleteField } from "@/components/autocomplete-fields";
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
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import {
  locateTrailerPayloadSchema,
  type LocateTrailerPayload,
} from "@/types/trailer";
import { useQueryClient } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type LocateTrailerDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  trailerId: string;
  targetLocationId?: string;
  onLocated?: () => void;
};

export function LocateTrailerDialog({
  open,
  onOpenChange,
  trailerId,
  targetLocationId,
  onLocated,
}: LocateTrailerDialogProps) {
  const queryClient = useQueryClient();

  const form = useForm<LocateTrailerPayload>({
    resolver: zodResolver(locateTrailerPayloadSchema),
    defaultValues: {
      newLocationId: "",
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    setError,
    setValue,
    formState: { isSubmitting },
  } = form;

  useEffect(() => {
    if (open && targetLocationId) {
      setValue("newLocationId", targetLocationId);
    }
  }, [open, targetLocationId, setValue]);

  const { mutateAsync } = useApiMutation({
    mutationFn: (payload: LocateTrailerPayload) =>
      apiService.trailerService.locate(trailerId, payload),
    resourceName: "Trailer",
    setFormError: setError,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["trailer-list"] });
      toast.success("Trailer located successfully");
      onLocated?.();
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const onSubmit = useCallback(
    async (values: LocateTrailerPayload) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent className="sm:max-w-125">
        <DialogHeader>
          <DialogTitle>Locate Trailer</DialogTitle>
          <DialogDescription>
            Set the trailer&apos;s new location. The system will create and
            complete an empty reposition move automatically.
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={(e) => {
            e.stopPropagation();
            void handleSubmit(onSubmit)(e);
          }}
        >
          <FormGroup cols={1} className="pb-4">
            <FormControl>
              <LocationAutocompleteField
                control={control}
                name="newLocationId"
                label="New Location"
                placeholder="Select location"
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText="Locating..."
            >
              Locate Trailer
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
