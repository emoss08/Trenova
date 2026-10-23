import { describeApiError } from "@/lib/api-error-message";
import type { RateMatrixRecord } from "@/lib/graphql/rate-matrix";
import { invalidateRateMatrix, rateMatrix } from "@/lib/queries/rate-matrix";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArchiveIcon,
  CheckIcon,
  ClockRewindIcon,
  MessageAlertSquareIcon,
  Send01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { toast } from "@trenova/shared/components/ui/toast";
import { defineLabels, type LabelMap } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState, type ReactNode } from "react";

const MAX_NOTE = 4000;

function Label({ children }: { children: ReactNode }) {
  return <span className="text-foreground-subtle pt-1 text-xs font-medium">{children}</span>;
}

/**
 * Sending a matrix draft to review: who should look and a note for them. The reviewer publishes
 * exactly the saved draft as the next version.
 */
export function RateMatrixSubmitDialog({
  open,
  onOpenChange,
  matrix,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  matrix: RateMatrixRecord;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [reviewerId, setReviewerId] = useState<string | null>(null);
  const [note, setNote] = useState("");
  const reviewers = useQuery({ ...rateMatrix.reviewers(matrix.id), enabled: open });

  const [seenOpen, setSeenOpen] = useState(open);
  if (open !== seenOpen) {
    setSeenOpen(open);
    if (!open) {
      setNote("");
      setReviewerId(null);
    }
  }
  const firstReviewer = reviewers.data?.[0];
  if (open && reviewerId === null && firstReviewer) setReviewerId(firstReviewer.id);

  const submit = useMutation({
    mutationFn: () =>
      apiService.rateMatrixService.submit(matrix.id, {
        comment: "",
        reviewerId: reviewerId ?? undefined,
        reviewerNote: note.trim(),
      }),
    onSuccess: async () => {
      onOpenChange(false);
      await invalidateRateMatrix(queryClient);
    },
    onError: (error) =>
      toast.error(t("Could not submit the matrix"), { description: describeApiError(error) }),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Submit {0} for review", matrix.code)}</DialogTitle>
          <DialogDescription>
            {t(
              "The reviewer publishes exactly what is saved as the next version. Template versions approved from then on pin it; those already approved keep the version they pinned.",
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          <Label>{t("Reviewer")}</Label>
          {reviewers.isPending ? (
            <div
              className="flex flex-col gap-1.5"
              aria-busy="true"
              aria-label={t("Loading reviewers")}
            >
              <Skeleton className="h-11 w-full" />
              <Skeleton className="h-11 w-full" />
            </div>
          ) : (reviewers.data?.length ?? 0) === 0 ? (
            <p className="text-foreground-muted rounded-md border px-3 py-2 text-xs">
              {t(
                "Nobody else here can publish rate matrices. Anyone with publish permission will see it in their notifications.",
              )}
            </p>
          ) : (
            <div role="radiogroup" aria-label={t("Reviewer")} className="flex flex-col gap-1.5">
              {reviewers.data?.map((person) => {
                const selected = person.id === reviewerId;
                return (
                  <Button
                    key={person.id}
                    type="button"
                    variant="outline"
                    size="bare"
                    role="radio"
                    aria-checked={selected}
                    onClick={() => setReviewerId(person.id)}
                    className={cn(
                      "flex h-auto items-center justify-start gap-2.5 rounded-md px-2.5 py-2 text-left",
                      selected && "border-foreground bg-surface-selected",
                    )}
                  >
                    <span className="flex min-w-0 flex-1 flex-col">
                      <span className="truncate text-sm font-medium">{person.name}</span>
                      {person.username && (
                        <span className="text-foreground-subtle truncate text-xs font-normal">
                          {person.username}
                        </span>
                      )}
                    </span>
                    {selected && <CheckIcon className="size-3.5" />}
                  </Button>
                );
              })}
            </div>
          )}
          <Label>{t("Note for the reviewer")}</Label>
          <Textarea
            value={note}
            onChange={(event) => setNote(event.target.value.slice(0, MAX_NOTE))}
            minRows={3}
            maxRows={8}
            aria-label={t("Note for the reviewer")}
            placeholder={t("What changed in the rates, and why")}
          />
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            className="gap-1.5"
            isLoading={submit.isPending}
            onClick={() => submit.mutate()}
          >
            <Send01Icon className="size-3.5" />
            {t("Submit")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export type RateMatrixDecision = "approve" | "requestChanges" | "reject";

const DECISIONS: Record<
  RateMatrixDecision,
  {
    copy: LabelMap<{
      title: string;
      description: string;
      confirm: string;
      failed: string;
      placeholder: string;
    }>;
    icon: typeof CheckIcon;
    destructive: boolean;
  }
> = {
  approve: {
    copy: defineLabels({
      title: "Publish the draft",
      description:
        "The draft becomes the next published version. Template versions approved from then on pin it; contracts already rating keep the version their template version pinned.",
      confirm: "Publish",
      failed: "Could not publish the matrix",
      placeholder: "A note for the record (optional)",
    }),
    icon: CheckIcon,
    destructive: false,
  },
  requestChanges: {
    copy: defineLabels({
      title: "Request changes",
      description: "The draft goes back to its editors, with your comment.",
      confirm: "Request changes",
      failed: "Could not request changes",
      placeholder: "What needs to change",
    }),
    icon: MessageAlertSquareIcon,
    destructive: false,
  },
  reject: {
    copy: defineLabels({
      title: "Reject the draft",
      description:
        "The draft is not published. A matrix never published is archived; one already published goes back to a draft and tariffs keep reading its newest version.",
      confirm: "Reject",
      failed: "Could not reject the draft",
      placeholder: "Why it is rejected",
    }),
    icon: XCloseIcon,
    destructive: true,
  },
};

/**
 * A reviewer's decision on a submitted draft. Requesting changes and rejecting need a comment;
 * it is the whole point.
 */
export function RateMatrixDecisionDialog({
  decision,
  onClose,
  matrix,
}: {
  decision: RateMatrixDecision | null;
  onClose: () => void;
  matrix: RateMatrixRecord;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [comment, setComment] = useState("");
  const config = decision ? DECISIONS[decision] : null;
  const commentRequired = decision === "reject" || decision === "requestChanges";

  const mutation = useMutation({
    mutationFn: (action: RateMatrixDecision) => {
      const text = comment.trim();
      if (action === "approve") return apiService.rateMatrixService.approve(matrix.id, text);
      if (action === "reject") return apiService.rateMatrixService.reject(matrix.id, text);
      return apiService.rateMatrixService.requestChanges(matrix.id, text);
    },
    onSuccess: async () => {
      setComment("");
      onClose();
      await invalidateRateMatrix(queryClient);
    },
    onError: (error, action) =>
      toast.error(DECISIONS[action].copy.failed, { description: describeApiError(error) }),
  });

  const Icon = config?.icon ?? CheckIcon;

  return (
    <Dialog
      open={decision !== null}
      onOpenChange={(next) => {
        if (!next) {
          setComment("");
          onClose();
        }
      }}
    >
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{config?.copy.title}</DialogTitle>
          <DialogDescription>{config?.copy.description}</DialogDescription>
        </DialogHeader>
        <Textarea
          value={comment}
          onChange={(event) => setComment(event.target.value)}
          minRows={3}
          maxLength={2000}
          aria-label={commentRequired ? t("Comment") : t("Comment (optional)")}
          aria-required={commentRequired || undefined}
          placeholder={config?.copy.placeholder}
        />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant={config?.destructive ? "destructive" : "default"}
            className="gap-1.5"
            disabled={commentRequired && comment.trim() === ""}
            isLoading={mutation.isPending}
            onClick={() => decision && mutation.mutate(decision)}
          >
            <Icon className="size-3.5" />
            {config?.copy.confirm}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/**
 * Archiving takes the matrix out of every new template version. Template versions that pin it
 * keep reading the version they pinned, so archiving one they pin asks for a confirmation.
 */
export function RateMatrixArchiveDialog({
  open,
  onOpenChange,
  matrix,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  matrix: RateMatrixRecord;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const pinned = matrix.usages.length;
  const [confirmed, setConfirmed] = useState(false);
  const confirmId = useId();

  const archive = useMutation({
    mutationFn: () => apiService.rateMatrixService.archive(matrix.id, pinned > 0),
    onSuccess: async () => {
      setConfirmed(false);
      onOpenChange(false);
      await invalidateRateMatrix(queryClient);
    },
    onError: (error) =>
      toast.error(t("Could not archive the matrix"), { description: describeApiError(error) }),
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) setConfirmed(false);
        onOpenChange(next);
      }}
    >
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Archive {0}", matrix.code)}</DialogTitle>
          <DialogDescription>
            {t(
              "No new template version can read tables.{0} once it is archived. Submitting a draft brings it back.",
              matrix.code,
            )}
          </DialogDescription>
        </DialogHeader>
        {pinned > 0 && (
          <div className="flex items-start gap-2 text-sm">
            <Checkbox
              id={confirmId}
              checked={confirmed}
              onCheckedChange={(checked) => setConfirmed(checked === true)}
            />
            <label htmlFor={confirmId}>
              {t(
                "{0, plural, one {# template version pins} other {# template versions pin}} a version of this matrix. They keep reading the version they pinned. Archive it anyway.",
                pinned,
              )}
            </label>
          </div>
        )}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            className="gap-1.5"
            disabled={pinned > 0 && !confirmed}
            isLoading={archive.isPending}
            onClick={() => archive.mutate()}
          >
            <ArchiveIcon className="size-3.5" />
            {t("Archive")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** Replacing the draft's axes and cells with a published version's. */
export function RateMatrixRestoreDialog({
  versionNumber,
  onClose,
  matrix,
}: {
  versionNumber: number | null;
  onClose: () => void;
  matrix: RateMatrixRecord;
}) {
  const t = useT();
  const queryClient = useQueryClient();

  const restore = useMutation({
    mutationFn: (target: number) => apiService.rateMatrixService.restore(matrix.id, target),
    onSuccess: async () => {
      onClose();
      await invalidateRateMatrix(queryClient);
    },
    onError: (error) =>
      toast.error(t("Could not restore the version"), { description: describeApiError(error) }),
  });

  return (
    <Dialog open={versionNumber !== null} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Restore version {0} to the draft", versionNumber ?? 0)}</DialogTitle>
          <DialogDescription>
            {versionNumber === matrix.publishedVersionNumber
              ? t(
                  "The draft's axes and rates are replaced with the published version's, discarding the draft's changes.",
                )
              : t(
                  "The draft's axes and rates are replaced with this version's. Tariffs keep reading version {0} until the draft is published.",
                  matrix.publishedVersionNumber,
                )}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            className="gap-1.5"
            isLoading={restore.isPending}
            onClick={() => versionNumber !== null && restore.mutate(versionNumber)}
          >
            <ClockRewindIcon className="size-3.5" />
            {t("Restore")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
