import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCheckCallLabels } from "@/hooks/use-check-call-labels";
import type { LoggableMove } from "@/lib/check-calls";
import { logCheckCall } from "@/lib/graphql/check-calls";
import { shipment } from "@/lib/queries/shipment";
import {
  CHECK_CALL_LIMITS,
  checkCallKindSchema,
  emptyCheckCallForm,
  isStopVisit,
  logCheckCallFormSchema,
  toLogCheckCallInput,
  type LogCheckCallFormValues,
} from "@/types/check-call";
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
import { useT } from "@trenova/shared/i18n/use-t";
import type { SelectOption } from "@trenova/shared/types/fields";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

export type CheckCallStopOption = {
  id: string;
  moveId: string;
  label: string;
};

type LogCheckCallDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  shipmentId: string;
  moves: LoggableMove[];
  stops: CheckCallStopOption[];
  defaultMoveId: string;
};

export function LogCheckCallDialog({
  open,
  onOpenChange,
  shipmentId,
  moves,
  stops,
  defaultMoveId,
}: LogCheckCallDialogProps) {
  const t = useT();
  const labels = useCheckCallLabels();
  const queryClient = useQueryClient();

  const form = useForm<LogCheckCallFormValues>({
    resolver: zodResolver(logCheckCallFormSchema),
    defaultValues: emptyCheckCallForm(defaultMoveId),
  });
  const {
    control,
    handleSubmit,
    reset,
    setValue,
    formState: { isSubmitting },
  } = form;

  useEffect(() => {
    if (open) {
      reset(emptyCheckCallForm(defaultMoveId));
    }
  }, [open, defaultMoveId, reset]);

  const moveId = useWatch({ control, name: "shipmentMoveId" });
  const kind = useWatch({ control, name: "kind" });
  const delayed = useWatch({ control, name: "delayed" });
  const visit = isStopVisit(kind);

  const moveOptions = useMemo<SelectOption[]>(
    () => moves.map((move) => ({ value: move.id, label: move.label })),
    [moves],
  );
  const stopOptions = useMemo<SelectOption[]>(
    () =>
      stops
        .filter((stop) => stop.moveId === moveId)
        .map((stop) => ({ value: stop.id, label: stop.label })),
    [stops, moveId],
  );
  const kindOptions = useMemo(
    () => checkCallKindSchema.options.map((value) => ({ value, label: labels.kind(value) })),
    [labels],
  );

  useEffect(() => {
    setValue("stopId", null);
  }, [moveId, setValue]);

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: LogCheckCallFormValues) => logCheckCall(toLogCheckCallInput(values)),
    resourceName: "Check call",
    form,
    onSuccess: (result) => {
      void queryClient.invalidateQueries({
        queryKey: shipment.checkCalls(shipmentId).queryKey,
      });
      if (isStopVisit(result.checkCall.kind)) {
        void queryClient.invalidateQueries({
          queryKey: shipment.trackingEvents(shipmentId).queryKey,
        });
        void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      }
      if (result.outcome === "Refused" || result.outcome === "Pending") {
        toast.warning(t("Check call logged"), {
          description:
            result.outcomeReason ?? t("The stop's times were not changed by this report."),
        });
        return;
      }
      toast.success(t("Check call logged"));
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const onSubmit = useCallback(
    async (values: LogCheckCallFormValues) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && handleClose()}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Log check call")}</DialogTitle>
          <DialogDescription>
            {t(
              "Record what the carrier or driver told you. An arrival or departure also records the stop's actual time.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.stopPropagation();
              void handleSubmit(onSubmit)(event);
            }}
          >
            <FormGroup cols={2}>
              {moves.length > 1 ? (
                <FormControl cols="full">
                  <SelectField
                    control={control}
                    name="shipmentMoveId"
                    label={t("Move")}
                    options={moveOptions}
                    rules={{ required: true }}
                  />
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <SegmentedField
                  control={control}
                  name="kind"
                  label={t("What happened")}
                  options={kindOptions}
                />
              </FormControl>
              {visit ? (
                <FormControl cols="full">
                  <SelectField
                    control={control}
                    name="stopId"
                    label={t("Stop")}
                    options={stopOptions}
                    placeholder={t("Choose the stop")}
                    rules={{ required: true }}
                  />
                </FormControl>
              ) : null}
              <FormControl cols="full">
                <AutoCompleteDateTimeField
                  control={control}
                  name="reportedAt"
                  label={t("When")}
                  placeholder={t("Now")}
                  description={t("Leave empty to record it as now.")}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="city"
                  label={t("City")}
                  maxLength={CHECK_CALL_LIMITS.city}
                  placeholder={t("Where the truck is")}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="stateCode"
                  label={t("State")}
                  maxLength={CHECK_CALL_LIMITS.stateCode}
                  placeholder="TX"
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField
                  control={control}
                  name="delayed"
                  label={t("Running late")}
                  description={t("The caller said the load will miss its appointment.")}
                  outlined
                />
              </FormControl>
              {delayed ? (
                <FormControl cols="full">
                  <InputField
                    control={control}
                    name="delayReason"
                    label={t("Why")}
                    maxLength={CHECK_CALL_LIMITS.delayReason}
                    rules={{ required: true }}
                  />
                </FormControl>
              ) : null}
              <FormControl>
                <AutoCompleteDateTimeField
                  control={control}
                  name="estimatedArrival"
                  label={t("Expected at next stop")}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="contactName"
                  label={t("Spoke to")}
                  maxLength={CHECK_CALL_LIMITS.contactName}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField
                  control={control}
                  name="note"
                  label={t("Note")}
                  maxLength={CHECK_CALL_LIMITS.note}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={handleClose}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isSubmitting} loadingText={t("Logging...")}>
                {t("Log check call")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
