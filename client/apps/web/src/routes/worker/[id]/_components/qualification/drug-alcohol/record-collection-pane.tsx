import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { recordDotTest, type DotTestRow } from "@/lib/graphql/worker-drug-alcohol";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { getTodayDate } from "@trenova/shared/lib/date";
import { dotTestTypeLabel } from "@trenova/shared/lib/drug-alcohol";
import {
  dotTestFormSchema,
  dotTestSubstanceSchema,
  dotTestTypeSchema,
  type DOTTestFormValues,
} from "@trenova/shared/types/worker-drug-alcohol";
import { useEffect, useRef } from "react";
import { substanceLabel } from "./drug-alcohol-labels";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { useTestingInvalidation } from "./use-testing-invalidation";

const TYPE_OPTIONS = dotTestTypeSchema.options.map((value) => ({
  value,
  label: dotTestTypeLabel(value),
}));
export type RecordCollectionPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** Set when the collection answers a random selection. */
  drawEntryId?: string | null;
  defaultSubstance?: "Drug" | "Alcohol";
  onSaved?: (testId: string) => void;
};

function emptyValues(drawEntryId: string | null, substance: "Drug" | "Alcohol"): DOTTestFormValues {
  return {
    testType: drawEntryId ? "Random" : "PreEmployment",
    substance,
    isDot: true,
    reason: null,
    scheduledAt: getTodayDate(),
    collectedAt: null,
    collectionSite: null,
    collectorName: null,
    specimenId: null,
    notes: null,
    drawEntryId,
  };
}

export function RecordCollectionPane({
  open,
  onOpenChange,
  workerId,
  drawEntryId = null,
  defaultSubstance = "Drug",
  onSaved,
}: RecordCollectionPaneProps) {
  const t = useT();
  const substanceOptions = dotTestSubstanceSchema.options.map((value) => ({
    value,
    label: substanceLabel(t, value),
  }));

  const invalidate = useTestingInvalidation(workerId);
  const form = useForm<DOTTestFormValues>({
    resolver: zodResolver(dotTestFormSchema) as Resolver<DOTTestFormValues>,
    defaultValues: emptyValues(drawEntryId, defaultSubstance),
  });
  const { control, handleSubmit, reset } = form;

  const loadedFor = useRef<string | null>(null);
  useEffect(() => {
    const key = open ? `${drawEntryId ?? ""}:${defaultSubstance}` : null;
    if (key === loadedFor.current) return;
    loadedFor.current = key;
    if (key !== null) reset(emptyValues(drawEntryId, defaultSubstance));
  }, [open, drawEntryId, defaultSubstance, reset]);

  const [testType] = useWatch({ control, name: ["testType"] });
  const needsReason = testType === "ReasonableSuspicion" || testType === "PostAccident";

  const { mutateAsync, isPending } = useApiMutation<
    DotTestRow,
    DOTTestFormValues,
    unknown,
    DOTTestFormValues
  >({
    form,
    resourceName: "Test",
    mutationFn: (values) =>
      recordDotTest({
        workerId,
        testType: values.testType,
        substance: values.substance,
        // A collection date on the form means the specimen has been taken; the
        // result comes later and moves the status on again.
        status: values.collectedAt ? "Collected" : "Scheduled",
        isDot: values.isDot,
        reason: values.reason ?? undefined,
        scheduledAt: values.scheduledAt ?? undefined,
        collectedAt: values.collectedAt ?? undefined,
        collectionSite: values.collectionSite ?? undefined,
        collectorName: values.collectorName ?? undefined,
        specimenId: values.specimenId ?? undefined,
        notes: values.notes ?? undefined,
        drawEntryId: values.drawEntryId ?? undefined,
      }),
    onSuccess: (saved) => {
      reset(form.getValues());
      void invalidate();
      onSaved?.(saved.id);
      onOpenChange(false);
    },
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={t("Record a collection")}
      description={t(
        "One record per substance analyzed. A collection covering both drug and alcohol is two records.",
      )}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form
          onSubmit={(submitEvent) => {
            submitEvent.preventDefault();
            submitEvent.stopPropagation();
            void handleSubmit(async (values) => {
              await mutateAsync(values);
            })(submitEvent);
          }}
        >
          <WorkPaneBody>
            <FormGroup cols={2}>
              <FormControl>
                <SelectField<DOTTestFormValues>
                  control={control}
                  name="testType"
                  label={t("Reason for the test")}
                  options={TYPE_OPTIONS}
                  placeholder={t("Pick a reason")}
                  description={t(
                    "Reasonable suspicion and post-accident tests must say what prompted them.",
                  )}
                  required
                />
              </FormControl>
              <FormControl>
                <SelectField<DOTTestFormValues>
                  control={control}
                  name="substance"
                  label={t("What is analyzed")}
                  options={substanceOptions}
                  placeholder={t("Pick a substance")}
                  description={t(
                    "Drug results go through an MRO; alcohol results are graded by the concentration.",
                  )}
                  required
                />
              </FormControl>

              {needsReason ? (
                <FormControl cols="full">
                  <TextareaField<DOTTestFormValues>
                    control={control}
                    name="reason"
                    label={t("What prompted it")}
                    placeholder={t(
                      "What the supervisor observed, or the accident that triggered the collection",
                    )}
                    description={t("Kept with the test as the documented basis for ordering it.")}
                    required
                    maxLength={2000}
                  />
                </FormControl>
              ) : null}

              <FormControl>
                <AutoCompleteDateField<DOTTestFormValues>
                  control={control}
                  name="scheduledAt"
                  label={t("Scheduled for")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t("When the driver is due at the collection site.")}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<DOTTestFormValues>
                  control={control}
                  name="collectedAt"
                  label={t("Collected on")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t(
                    "Entering a date marks the specimen as collected; leave it blank if it has not been taken yet.",
                  )}
                />
              </FormControl>
              <FormControl>
                <InputField<DOTTestFormValues>
                  control={control}
                  name="collectionSite"
                  label={t("Collection site")}
                  placeholder={t("e.g. Concentra, Joliet IL")}
                  description={t("Where the specimen was collected.")}
                />
              </FormControl>
              <FormControl>
                <InputField<DOTTestFormValues>
                  control={control}
                  name="collectorName"
                  label={t("Collector")}
                  placeholder={t("Name of the collector")}
                  description={t("The collector who took the specimen, as named on the CCF.")}
                />
              </FormControl>
              <FormControl>
                <InputField<DOTTestFormValues>
                  control={control}
                  name="specimenId"
                  label={t("Specimen ID")}
                  placeholder={t("CCF specimen number")}
                  description={t(
                    "The specimen ID from the custody and control form, for matching the laboratory report.",
                  )}
                />
              </FormControl>
              <FormControl>
                <SwitchField<DOTTestFormValues>
                  control={control}
                  name="isDot"
                  label={t("DOT test")}
                  description={t(
                    "Turn off for a company-policy test that is not made under 49 CFR 382.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<DOTTestFormValues>
                  control={control}
                  name="notes"
                  placeholder={t("e.g. Driver escorted to the site")}
                  description={t("Internal notes kept with the test record.")}
                  label={t("Notes")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {t("Record collection")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
