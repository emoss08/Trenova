import { useT } from "@trenova/shared/i18n/use-t";
import { LocationAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import { locateTrailerPayloadSchema, type LocateTrailerPayload } from "@/types/trailer";
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
  const t = useT();

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
    form,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["trailer-list"] });
      toast.success(t("Trailer located successfully"));
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
          <DialogTitle>{t("Locate Trailer")}</DialogTitle>
          <DialogDescription>
            {t(
              "Set the trailer's new location. The system will create and complete an empty reposition move automatically.",
            )}
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
                label={t("New Location")}
                placeholder={t("Select location")}
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={isSubmitting} loadingText={t("Locating...")}>
              {t("Locate Trailer")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
