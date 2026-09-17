import { useT } from "@trenova/shared/i18n/use-t";
import { DateField } from "@/components/fields/date-field/date-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  CARRIER_INTEL_NOTE_MAX_LENGTH,
  DEFAULT_OVERRIDE_DAYS,
  MAX_OVERRIDE_DAYS,
  SECONDS_PER_DAY,
  createOverrideFormSchema,
  type OverrideFormValues,
} from "@/lib/carrier-intelligence";
import {
  grantCarrierIntelOverride,
  type CarrierIntelFinding,
  type CarrierIntelOverride,
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
import { dateToUnixTimestamp, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useCallback, useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { FindingList } from "./finding-list";

export type GrantOverrideDialogProps = {
  carrierId: string;
  finding: CarrierIntelFinding | null;
  ruleLabel?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onGranted?: (override: CarrierIntelOverride) => void;
};

const PRESET_DAYS = [7, DEFAULT_OVERRIDE_DAYS, 30, MAX_OVERRIDE_DAYS] as const;

function defaultValues(): OverrideFormValues {
  return {
    reason: "",
    expiresAt: dateToUnixTimestamp(new Date()) + DEFAULT_OVERRIDE_DAYS * SECONDS_PER_DAY,
  };
}

const overrideResolver: Resolver<OverrideFormValues> = (values, context, options) =>
  (
    zodResolver(
      createOverrideFormSchema(dateToUnixTimestamp(new Date())),
    ) as Resolver<OverrideFormValues>
  )(values, context, options);

export function GrantOverrideDialog({
  carrierId,
  finding,
  ruleLabel,
  open,
  onOpenChange,
  onGranted,
}: GrantOverrideDialogProps) {
  const t = useT();

  const form = useForm<OverrideFormValues>({
    resolver: overrideResolver,
    defaultValues: defaultValues(),
  });
  const { control, handleSubmit, reset, setValue } = form;
  const expiresAt = useWatch({ control, name: "expiresAt" });

  useEffect(() => {
    if (open) {
      reset(defaultValues());
    }
  }, [open, reset]);

  const applyPreset = useCallback(
    (days: number) => {
      setValue("expiresAt", dateToUnixTimestamp(new Date()) + days * SECONDS_PER_DAY, {
        shouldDirty: true,
        shouldValidate: true,
      });
    },
    [setValue],
  );

  const { mutateAsync, isPending } = useApiMutation<
    CarrierIntelOverride,
    OverrideFormValues,
    unknown,
    OverrideFormValues
  >({
    form,
    resourceName: "Carrier intelligence override",
    mutationFn: (values) => {
      if (!finding) {
        throw new Error("No finding selected");
      }
      return grantCarrierIntelOverride({
        carrierId,
        ruleCode: finding.code,
        reason: values.reason.trim(),
        expiresAt: values.expiresAt,
      });
    },
    onSuccess: (override) => {
      toast.success(t("Override granted"), {
        description: t("Expires {0}", formatUnixDateTimeMedium(override.expiresAt)),
      });
      onGranted?.(override);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Grant override")}</DialogTitle>
          <DialogDescription>
            {t(
              "Lets this carrier through the blocking rule until the override expires. Overrides last at most {0} days and are recorded with your name and reason.",
              MAX_OVERRIDE_DAYS,
            )}
          </DialogDescription>
        </DialogHeader>
        {finding ? (
          <FindingList
            findings={[finding]}
            ruleLabels={ruleLabel ? { [finding.code]: ruleLabel } : undefined}
            emptyMessage={null}
            className="border-y"
          />
        ) : null}
        <FormProvider {...form}>
          <Form
            aria-label={t("Grant override")}
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1} className="pb-2">
              <FormControl>
                <TextareaField<OverrideFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("Why this carrier may be used despite the finding")}
                  rules={{ required: true }}
                  maxLength={CARRIER_INTEL_NOTE_MAX_LENGTH}
                />
              </FormControl>
              <FormControl>
                <DateField<OverrideFormValues>
                  control={control}
                  name="expiresAt"
                  label={t("Expires")}
                  description={
                    expiresAt
                      ? t("Override ends {0}", formatUnixDateTimeMedium(expiresAt))
                      : t("Choose when the override ends")
                  }
                  rules={{ required: true }}
                />
                <div className="flex flex-wrap gap-1 pt-1.5" role="group" aria-label={t("Presets")}>
                  {PRESET_DAYS.map((days) => (
                    <Button
                      key={days}
                      type="button"
                      size="xs"
                      variant="outline"
                      onClick={() => applyPreset(days)}
                    >
                      {t("{0} days", days)}
                    </Button>
                  ))}
                </div>
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} disabled={!finding}>
                {t("Grant override")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
