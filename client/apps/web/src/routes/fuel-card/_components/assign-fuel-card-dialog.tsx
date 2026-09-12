import { useT } from "@trenova/shared/i18n/use-t";
import {
  TractorAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assignFuelCard, type FuelCardRow } from "@/lib/graphql/fuel-card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { maskedCardNumber } from "./fuel-card-columns";

/**
 * A card a feed discovered arrives with no tractor and no driver on it, because
 * the transaction only named the card. Until it is assigned, purchases on it can
 * only be matched by the unit number printed on the receipt, and rows without one
 * wait in the review queue.
 */

type AssignFuelCardValues = {
  assignedTractorId: string;
  assignedWorkerId: string;
};

type AssignFuelCardDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  card: FuelCardRow | null;
  onAssigned: () => Promise<void> | void;
};

export function AssignFuelCardDialog({
  open,
  onOpenChange,
  card,
  onAssigned,
}: AssignFuelCardDialogProps) {
  const t = useT();

  const form = useForm<AssignFuelCardValues>({
    defaultValues: { assignedTractorId: "", assignedWorkerId: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) {
      return;
    }

    reset({
      assignedTractorId: card?.assignedTractorId ?? "",
      assignedWorkerId: card?.assignedWorkerId ?? "",
    });
  }, [open, card, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    FuelCardRow,
    AssignFuelCardValues,
    unknown,
    AssignFuelCardValues
  >({
    form,
    resourceName: "Fuel Card",
    mutationFn: async (values) => {
      if (!card) {
        throw new Error("No fuel card selected");
      }

      return assignFuelCard({
        id: card.id,
        version: card.version,
        // An empty picker means "not assigned", which the server stores as null
        // rather than as an empty id.
        assignedTractorId: values.assignedTractorId || null,
        assignedWorkerId: values.assignedWorkerId || null,
      });
    },
    onSuccess: async () => {
      toast.success(t("Card assigned"), {
        description: t("Purchases on this card will match to what it is assigned to."),
      });
      onOpenChange(false);
      await onAssigned();
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("Assign card {0}", card ? maskedCardNumber(card.lastFour) : "")}</DialogTitle>
          <DialogDescription>
            {t("Tie this card to the tractor it lives in, the driver who carries it, or both. Leaving both empty puts it back in the unassigned list.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit((values) => mutateAsync(values))} className="space-y-4">
            <FormGroup cols={1}>
              <FormControl cols="full">
                <TractorAutocompleteField
                  name="assignedTractorId"
                  control={control}
                  label={t("Tractor")}
                  placeholder={t("Select a tractor")}
                  clearable
                />
              </FormControl>
              <FormControl cols="full">
                <WorkerAutocompleteField
                  name="assignedWorkerId"
                  control={control}
                  label={t("Driver")}
                  placeholder={t("Select a driver")}
                  clearable
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} loadingText={t("Assigning...")}>
                {t("Assign")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
