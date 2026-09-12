import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  recordSafetyViolation,
  SAFETY_VIOLATIONS_KEY,
  updateSafetyViolation,
  type SafetyViolationRow,
} from "@/lib/graphql/fleet-safety";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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
import { CSA_BASIC_ORDER, csaBasicHint, csaBasicLabel } from "@trenova/shared/lib/csa";
import {
  safetyViolationFormSchema,
  type SafetyViolationFormValues,
} from "@trenova/shared/types/fleet-safety";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const BASIC_OPTIONS = CSA_BASIC_ORDER.map((value) => ({
  value,
  label: csaBasicLabel(value),
  description: csaBasicHint(value),
}));

export type ViolationDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  safetyEventId: string;
  suggestedBasic: string | null;
  violation: SafetyViolationRow | null;
};

function defaultsFor(
  violation: SafetyViolationRow | null,
  suggestedBasic: string | null,
): SafetyViolationFormValues {
  if (!violation) {
    return {
      basic: (suggestedBasic ?? "VehicleMaintenance") as SafetyViolationFormValues["basic"],
      code: null,
      description: "",
      severityWeight: 1,
      outOfService: false,
    };
  }
  return {
    basic: violation.basic as SafetyViolationFormValues["basic"],
    code: violation.code ?? null,
    description: violation.description,
    severityWeight: violation.severityWeight,
    outOfService: violation.outOfService,
  };
}

export function ViolationDialog({
  open,
  onOpenChange,
  safetyEventId,
  suggestedBasic,
  violation,
}: ViolationDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const isEdit = Boolean(violation);
  const form = useForm<SafetyViolationFormValues>({
    resolver: zodResolver(safetyViolationFormSchema) as Resolver<SafetyViolationFormValues>,
    defaultValues: defaultsFor(violation, suggestedBasic),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(violation, suggestedBasic));
  }, [open, violation, suggestedBasic, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    SafetyViolationFormValues,
    unknown,
    SafetyViolationFormValues
  >({
    form,
    resourceName: "Violation",
    mutationFn: (values) => {
      const shared = {
        basic: values.basic,
        code: values.code ?? undefined,
        description: values.description,
        severityWeight: values.severityWeight,
        outOfService: values.outOfService,
      };
      return violation
        ? updateSafetyViolation({ ...shared, id: violation.id, version: violation.version })
        : recordSafetyViolation({ ...shared, safetyEventId });
    },
    onSuccess: () => {
      toast.success(isEdit ? "Violation corrected" : "Violation cited");
      void queryClient.invalidateQueries({ queryKey: [SAFETY_VIOLATIONS_KEY, safetyEventId] });
      void queryClient.invalidateQueries({ queryKey: ["fleet-safety"] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("Correct the violation") : t("Cite a violation")}</DialogTitle>
          <DialogDescription>
            {t(
              "One row per violation. A single inspection routinely cites several in different BASICs, and only the rows can say which.",
            )}
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
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <SelectField<SafetyViolationFormValues>
                  control={control}
                  name="basic"
                  label="BASIC"
                  options={BASIC_OPTIONS}
                  placeholder={t("Pick a BASIC")}
                  description={t(
                    "The CSA category the violation is scored under; each BASIC has its own measure and threshold.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <InputField<SafetyViolationFormValues>
                  control={control}
                  name="code"
                  label={t("Violation code")}
                  placeholder={t("e.g. 395.8")}
                  description={t("From the inspection report, so it can be looked up.")}
                />
              </FormControl>
              <FormControl>
                <NumberField<SafetyViolationFormValues>
                  control={control}
                  name="severityWeight"
                  placeholder={t("e.g. 5")}
                  label={t("Severity weight")}
                  rules={{ required: true }}
                  description={t("The FMCSA's published weight, 1 to 10.")}
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<SafetyViolationFormValues>
                  control={control}
                  name="description"
                  label={t("What was cited")}
                  placeholder={t("e.g. Brake out of adjustment on two axles")}
                  description={t("The violation as written on the inspection report.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <SwitchField<SafetyViolationFormValues>
                  control={control}
                  name="outOfService"
                  label={t("Out of service")}
                  description={t(
                    "Adds two to the weight before the recency multiplier, the way the FMCSA scores it.",
                  )}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? t("Save") : t("Cite")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
