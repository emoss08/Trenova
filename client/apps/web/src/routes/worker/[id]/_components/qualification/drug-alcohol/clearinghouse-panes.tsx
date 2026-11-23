import { useT } from "@trenova/shared/i18n/use-t";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  completeClearinghouseQuery,
  recordClearinghouseQuery,
  type ClearinghouseQueryRow,
} from "@/lib/graphql/worker-drug-alcohol";
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
import { formatUnixDateMedium, getTodayDate } from "@trenova/shared/lib/date";
import {
  clearinghouseQueryTypeLabel,
  clearinghouseResultLabel,
  clearinghouseResultTone,
} from "@trenova/shared/lib/drug-alcohol";
import {
  clearinghouseAnswerFormSchema,
  clearinghouseQueryFormSchema,
  clearinghouseQueryTypeSchema,
  clearinghouseResultSchema,
  type ClearinghouseAnswerFormValues,
  type ClearinghouseQueryFormValues,
} from "@trenova/shared/types/worker-drug-alcohol";
import { SubjectCard } from "../subject-card";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { useWorkerInvalidation } from "../../use-worker-invalidation";

const TYPE_OPTIONS = clearinghouseQueryTypeSchema.options.map((value) => ({
  value,
  label: clearinghouseQueryTypeLabel(value),
}));
// Pending is the state a query starts in, not an answer anyone reports.
const RESULT_OPTIONS = clearinghouseResultSchema.options
  .filter((value) => value !== "Pending")
  .map((value) => ({ value, label: clearinghouseResultLabel(value) }));

export type LogQueryPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  onSaved?: (queryId: string) => void;
};

function emptyQuery(): ClearinghouseQueryFormValues {
  return {
    queryType: "AnnualLimited",
    requestedAt: getTodayDate(),
    consentObtainedAt: null,
    consentExpiresAt: null,
    reference: null,
    notes: null,
  };
}

export function LogQueryPane({ open, onOpenChange, workerId, onSaved }: LogQueryPaneProps) {
  const t = useT();

  const invalidate = useWorkerInvalidation(workerId, "testing");
  const form = useForm<ClearinghouseQueryFormValues>({
    resolver: zodResolver(clearinghouseQueryFormSchema) as Resolver<ClearinghouseQueryFormValues>,
    defaultValues: emptyQuery(),
  });
  const { control, handleSubmit, reset } = form;

  const [queryType] = useWatch({ control, name: ["queryType"] });
  const isFull = queryType === "PreEmploymentFull" || queryType === "Full";

  const { mutateAsync, isPending } = useApiMutation<
    ClearinghouseQueryRow,
    ClearinghouseQueryFormValues,
    unknown,
    ClearinghouseQueryFormValues
  >({
    form,
    resourceName: "Clearinghouse query",
    mutationFn: (values) =>
      recordClearinghouseQuery({
        workerId,
        queryType: values.queryType,
        requestedAt: values.requestedAt,
        consentObtainedAt: values.consentObtainedAt ?? undefined,
        consentExpiresAt: values.consentExpiresAt ?? undefined,
        reference: values.reference ?? undefined,
        notes: values.notes ?? undefined,
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
      title={t("Log query")}
      description={t(
        "The 12-month clock restarts when the answer comes back clear, not when you log the query.",
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
                <SelectField<ClearinghouseQueryFormValues>
                  control={control}
                  name="queryType"
                  label={t("Query")}
                  options={TYPE_OPTIONS}
                  placeholder={t("Pick a query type")}
                  description={t(
                    "A full query needs the driver's specific consent; a limited query runs on the general consent on file.",
                  )}
                  required
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseQueryFormValues>
                  control={control}
                  name="requestedAt"
                  label={t("Requested on")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t("The date the query was submitted to the Clearinghouse.")}
                  required
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseQueryFormValues>
                  control={control}
                  name="consentObtainedAt"
                  label={t("Consent obtained")}
                  placeholder={t("MM/DD/YYYY")}
                  required={isFull}
                  description={
                    isFull
                      ? t("A full query cannot be run without the driver's consent.")
                      : t("When the driver's general consent for limited queries was signed.")
                  }
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseQueryFormValues>
                  control={control}
                  name="consentExpiresAt"
                  label={t("Consent expires")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t(
                    "When the consent on file lapses; a new one is needed before the next query.",
                  )}
                />
              </FormControl>
              <FormControl>
                <InputField<ClearinghouseQueryFormValues>
                  control={control}
                  name="reference"
                  label={t("Reference")}
                  placeholder={t("Clearinghouse query reference")}
                  inputClassProps="font-mono"
                  description={t(
                    "The query ID the Clearinghouse assigned, for matching the answer when it comes back.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<ClearinghouseQueryFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("Anything an auditor should know about this query")}
                  description={t("Internal notes kept with the query.")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {t("Log query")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}

export type AnswerQueryPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  query: ClearinghouseQueryRow | null;
  onSaved?: (queryId: string) => void;
};

function emptyAnswer(): ClearinghouseAnswerFormValues {
  return {
    result: "NoViolations",
    completedAt: getTodayDate(),
    violationCount: 0,
    reference: null,
    notes: null,
  };
}

export function AnswerQueryPane({
  open,
  onOpenChange,
  workerId,
  query,
  onSaved,
}: AnswerQueryPaneProps) {
  const t = useT();

  const invalidate = useWorkerInvalidation(workerId, "testing");
  const form = useForm<ClearinghouseAnswerFormValues>({
    resolver: zodResolver(clearinghouseAnswerFormSchema) as Resolver<ClearinghouseAnswerFormValues>,
    defaultValues: emptyAnswer(),
  });
  const { control, handleSubmit, reset } = form;

  const [result] = useWatch({ control, name: ["result"] });
  const foundViolations = result === "ViolationsFound";

  const { mutateAsync, isPending } = useApiMutation<
    ClearinghouseQueryRow,
    ClearinghouseAnswerFormValues,
    unknown,
    ClearinghouseAnswerFormValues
  >({
    form,
    resourceName: "Clearinghouse answer",
    mutationFn: (values) => {
      if (!query) throw new Error("No query selected");
      return completeClearinghouseQuery({
        queryId: query.id,
        result: values.result,
        completedAt: values.completedAt,
        violationCount: foundViolations ? values.violationCount : 0,
        reference: values.reference ?? undefined,
        notes: values.notes ?? undefined,
      });
    },
    onSuccess: (saved) => {
      reset(form.getValues());
      void invalidate();
      onSaved?.(saved.id);
      onOpenChange(false);
    },
  });

  return (
    <WorkPane open={open} onOpenChange={onOpenChange} title={t("Record answer")}>
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
            {query ? (
              <SubjectCard
                title={clearinghouseQueryTypeLabel(query.queryType)}
                badge={{
                  label: clearinghouseResultLabel(query.result),
                  variant: clearinghouseResultTone(query.result),
                }}
                detail={[
                  t("Requested {0}", formatUnixDateMedium(query.requestedAt)),
                  query.reference,
                ]
                  .filter(Boolean)
                  .join(" · ")}
              />
            ) : null}
            <FormGroup cols={2}>
              <FormControl>
                <SelectField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="result"
                  label={t("Answer")}
                  options={RESULT_OPTIONS}
                  placeholder={t("Pick an answer")}
                  description={t(
                    "Violations found or consent denied prohibits the driver from safety-sensitive duty at once.",
                  )}
                  required
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="completedAt"
                  label={t("Answered on")}
                  placeholder={t("MM/DD/YYYY")}
                  description={t(
                    "When the Clearinghouse returned the answer; the next annual query is due twelve months on.",
                  )}
                  required
                />
              </FormControl>
              {foundViolations ? (
                <FormControl>
                  <NumberField<ClearinghouseAnswerFormValues>
                    control={control}
                    name="violationCount"
                    label={t("Violations returned")}
                    placeholder={t("e.g. 1")}
                    description={t("How many violations the Clearinghouse reported.")}
                    required
                  />
                </FormControl>
              ) : null}
              <FormControl>
                <InputField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="reference"
                  label={t("Reference")}
                  placeholder={t("Clearinghouse response reference")}
                  inputClassProps="font-mono"
                  description={t("The query ID on the Clearinghouse response.")}
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="notes"
                  label={t("Notes")}
                  placeholder={t("e.g. Driver notified and taken off the board")}
                  description={t("Internal notes kept with the answer.")}
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {t("Record answer")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
