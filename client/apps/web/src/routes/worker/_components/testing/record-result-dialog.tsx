import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { recordDotTestResult, type DotTestRow } from "@/lib/graphql/worker-drug-alcohol";
import { zodResolver } from "@hookform/resolvers/zod";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import { dotTestResultLabel, dotTestTypeLabel } from "@trenova/shared/lib/drug-alcohol";
import {
  dotTestResultFormSchema,
  dotTestResultSchema,
  type DOTTestResultFormValues,
} from "@trenova/shared/types/worker-drug-alcohol";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useTestingInvalidation } from "./use-testing-invalidation";

// Pending is what the record starts as, not something anyone reports back.
const RESULT_OPTIONS = dotTestResultSchema.options
  .filter((value) => value !== "Pending")
  .map((value) => ({ value, label: dotTestResultLabel(value) }));

export type RecordResultDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  test: DotTestRow | null;
};

function defaultsFor(test: DotTestRow | null): DOTTestResultFormValues {
  return {
    substance: (test?.substance ?? "Drug") as "Drug" | "Alcohol",
    result: "Negative",
    resultAt: getTodayDate(),
    labName: test?.labName ?? null,
    mroName: test?.mroName ?? null,
    mroVerifiedAt: null,
    alcoholConcentration: null,
    notes: null,
  };
}

export function RecordResultDialog({
  open,
  onOpenChange,
  workerId,
  test,
}: RecordResultDialogProps) {
  const t = useT();

  const invalidate = useTestingInvalidation(workerId);
  const form = useForm<DOTTestResultFormValues>({
    resolver: zodResolver(dotTestResultFormSchema) as Resolver<DOTTestResultFormValues>,
    defaultValues: defaultsFor(test),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(test));
  }, [open, test, reset]);

  const isAlcohol = test?.substance === "Alcohol";
  const [concentration] = useWatch({ control, name: ["alcoholConcentration"] });
  const reading = Number(concentration);
  const readingIsViolation = Number.isFinite(reading) && reading >= 0.04;

  const { mutateAsync, isPending } = useApiMutation<
    DotTestRow,
    DOTTestResultFormValues,
    unknown,
    DOTTestResultFormValues
  >({
    form,
    resourceName: "Result",
    mutationFn: (values) => {
      if (!test) throw new Error("No test selected");
      return recordDotTestResult({
        testId: test.id,
        result: values.result,
        resultAt: values.resultAt ?? undefined,
        labName: values.labName ?? undefined,
        mroName: isAlcohol ? undefined : (values.mroName ?? undefined),
        mroVerifiedAt: isAlcohol ? undefined : (values.mroVerifiedAt ?? undefined),
        alcoholConcentration: isAlcohol ? (values.alcoholConcentration ?? undefined) : undefined,
        notes: values.notes ?? undefined,
      });
    },
    onSuccess: (saved) => {
      const violation = ["Positive", "Refusal", "Adulterated", "Substituted"].includes(
        saved.result,
      );
      toast.success(t("Result recorded"), {
        description: violation
          ? "A violation has been opened. The driver is prohibited from safety-sensitive duty until the return-to-duty process is complete."
          : `Filed as ${dotTestResultLabel(saved.result).toLowerCase()}.`,
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{t("Record the result")}</DialogTitle>
          <DialogDescription>
            {test
              ? `${dotTestTypeLabel(test.testType)} · ${
                  test.substance === "Alcohol" ? "Alcohol" : "Controlled substances"
                }`
              : null}
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
              {isAlcohol ? (
                <>
                  <FormControl>
                    <InputField<DOTTestResultFormValues>
                      control={control}
                      name="alcoholConcentration"
                      label={t("Concentration")}
                      placeholder="0.000"
                      description={t(
                        "The breath alcohol concentration from the confirmation test; it decides the result on its own.",
                      )}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <Alert variant={readingIsViolation ? "destructive" : "default"}>
                      <AlertDescription>
                        {readingIsViolation
                          ? t(
                              "0.04 and above is a violation (49 CFR 382.201). The result will be filed as positive whatever is chosen below.",
                            )
                          : t(
                              "0.02 to 0.039 takes the driver off duty for 24 hours but is not a violation (49 CFR 382.505).",
                            )}
                      </AlertDescription>
                    </Alert>
                  </FormControl>
                </>
              ) : null}

              <FormControl>
                <SelectField<DOTTestResultFormValues>
                  control={control}
                  name="result"
                  label={t("Result")}
                  options={RESULT_OPTIONS}
                  placeholder={t("Pick a result")}
                  description={t(
                    "A positive, refusal, adulterated or substituted result opens a violation and prohibits the driver at once.",
                  )}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<DOTTestResultFormValues>
                  control={control}
                  name="resultAt"
                  label={t("Reported on")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t(
                    "When the laboratory or MRO reported the result; it also fills the collection date if none was recorded.",
                  )}
                />
              </FormControl>

              {isAlcohol ? null : (
                <>
                  <FormControl>
                    <InputField<DOTTestResultFormValues>
                      control={control}
                      name="labName"
                      label={t("Laboratory")}
                      placeholder={t("HHS-certified laboratory")}
                      description={t("The HHS-certified laboratory that analysed the specimen.")}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField<DOTTestResultFormValues>
                      control={control}
                      name="mroName"
                      label={t("Medical review officer")}
                      placeholder={t("Name of the MRO")}
                      description={t(
                        "The medical review officer who verified the laboratory result.",
                      )}
                    />
                  </FormControl>
                  <FormControl>
                    <AutoCompleteDateField<DOTTestResultFormValues>
                      control={control}
                      name="mroVerifiedAt"
                      label={t("MRO verified on")}
                      placeholder={t("MM/DD/YYYY")}
                      description={t(
                        "When the MRO verified the result; a drug result is not final until then.",
                      )}
                    />
                  </FormControl>
                </>
              )}

              <FormControl cols="full">
                <TextareaField<DOTTestResultFormValues>
                  control={control}
                  name="notes"
                  placeholder={t("e.g. Split specimen requested")}
                  description={t("Internal notes kept with the test record.")}
                  label={t("Notes")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {t("Record result")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
