import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { cancelFuelCard, type FuelCardRow } from "@/lib/graphql/fuel-card";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { cancelFuelCardSchema, type CancelFuelCardValues } from "@trenova/shared/types/fuel-card";
import { BanIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { maskedCardNumber } from "./fuel-card-columns";

type CancelFuelCardDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  card: FuelCardRow | null;
  onCancelled: () => Promise<void> | void;
};

export function CancelFuelCardDialog({
  open,
  onOpenChange,
  card,
  onCancelled,
}: CancelFuelCardDialogProps) {
  const form = useForm<CancelFuelCardValues>({
    resolver: zodResolver(cancelFuelCardSchema) as Resolver<CancelFuelCardValues>,
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    FuelCardRow,
    CancelFuelCardValues,
    unknown,
    CancelFuelCardValues
  >({
    form,
    resourceName: "Fuel Card",
    mutationFn: async (values) => {
      if (!card) throw new Error("No fuel card selected");
      return cancelFuelCard({ id: card.id, version: card.version, reason: values.reason.trim() });
    },
    onSuccess: async () => {
      toast.success("Card cancelled", {
        description: "It stays on file for the purchases already made with it.",
      });
      await onCancelled();
      onOpenChange(false);
    },
  });

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <AlertDialogHeader>
              <AlertDialogMedia className="bg-destructive/10 text-destructive">
                <BanIcon />
              </AlertDialogMedia>
              <AlertDialogTitle>
                Cancel {card ? `${card.label} (${maskedCardNumber(card.lastFour)})` : "this card"}?
              </AlertDialogTitle>
              <AlertDialogDescription>
                Cancelling is permanent. The card stops matching statement rows and cannot be made
                active again; purchases already recorded against it are kept. To pause a card
                instead, suspend it.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <FormGroup cols={1} className="mt-4">
              <FormControl cols="full">
                <TextareaField
                  control={control}
                  name="reason"
                  label="Reason"
                  placeholder="e.g. Reported lost by the driver on 14 May"
                  rules={{ required: true }}
                  maxLength={500}
                  description="At least ten characters. Kept with the card as the record of why it was cancelled."
                />
              </FormControl>
            </FormGroup>
            <AlertDialogFooter className="mt-4">
              <AlertDialogCancel type="button" disabled={isPending}>
                Keep card
              </AlertDialogCancel>
              <AlertDialogAction type="submit" variant="destructive" disabled={isPending || !card}>
                {isPending ? "Cancelling..." : "Cancel card"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </Form>
        </FormProvider>
      </AlertDialogContent>
    </AlertDialog>
  );
}
