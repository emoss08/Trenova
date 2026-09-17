import { useT } from "@trenova/shared/i18n/use-t";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  CARRIER_INTEL_EVENT_RESOLUTIONS,
  resolveEventFormSchema,
  type ResolveEventFormValues,
} from "@/lib/carrier-intelligence";
import {
  resolveCarrierIntelEvent,
  type CarrierIntelEvent,
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
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export type ResolveEventDialogProps = {
  event: CarrierIntelEvent | null;
  title?: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onResolved?: (event: CarrierIntelEvent) => void;
};

const DEFAULT_VALUES: ResolveEventFormValues = {
  resolution: "NoActionRequired",
  note: "",
};

export function ResolveEventDialog({
  event,
  title,
  open,
  onOpenChange,
  onResolved,
}: ResolveEventDialogProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();

  const form = useForm<ResolveEventFormValues>({
    resolver: zodResolver(resolveEventFormSchema) as Resolver<ResolveEventFormValues>,
    defaultValues: DEFAULT_VALUES,
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) {
      reset(DEFAULT_VALUES);
    }
  }, [open, reset]);

  const options = useMemo(
    () =>
      CARRIER_INTEL_EVENT_RESOLUTIONS.map((value) => ({
        value,
        label: labels.resolution[value],
        description: labels.resolutionHint[value],
      })),
    [labels],
  );

  const { mutateAsync, isPending } = useApiMutation<
    CarrierIntelEvent,
    ResolveEventFormValues,
    unknown,
    ResolveEventFormValues
  >({
    form,
    resourceName: "Carrier intelligence event",
    mutationFn: (values) => {
      if (!event) {
        throw new Error("No event selected");
      }
      return resolveCarrierIntelEvent({
        id: event.id,
        resolution: values.resolution,
        note: values.note === "" ? null : values.note,
      });
    },
    onSuccess: (resolved) => {
      toast.success(t("Event resolved"));
      onResolved?.(resolved);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Resolve event")}</DialogTitle>
          <DialogDescription>
            {title ?? event?.summary ?? t("Record how this change was handled.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1} className="pb-2">
              <FormControl>
                <SelectField<ResolveEventFormValues>
                  control={control}
                  name="resolution"
                  label={t("Resolution")}
                  options={options}
                  placeholder={t("Choose a resolution")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <TextareaField<ResolveEventFormValues>
                  control={control}
                  name="note"
                  label={t("Note")}
                  placeholder={t("What was checked, and what was done about it")}
                  description={t("Required when marking the event a false positive.")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} disabled={!event}>
                {t("Resolve")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
