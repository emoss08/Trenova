import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  closePerformanceReview,
  deletePerformanceReview,
  fetchActivePerformanceReviewTemplates,
  fetchWorkerPerformanceReviews,
  reopenPerformanceReview,
  REVIEW_TEMPLATES_KEY,
  submitPerformanceReview,
  WORKER_REVIEWS_KEY,
  type PerformanceReviewRow,
} from "@/lib/graphql/performance-review";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ChevronDownIcon, ClipboardCheckIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { ReviewCard, type ReviewPermissions } from "./reviews/review-card";
import { ReviewEditorDialog } from "./reviews/review-editor-dialog";
import { useReviewInvalidation } from "./reviews/use-review-invalidation";

export default function WorkerReviewsTab({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canCreate } = usePermission(Resource.PerformanceReview, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.PerformanceReview, Operation.Update);
  const { allowed: canSubmit } = usePermission(Resource.PerformanceReview, Operation.Submit);
  const { allowed: canClose } = usePermission(Resource.PerformanceReview, Operation.Close);
  const { allowed: canDelete } = usePermission(Resource.PerformanceReview, Operation.Delete);
  const invalidate = useReviewInvalidation(workerId);
  const [editing, setEditing] = useState<{ review: PerformanceReviewRow | null } | null>(null);
  const [historyOpen, setHistoryOpen] = useState(false);

  const reviewsQuery = useQuery({
    queryKey: [WORKER_REVIEWS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPerformanceReviews(workerId, { signal }),
  });
  const templatesQuery = useQuery({
    queryKey: [REVIEW_TEMPLATES_KEY],
    queryFn: ({ signal }) => fetchActivePerformanceReviewTemplates({ signal }),
    enabled: canCreate,
    staleTime: 5 * 60 * 1000,
  });

  const reviews = useMemo(() => reviewsQuery.data ?? [], [reviewsQuery.data]);
  const open = useMemo(() => reviews.filter((r) => r.status !== "Closed"), [reviews]);
  const closed = useMemo(() => reviews.filter((r) => r.status === "Closed"), [reviews]);
  const hasTemplates = (templatesQuery.data ?? []).length > 0;

  const submit = useMutation({
    mutationFn: (review: PerformanceReviewRow) =>
      submitPerformanceReview({ id: review.id, version: review.version }),
    onSuccess: () => {
      toast.success(t("Review submitted"), {
        description: t("The worker has been asked to read it and sign off in Dash."),
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not submit review"), { description: error.message }),
  });
  const reopen = useMutation({
    mutationFn: (review: PerformanceReviewRow) =>
      reopenPerformanceReview({ id: review.id, version: review.version }),
    onSuccess: () => {
      toast.success(t("Review reopened as a draft"));
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not reopen review"), { description: error.message }),
  });
  const close = useMutation({
    mutationFn: (review: PerformanceReviewRow) =>
      closePerformanceReview({ id: review.id, version: review.version }),
    onSuccess: (review) => {
      toast.success(t("Review closed"), {
        description: review.nextReviewAt
          ? "The next review is scheduled from the template cadence."
          : "Filed on the worker's record.",
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not close review"), { description: error.message }),
  });
  const remove = useMutation({
    mutationFn: (review: PerformanceReviewRow) => deletePerformanceReview(review.id),
    onSuccess: () => {
      toast.success(t("Draft deleted"));
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not delete draft"), { description: error.message }),
  });

  const permissions = useMemo<ReviewPermissions>(
    () => ({ canUpdate, canSubmit, canClose, canDelete }),
    [canClose, canDelete, canSubmit, canUpdate],
  );
  const busy = submit.isPending || reopen.isPending || close.isPending || remove.isPending;

  if (reviewsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-40 w-full rounded-xl" />
        <Skeleton className="h-24 w-full rounded-xl" />
      </div>
    );
  }

  const cardProps = {
    permissions,
    busy,
    onEdit: (review: PerformanceReviewRow) => setEditing({ review }),
    onSubmit: (review: PerformanceReviewRow) => submit.mutate(review),
    onReopen: (review: PerformanceReviewRow) => reopen.mutate(review),
    onClose: (review: PerformanceReviewRow) => close.mutate(review),
    onDelete: (review: PerformanceReviewRow) => remove.mutate(review),
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold">{t("Performance reviews")}</h3>
          <p className="text-muted-foreground text-xs">
            {t("Drafted here, signed off by the worker in Dash, then closed and scheduled again.")}
          </p>
        </div>
        {canCreate ? (
          <Button
            size="sm"
            disabled={!hasTemplates && !templatesQuery.isLoading}
            onClick={() => setEditing({ review: null })}
          >
            <PlusIcon className="size-3.5" />
            {t("Start a review")}
          </Button>
        ) : null}
      </div>

      {reviews.length === 0 ? (
        <div className="border-border text-muted-foreground flex flex-col items-center gap-2 rounded-xl border border-dashed px-4 py-8 text-center text-sm">
          <ClipboardCheckIcon className="size-5" />
          {t("No reviews yet")}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {open.map((review) => (
            <ReviewCard key={review.id} review={review} {...cardProps} />
          ))}
        </div>
      )}

      {closed.length > 0 ? (
        <section className="flex flex-col gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground w-fit px-1"
            aria-expanded={historyOpen}
            onClick={() => setHistoryOpen((value) => !value)}
          >
            <ChevronDownIcon
              className={`size-3.5 transition-transform ${historyOpen ? "rotate-180" : ""}`}
            />
            {t("History ({0})", closed.length)}
          </Button>
          {historyOpen
            ? closed.map((review) => <ReviewCard key={review.id} review={review} {...cardProps} />)
            : null}
        </section>
      ) : null}

      <ReviewEditorDialog
        open={editing !== null}
        onOpenChange={(isOpen) => {
          if (!isOpen) setEditing(null);
        }}
        workerId={workerId}
        review={editing?.review ?? null}
      />
    </div>
  );
}
