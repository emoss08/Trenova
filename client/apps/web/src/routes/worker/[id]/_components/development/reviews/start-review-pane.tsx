import { PerformanceReviewTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import {
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
import { useEffect, useRef } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { useReviewInvalidation } from "./use-review-invalidation";

const YEAR = 365 * 86_400;

function startDefaults(templateId: string): CreateReviewFormValues {
  const today = getTodayDate();
  return { templateId, title: null, periodStart: today - YEAR, periodEnd: today };
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
  const invalidate = useReviewInvalidation(workerId);
  const form = useForm<CreateReviewFormValues>({
    resolver: zodResolver(createReviewFormSchema) as Resolver<CreateReviewFormValues>,
    defaultValues: startDefaults(""),
  });
  const { control, handleSubmit, reset, getValues } = form;

  // The provider orders default templates first, so the first option is the one a reviewer
  // would have picked anyway.
  const { option: preferred, isLoading: templatesLoading } = useFirstSelectOption(
    "PERFORMANCE_REVIEW_TEMPLATE",
    { enabled: open },
  );
  const noTemplates = !templatesLoading && preferred == null;

  const openedFor = useRef<boolean>(false);
  useEffect(() => {
    if (open && !openedFor.current) reset(startDefaults(preferred?.id ?? ""));
    if (open && preferred && !getValues("templateId")) {
      reset({ ...getValues(), templateId: preferred.id }, { keepDirtyValues: true });
    }
    openedFor.current = open;
  }, [open, preferred, reset, getValues]);

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
    onSuccess: (review) => {
      void invalidate();
      onStarted(review);
    },
  });
  const submit = useSingleFlight((values: CreateReviewFormValues) => save.mutateAsync(values));

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
              {t("Start review")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
