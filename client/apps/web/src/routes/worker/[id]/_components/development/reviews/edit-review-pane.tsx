import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSingleFlight } from "@/hooks/use-single-flight";
import {
  updatePerformanceReview,
  type PerformanceReviewRow,
} from "@/lib/graphql/performance-review";
import { zodResolver } from "@hookform/resolvers/zod";
import { PlusIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  computeReviewScore,
  REVIEW_GOAL_STATUS_LABELS,
  reviewDraftFormSchema,
  reviewGoalStatusSchema,
  type ReviewDraftFormValues,
} from "@trenova/shared/types/performance-review";
import { useEffect, useRef, useState } from "react";
import {
  Controller,
  FormProvider,
  useFieldArray,
  useForm,
  useFormContext,
  useWatch,
  type Resolver,
} from "react-hook-form";
import { RatingScaleInput, scoreWord } from "./rating-scale";
import { OverallScore } from "./review-card";
import { useReviewInvalidation } from "./use-review-invalidation";

function draftDefaults(review: PerformanceReviewRow | null): ReviewDraftFormValues {
  return {
    title: review?.title ?? "",
    periodStart: review?.periodStart ?? 0,
    periodEnd: review?.periodEnd ?? 0,
    ratings: (review?.ratings ?? []).map((rating) => ({
      key: rating.key,
      label: rating.label,
      weight: rating.weight,
      score: rating.score ?? null,
      comment: rating.comment ?? null,
    })),
    summary: review?.summary ?? null,
    strengths: review?.strengths ?? null,
    improvements: review?.improvements ?? null,
    goals: (review?.goals ?? []).map((goal) => ({
      id: goal.id,
      title: goal.title,
      dueAt: goal.dueAt ?? null,
      status: goal.status,
    })),
  };
}

export type EditReviewPaneProps = {
  /** The draft being edited; the pane is open while it is set. */
  review: PerformanceReviewRow | null;
  workerId: string;
  onClose: () => void;
};

/**
 * Edit a draft review beside it: each criterion set by clicking its bar on the five-step
 * scale, the overall score updating as it goes, then the summary and goals.
 */
export function EditReviewPane({ review: requested, workerId, onClose }: EditReviewPaneProps) {
  const t = useT();
  const invalidate = useReviewInvalidation(workerId);
  const [held, setHeld] = useState(requested);
  if (requested !== null && requested !== held) setHeld(requested);
  const review = requested ?? held;

  const form = useForm<ReviewDraftFormValues>({
    resolver: zodResolver(reviewDraftFormSchema) as Resolver<ReviewDraftFormValues>,
    defaultValues: draftDefaults(review),
  });
  const { control, handleSubmit, reset } = form;

  const loadedFor = useRef<string | null>(null);
  useEffect(() => {
    const key = requested ? `${requested.id}:${requested.version}` : null;
    if (key !== null && loadedFor.current !== key) reset(draftDefaults(requested));
    loadedFor.current = key;
  }, [requested, reset]);

  const save = useApiMutation<
    PerformanceReviewRow,
    ReviewDraftFormValues,
    unknown,
    ReviewDraftFormValues
  >({
    form,
    resourceName: "Review",
    scope: { id: `worker-review-edit:${review?.id ?? "none"}` },
    mutationFn: (values) => {
      if (!review) throw new Error("No review to edit");
      return updatePerformanceReview({
        id: review.id,
        title: values.title,
        periodStart: values.periodStart,
        periodEnd: values.periodEnd,
        ratings: values.ratings.map((rating) => ({
          key: rating.key,
          score: rating.score ?? undefined,
          comment: rating.comment ?? undefined,
        })),
        summary: values.summary ?? undefined,
        strengths: values.strengths ?? undefined,
        improvements: values.improvements ?? undefined,
        goals: values.goals.map((goal) => ({
          id: goal.id ?? undefined,
          title: goal.title,
          dueAt: goal.dueAt ?? undefined,
          status: goal.status,
        })),
        version: review.version,
      });
    },
    onSuccess: () => {
      void invalidate();
      onClose();
    },
  });
  const submit = useSingleFlight((values: ReviewDraftFormValues) => save.mutateAsync(values));

  return (
    <WorkPane
      open={requested !== null}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={review ? t("Edit {0}", review.title) : t("Edit review")}
      description={t("Click a bar to set the score; nothing reaches the worker until you submit.")}
      contentKey={review?.id}
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
            <LiveOverall />
            <section aria-label={t("Ratings")} className="flex flex-col gap-3.5">
              <h4 className="text-xs font-medium text-foreground-subtle">{t("Ratings")}</h4>
              <RatingRows />
            </section>
            <FormGroup cols={1}>
              <FormControl>
                <InputField<ReviewDraftFormValues>
                  control={control}
                  name="title"
                  label={t("Title")}
                  required
                  maxLength={120}
                />
              </FormControl>
              <FormControl>
                <TextareaField<ReviewDraftFormValues>
                  control={control}
                  name="summary"
                  label={t("Summary")}
                  placeholder={t("The overall picture, in the words you would use to their face")}
                  maxLength={4000}
                />
              </FormControl>
              <FormControl>
                <TextareaField<ReviewDraftFormValues>
                  control={control}
                  name="strengths"
                  label={t("Strengths")}
                  placeholder={t("e.g. Clean inspections and on-time deliveries all year")}
                  maxLength={4000}
                />
              </FormControl>
              <FormControl>
                <TextareaField<ReviewDraftFormValues>
                  control={control}
                  name="improvements"
                  label={t("Where to improve")}
                  placeholder={t("e.g. Log fuel receipts the same day")}
                  maxLength={4000}
                />
              </FormControl>
            </FormGroup>
            <GoalRows />
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={submit.pending} disabled={!review}>
              {t("Save draft")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}

/** The overall score as the ratings stand now, weighted the way the server weighs them. */
function LiveOverall() {
  const t = useT();
  const { control } = useFormContext<ReviewDraftFormValues>();
  const ratings = useWatch({ control, name: "ratings" }) ?? [];
  const score = computeReviewScore(
    ratings.map((rating) => ({ weight: rating.weight ?? 0, score: rating.score ?? null })),
  );
  const rated = ratings.filter((rating) => rating.score != null).length;
  return (
    <div
      data-testid="review-overall"
      className="flex items-baseline gap-2 rounded-md border border-border-subtle bg-sunken px-3 py-2.5"
    >
      <OverallScore score={score} />
      <span className="ml-auto text-xs text-muted-foreground tabular-nums">
        {t("{0} of {1} rated", rated, ratings.length)}
      </span>
    </div>
  );
}

function RatingRows() {
  const t = useT();
  const { control } = useFormContext<ReviewDraftFormValues>();
  const { fields } = useFieldArray({ control, name: "ratings", keyName: "fieldKey" });
  if (fields.length === 0) {
    return <p className="text-xs text-muted-foreground">{t("This template rates nothing.")}</p>;
  }
  return (
    <>
      {fields.map((field, index) => (
        <div key={field.fieldKey} className="flex flex-col gap-1.5">
          <Controller
            control={control}
            name={`ratings.${index}.score`}
            render={({ field: score }) => (
              <div className="flex items-center gap-3">
                <span className="min-w-0 flex-1 text-sm font-medium">{field.label}</span>
                <RatingScaleInput
                  value={score.value ?? null}
                  label={field.label}
                  onChange={score.onChange}
                />
                <span className="w-22 text-right text-xs text-muted-foreground">
                  {scoreWord(score.value) ?? t("Not rated yet")}
                </span>
              </div>
            )}
          />
          <TextareaField<ReviewDraftFormValues>
            control={control}
            name={`ratings.${index}.comment`}
            label={t("Comment")}
            placeholder={t("What you saw, in a line")}
            maxLength={2000}
            rows={1}
          />
        </div>
      ))}
    </>
  );
}

const GOAL_STATUS_OPTIONS = reviewGoalStatusSchema.options.map((value) => ({
  value,
  label: REVIEW_GOAL_STATUS_LABELS[value],
}));

function GoalRows() {
  const t = useT();
  const { control } = useFormContext<ReviewDraftFormValues>();
  const goals = useFieldArray({ control, name: "goals", keyName: "fieldKey" });
  return (
    <section aria-label={t("Goals")} className="flex flex-col gap-2 border-t pt-4">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-medium text-foreground-subtle">{t("Goals")}</h4>
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={() => goals.append({ id: null, title: "", dueAt: null, status: "Open" })}
        >
          <PlusIcon className="size-3" />
          {t("Add goal")}
        </Button>
      </div>
      {goals.fields.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t("No goals set for the next period.")}</p>
      ) : null}
      {goals.fields.map((field, index) => (
        <div
          key={field.fieldKey}
          className="grid animate-expand-in grid-cols-[minmax(0,1fr)_auto] gap-2 rounded-md border border-border-subtle p-2 motion-reduce:animate-none"
        >
          <InputField<ReviewDraftFormValues>
            control={control}
            name={`goals.${index}.title`}
            label={t("Goal")}
            placeholder={t("e.g. Log fuel receipts the same day")}
          />
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            className="mt-5"
            aria-label={t("Remove goal {0}", index + 1)}
            onClick={() => goals.remove(index)}
          >
            <Trash01Icon className="size-3.5" />
          </Button>
          <div className="col-span-full grid grid-cols-2 gap-2">
            <AutoCompleteDateField<ReviewDraftFormValues>
              control={control}
              name={`goals.${index}.dueAt`}
              label={t("By")}
              placeholder={t("No date")}
            />
            <SelectField<ReviewDraftFormValues>
              control={control}
              name={`goals.${index}.status`}
              label={t("Status")}
              options={GOAL_STATUS_OPTIONS}
              clearable={false}
            />
          </div>
        </div>
      ))}
    </section>
  );
}
