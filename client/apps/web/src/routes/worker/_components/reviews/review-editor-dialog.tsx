import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createPerformanceReview,
  fetchActivePerformanceReviewTemplates,
  REVIEW_TEMPLATES_KEY,
  updatePerformanceReview,
  type PerformanceReviewRow,
} from "@/lib/graphql/performance-review";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
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
import { cn } from "@trenova/shared/lib/utils";
import {
  computeReviewScore,
  createReviewFormSchema,
  REVIEW_GOAL_STATUS_LABELS,
  REVIEW_SCORE_LABELS,
  reviewDraftFormSchema,
  reviewGoalStatusSchema,
  type CreateReviewFormValues,
  type ReviewDraftFormValues,
} from "@trenova/shared/types/performance-review";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { useEffect, useMemo } from "react";
import {
  FormProvider,
  useFieldArray,
  useForm,
  useFormContext,
  useWatch,
  type Resolver,
} from "react-hook-form";
import { toast } from "sonner";
import { useReviewInvalidation } from "./use-review-invalidation";

const YEAR = 365 * 86_400;

const GOAL_STATUS_OPTIONS = reviewGoalStatusSchema.options.map((value) => ({
  value,
  label: REVIEW_GOAL_STATUS_LABELS[value],
}));

export type ReviewEditorDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The draft being edited; omit to start a new review. */
  review?: PerformanceReviewRow | null;
};

export function ReviewEditorDialog(props: ReviewEditorDialogProps) {
  return props.review ? <EditDraft {...props} review={props.review} /> : <StartReview {...props} />;
}

function StartReview({ open, onOpenChange, workerId }: ReviewEditorDialogProps) {
  const invalidate = useReviewInvalidation(workerId);
  const templatesQuery = useQuery({
    queryKey: [REVIEW_TEMPLATES_KEY],
    queryFn: ({ signal }) => fetchActivePerformanceReviewTemplates({ signal }),
    enabled: open,
    staleTime: 5 * 60 * 1000,
  });

  const form = useForm<CreateReviewFormValues>({
    resolver: zodResolver(createReviewFormSchema) as Resolver<CreateReviewFormValues>,
    defaultValues: {
      templateId: "",
      title: null,
      periodStart: getTodayDate() - YEAR,
      periodEnd: getTodayDate(),
    },
  });
  const { control, handleSubmit, reset, setValue } = form;

  useEffect(() => {
    if (!open) return;
    reset({
      templateId: "",
      title: null,
      periodStart: getTodayDate() - YEAR,
      periodEnd: getTodayDate(),
    });
  }, [open, reset]);

  const templates = useMemo(() => templatesQuery.data ?? [], [templatesQuery.data]);
  useEffect(() => {
    const preferred = templates.find((template) => template.isDefault) ?? templates[0];
    if (preferred) setValue("templateId", preferred.id);
  }, [templates, setValue]);

  const { mutateAsync, isPending } = useApiMutation<
    PerformanceReviewRow,
    CreateReviewFormValues,
    unknown,
    CreateReviewFormValues
  >({
    form,
    resourceName: "Review",
    mutationFn: (values) =>
      createPerformanceReview({
        workerId,
        templateId: values.templateId,
        title: values.title ?? undefined,
        periodStart: values.periodStart,
        periodEnd: values.periodEnd,
      }),
    onSuccess: (saved) => {
      toast.success("Review started", {
        description: `${saved.ratings.length} item${saved.ratings.length === 1 ? "" : "s"} to rate. It stays a draft until you submit it.`,
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Start a review</DialogTitle>
          <DialogDescription>
            The template decides what gets rated. Items are copied onto the review, so later
            template edits will not rewrite it.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <SelectField<CreateReviewFormValues>
                  control={control}
                  name="templateId"
                  label="Template"
                  placeholder={templatesQuery.isLoading ? "Loading..." : "Select a template"}
                  options={templates.map((template) => ({
                    value: template.id,
                    label: template.name,
                    description: `${template.items.length} item${template.items.length === 1 ? "" : "s"}${
                      template.cadenceMonths ? ` · every ${template.cadenceMonths} months` : ""
                    }`,
                  }))}
                  rules={{ required: true }}
                  description={
                    templates.length === 0 && !templatesQuery.isLoading
                      ? "No active templates. Create one under Review Templates first."
                      : "Decides which items are rated and how they are weighted."
                  }
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<CreateReviewFormValues>
                  control={control}
                  name="periodStart"
                  label="Period from"
                  placeholder="A year ago"
                  rules={{ required: true }}
                  description="The first day of the work being reviewed."
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<CreateReviewFormValues>
                  control={control}
                  name="periodEnd"
                  label="Period to"
                  placeholder="Today"
                  rules={{ required: true }}
                  description="The last day being reviewed; it also dates the default title."
                />
              </FormControl>
              <FormControl cols="full">
                <InputField<CreateReviewFormValues>
                  control={control}
                  name="title"
                  label="Title"
                  placeholder="e.g. Annual driver review"
                  maxLength={120}
                  description="Leave blank to use the template name and the period end date."
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                isLoading={isPending}
                loadingText="Starting..."
                disabled={templates.length === 0}
              >
                Start review
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function EditDraft({
  open,
  onOpenChange,
  workerId,
  review,
}: ReviewEditorDialogProps & { review: PerformanceReviewRow }) {
  const invalidate = useReviewInvalidation(workerId);

  const defaults = useMemo<ReviewDraftFormValues>(
    () => ({
      title: review.title,
      periodStart: review.periodStart,
      periodEnd: review.periodEnd,
      ratings: review.ratings.map((rating) => ({
        key: rating.key,
        label: rating.label,
        weight: rating.weight,
        score: rating.score ?? null,
        comment: rating.comment ?? null,
      })),
      summary: review.summary ?? null,
      strengths: review.strengths ?? null,
      improvements: review.improvements ?? null,
      goals: review.goals.map((goal) => ({
        id: goal.id,
        title: goal.title,
        dueAt: goal.dueAt ?? null,
        status: goal.status,
      })),
    }),
    [review],
  );

  const form = useForm<ReviewDraftFormValues>({
    resolver: zodResolver(reviewDraftFormSchema) as Resolver<ReviewDraftFormValues>,
    defaultValues: defaults,
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset(defaults);
  }, [open, defaults, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    PerformanceReviewRow,
    ReviewDraftFormValues,
    unknown,
    ReviewDraftFormValues
  >({
    form,
    resourceName: "Review",
    mutationFn: (values) =>
      updatePerformanceReview({
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
      }),
    onSuccess: () => {
      toast.success("Draft saved", {
        description: "Submit it when you are ready for the worker to sign off.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{review.title}</DialogTitle>
          <DialogDescription>
            Rate every item from 1 to 5 and write the summary the worker will read. Nothing reaches
            them until you submit.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            className="flex min-h-0 flex-1 flex-col"
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <div className="min-h-0 flex-1 overflow-y-auto px-1">
              <RatingsSection />
              <FormGroup className="pb-2" cols={1}>
                <FormControl cols="full">
                  <TextareaField<ReviewDraftFormValues>
                    control={control}
                    name="summary"
                    label="Summary"
                    placeholder="The overall picture, in the words you would use to their face"
                    maxLength={4000}
                    description="The first thing the worker reads once the review is submitted."
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField<ReviewDraftFormValues>
                    control={control}
                    name="strengths"
                    label="Strengths"
                    placeholder="e.g. Clean inspections and on-time deliveries all year"
                    maxLength={4000}
                    description="Specific things the worker should keep doing."
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField<ReviewDraftFormValues>
                    control={control}
                    name="improvements"
                    label="Where to improve"
                    placeholder="e.g. Log fuel receipts the same day"
                    maxLength={4000}
                    description="Where the worker should focus before the next review."
                  />
                </FormControl>
              </FormGroup>
              <GoalsSection />
            </div>
            <DialogFooter className="border-border border-t pt-3">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending} loadingText="Saving...">
                Save draft
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function RatingsSection() {
  const { control } = useFormContext<ReviewDraftFormValues>();
  const ratings = useWatch({ control, name: "ratings" });
  const score = useMemo(
    () => computeReviewScore((ratings ?? []).map((r) => ({ weight: r.weight, score: r.score }))),
    [ratings],
  );

  return (
    <section className="flex flex-col gap-2 py-2">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold">Ratings</h3>
          <p className="text-muted-foreground text-xs">
            Weighted by the template; the score updates as you go.
          </p>
        </div>
        <span className="text-lg font-semibold tabular-nums">
          {score == null ? "—" : score.toFixed(2)}
        </span>
      </div>
      <div className="flex flex-col gap-2">
        {(ratings ?? []).map((rating, index) => (
          <RatingRow key={rating.key} index={index} label={rating.label} weight={rating.weight} />
        ))}
      </div>
    </section>
  );
}

function RatingRow({ index, label, weight }: { index: number; label: string; weight: number }) {
  const { control, setValue } = useFormContext<ReviewDraftFormValues>();
  const score = useWatch({ control, name: `ratings.${index}.score` });

  return (
    <div className="bg-muted/30 rounded-lg border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm font-medium">
          {label}
          <span className="text-muted-foreground ml-1.5 text-[11px]">weight {weight}</span>
        </p>
        <div className="flex items-center gap-1">
          {[1, 2, 3, 4, 5].map((mark) => (
            <button
              key={mark}
              type="button"
              aria-label={`${label}: ${mark} — ${REVIEW_SCORE_LABELS[mark]}`}
              aria-pressed={score === mark}
              className={cn(
                "size-7 rounded-md border text-xs font-medium tabular-nums transition-colors",
                score === mark
                  ? "bg-primary text-primary-foreground border-primary"
                  : "hover:bg-muted",
              )}
              onClick={() =>
                setValue(`ratings.${index}.score`, score === mark ? null : mark, {
                  shouldDirty: true,
                })
              }
            >
              {mark}
            </button>
          ))}
        </div>
      </div>
      <p className="text-muted-foreground mt-1 text-[11px]">
        {score == null ? "Not rated yet" : REVIEW_SCORE_LABELS[score]}
      </p>
      <TextareaField<ReviewDraftFormValues>
        control={control}
        name={`ratings.${index}.comment`}
        label="Comment"
        placeholder="Optional — what you saw"
        maxLength={2000}
        description="The example behind the score; kept with the rating."
      />
    </div>
  );
}

function GoalsSection() {
  const { control } = useFormContext<ReviewDraftFormValues>();
  const goals = useFieldArray({ control, name: "goals" });

  return (
    <section className="flex flex-col gap-2 py-2">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold">Goals</h3>
          <p className="text-muted-foreground text-xs">
            What the worker is aiming at before the next review.
          </p>
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => goals.append({ id: null, title: "", dueAt: null, status: "Open" })}
        >
          <PlusIcon className="size-3.5" />
          Add goal
        </Button>
      </div>
      {goals.fields.map((field, index) => (
        <div
          key={field.id}
          className="bg-muted/30 grid grid-cols-[1fr_auto_auto_auto] items-end gap-2 rounded-lg border p-2"
        >
          <InputField<ReviewDraftFormValues>
            control={control}
            name={`goals.${index}.title`}
            label="Goal"
            placeholder="e.g. Zero late expense submissions this quarter"
            description="One outcome the worker can be measured against."
          />
          <AutoCompleteDateField<ReviewDraftFormValues>
            control={control}
            name={`goals.${index}.dueAt`}
            label="By"
            placeholder="No date"
            description="When the goal should be met."
          />
          <SelectField<ReviewDraftFormValues>
            control={control}
            name={`goals.${index}.status`}
            label="Status"
            placeholder="Pick a status"
            options={GOAL_STATUS_OPTIONS}
            description="Where the goal stands right now."
          />
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="mb-0.5 size-7"
            aria-label={`Remove goal ${index + 1}`}
            onClick={() => goals.remove(index)}
          >
            <Trash2Icon className="size-3.5" />
          </Button>
        </div>
      ))}
    </section>
  );
}
