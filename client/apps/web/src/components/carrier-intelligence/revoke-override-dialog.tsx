import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  CARRIER_INTEL_NOTE_MAX_LENGTH,
  revokeOverrideFormSchema,
  type RevokeOverrideFormValues,
} from "@/lib/carrier-intelligence";
import {
  revokeCarrierIntelOverride,
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
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

export type RevokeOverrideDialogProps = {
  override: CarrierIntelOverride | null;
  ruleLabel?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onRevoked?: (override: CarrierIntelOverride) => void;
};

const DEFAULT_VALUES: RevokeOverrideFormValues = { reason: "" };

export function RevokeOverrideDialog({
  override,
  ruleLabel,
  open,
  onOpenChange,
  onRevoked,
}: RevokeOverrideDialogProps) {
  const t = useT();

  const form = useForm<RevokeOverrideFormValues>({
    resolver: zodResolver(revokeOverrideFormSchema) as Resolver<RevokeOverrideFormValues>,
    defaultValues: DEFAULT_VALUES,
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) {
      reset(DEFAULT_VALUES);
    }
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    CarrierIntelOverride,
    RevokeOverrideFormValues,
    unknown,
    RevokeOverrideFormValues
  >({
    form,
    resourceName: "Carrier intelligence override",
    mutationFn: (values) => {
      if (!override) {
        throw new Error("No override selected");
      }
      return revokeCarrierIntelOverride(override.id, values.reason);
    },
    onSuccess: (revoked) => {
      toast.success(t("Override revoked"));
      onRevoked?.(revoked);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Revoke override")}</DialogTitle>
          <DialogDescription>
            {t(
              "The blocking rule applies again immediately. Open tenders are re-checked the next time the gate runs.",
            )}
          </DialogDescription>
        </DialogHeader>
        {override ? (
          <div className="flex flex-col gap-1 border-y py-3 text-sm">
            <span className="font-medium">{ruleLabel ?? override.ruleCode}</span>
            <span className="text-muted-foreground text-xs">
              {t(
                "Granted {0}, expires {1}",
                formatUnixDateTimeMedium(override.grantedAt),
                formatUnixDateTimeMedium(override.expiresAt),
              )}
            </span>
            <span>{override.reason}</span>
          </div>
        ) : null}
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
                <TextareaField<RevokeOverrideFormValues>
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("Why the override is no longer warranted")}
                  rules={{ required: true }}
                  maxLength={CARRIER_INTEL_NOTE_MAX_LENGTH}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                variant="destructive"
                isLoading={isPending}
                disabled={!override}
              >
                {t("Revoke")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
