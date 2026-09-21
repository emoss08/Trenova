import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  EQUIPMENT_OVERRIDE_REASON_MAX_LENGTH,
  equipmentOverrideFormSchema,
  type EquipmentOverrideFormValues,
} from "@/lib/equipment-verification";
import {
  overrideCarrierEquipmentVerification,
  type CarrierEquipmentVerification,
} from "@/lib/graphql/carrier-intelligence";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

export function EquipmentOverrideDialog({
  verification,
  open,
  onOpenChange,
  onOverridden,
}: {
  verification: CarrierEquipmentVerification | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onOverridden: (verification: CarrierEquipmentVerification) => void;
}) {
  const t = useT();

  const form = useForm<EquipmentOverrideFormValues>({
    resolver: zodResolver(equipmentOverrideFormSchema),
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) {
      reset({ reason: "" });
    }
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    CarrierEquipmentVerification,
    EquipmentOverrideFormValues,
    unknown,
    EquipmentOverrideFormValues
  >({
    form,
    resourceName: "Equipment verification override",
    mutationFn: (values) => {
      if (!verification) {
        throw new Error("No verification selected");
      }
      return overrideCarrierEquipmentVerification(verification.id, values.reason.trim());
    },
    onSuccess: (updated) => {
      toast.success(t("Verification overridden"), {
        description: t("The equipment is cleared for this assignment."),
      });
      onOverridden(updated);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("Override verification")}</DialogTitle>
          <DialogDescription>
            {verification?.result === "Mismatch"
              ? t(
                  "The equipment is registered to a different carrier. Record why it is acceptable on this assignment; your name and reason are kept with the verification.",
                )
              : t(
                  "The equipment could not be confirmed. Record why it is acceptable on this assignment; your name and reason are kept with the verification.",
                )}
          </DialogDescription>
        </DialogHeader>
        {verification?.mismatchReason ? (
          <p className="bg-muted/50 rounded-md border px-3 py-2 text-sm">
            {verification.mismatchReason}
          </p>
        ) : null}
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            aria-label={t("Override verification")}
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1}>
              <FormControl>
                <TextareaField<EquipmentOverrideFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("e.g., Unit is leased on to the carrier; lease agreement on file")}
                  rules={{ required: true }}
                  maxLength={EQUIPMENT_OVERRIDE_REASON_MAX_LENGTH}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} disabled={!verification}>
                {t("Override verification")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
