import { PerformanceReviewTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import {
  PaneFormLoading,
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import { useFirstSelectOption } from "@/hooks/use-select-option";
import {
  createPerformanceReview,
  type PerformanceReviewRow,
} from "@/lib/graphql/performance-review";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  createReviewFormSchema,
  type CreateReviewFormValues,
} from "@trenova/shared/types/performance-review";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { SECONDS_IN_DAY } from "@/lib/time-units";
import { useWorkerInvalidation } from "../../use-worker-invalidation";

/** The 365 days to today, the period an annual review covers unless changed. */
function startDefaults(templateId: string): CreateReviewFormValues {
  const today = getTodayDate();
  return { templateId, title: null, periodStart: today - 365 * SECONDS_IN_DAY, periodEnd: today };
}

export type StartReviewPaneProps = {
  open: boolean;
  workerId: string;
  workerName: string;
  /** Told the new draft, so the page can open it for editing. */
  onStarted: (review: PerformanceReviewRow) => void;
  onClose: () => void;
};

/**
 * Start a review beside the reviews. It opens as a draft only the office sees until it is
 * submitted to the worker; the template decides what gets rated.
 */
export function StartReviewPane({
  open,
  workerId,
  workerName,
  onStarted,
  onClose,
}: StartReviewPaneProps) {
  const t = useT();
  // The provider orders default templates first, so the first option is the one a reviewer
  // would have picked anyway. It is read before the form mounts, so it is the form's default.
  const { option: preferred, isLoading } = useFirstSelectOption("PERFORMANCE_REVIEW_TEMPLATE");

  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("Start a review")}
      description={t(
        "It opens as a draft only the office sees until you submit it to {0}.",
        workerName,
      )}
    >
      {isLoading ? (
        <PaneFormLoading fields={4} />
      ) : (
        <StartReviewForm
          workerId={workerId}
          templateId={preferred?.id ?? null}
          onStarted={onStarted}
        />
      )}
    </WorkPane>
  );
}

function StartReviewForm({
  workerId,
  templateId,
  onStarted,
}: {
  workerId: string;
  /** The default template; null when the organization has no active one. */
  templateId: string | null;
  onStarted: (review: PerformanceReviewRow) => void;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "reviews");
  const noTemplates = templateId === null;
  const form = useForm<CreateReviewFormValues>({
    resolver: zodResolver(createReviewFormSchema) as Resolver<CreateReviewFormValues>,
    defaultValues: startDefaults(templateId ?? ""),
  });
  const { control, handleSubmit } = form;

  const save = useApiMutation<
    PerformanceReviewRow,
    CreateReviewFormValues,
    unknown,
    CreateReviewFormValues
  >({
    form,
    resourceName: "Review",
    scope: { id: `worker-review-start:${workerId}` },
    mutationFn: (values) =>
      createPerformanceReview({
        workerId,
        templateId: values.templateId,
        title: values.title ?? undefined,
        periodStart: values.periodStart,
        periodEnd: values.periodEnd,
      }),
    onSuccess: async (review) => {
      await invalidate();
      onStarted(review);
    },
  });
  const submit = useSingleFlight((values: CreateReviewFormValues) => save.mutateAsync(values));

  return (
    <FormProvider {...form}>
      <WorkPaneDirtyTracker form={form} />
      <Form
        onSubmit={(event) => {
          event.preventDefault();
          event.stopPropagation();
          void handleSubmit((submitted) => submit.run(submitted))(event);
        }}
      >
        <WorkPaneBody>
          <FormGroup cols={2}>
            <FormControl cols="full">
              <PerformanceReviewTemplateAutocompleteField<CreateReviewFormValues>
                control={control}
                name="templateId"
                label={t("Template")}
                placeholder={t("Select a template")}
                required
                noResultsMessage={t(
                  "No active templates. Create one under Review Templates first.",
                )}
                description={
                  noTemplates
                    ? t("No active templates. Create one under Review Templates first.")
                    : t("Decides which criteria are rated and how they are weighted.")
                }
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField<CreateReviewFormValues>
                control={control}
                name="periodStart"
                label={t("Period from")}
                placeholder={t("A year ago")}
                required
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField<CreateReviewFormValues>
                control={control}
                name="periodEnd"
                label={t("Period to")}
                placeholder={t("Today")}
                required
                description={t("It also dates the default title.")}
              />
            </FormControl>
            <FormControl cols="full">
              <InputField<CreateReviewFormValues>
                control={control}
                name="title"
                label={t("Title")}
                placeholder={t("e.g. 2026 annual review")}
                maxLength={120}
                description={t("Leave blank to use the template name and the period end date.")}
              />
            </FormControl>
          </FormGroup>
        </WorkPaneBody>
        <WorkPaneFooter>
          <WorkPaneCancel />
          <Button type="submit" size="sm" isLoading={submit.pending} disabled={noTemplates}>
            {t("Start a review")}
          </Button>
        </WorkPaneFooter>
      </Form>
    </FormProvider>
  );
}
