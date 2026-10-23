import {
  ReadinessView,
  isReadyFor,
  type ReadinessSourceLink,
} from "@/components/tariff-studio/readiness-view";
import type { ReadinessCheck } from "@trenova/shared/types/formula-template";
import { ReviewDiffView } from "@/components/tariff-studio/review-diff-view";
import { ReviewHistoryView } from "@/components/tariff-studio/review-history-view";
import { describeApiError } from "@/lib/api-error-message";
import type { FormulaLibraryRecord } from "@/lib/graphql/formula-library";
import { formulaLibrary, invalidateFormulaLibrary } from "@/lib/queries/formula-library";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  MessageAlertSquareIcon,
  Send01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { toast } from "@trenova/shared/components/ui/toast";
import { defineLabels, type LabelMap } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState, type ReactNode } from "react";
import { reviewEntriesOf } from "./library-history-pane";
import { libraryPath } from "./starter";

const MAX_NOTE = 4000;

function Label({ children }: { children: ReactNode }) {
  return <span className="text-foreground-subtle pt-1 text-xs font-medium">{children}</span>;
}

/**
 * Sending the saved library to review: the checks Submit enforces, what changed since the
 * approved version, who should look and a note for them.
 */
export function LibrarySubmitDialog({
  open,
  onOpenChange,
  library,
  liveChecks,
  link,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  library: FormulaLibraryRecord;
  /** The Studio's live checks, shown for what the server's readiness does not judge. */
  liveChecks?: readonly ReadinessCheck[];
  /** Takes a readiness row's region to the source, closing the dialog. */
  link?: ReadinessSourceLink;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [reviewerId, setReviewerId] = useState<string | null>(null);
  const [note, setNote] = useState("");

  const readiness = useQuery({
    ...formulaLibrary.readiness(library.id),
    enabled: open,
    staleTime: 0,
  });
  const diff = useQuery({ ...formulaLibrary.reviewDiff(library.id), enabled: open, staleTime: 0 });
  const reviewers = useQuery({ ...formulaLibrary.reviewers(library.id), enabled: open });

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
      apiService.formulaLibraryService.submit(library.id, {
        comment: "",
        reviewerId: reviewerId ?? undefined,
        reviewerNote: note.trim(),
      }),
    onSuccess: async () => {
      const reviewer = reviewers.data?.find((person) => person.id === reviewerId);
      toast(reviewer ? t("Sent to {0} for review", reviewer.name) : t("Submitted for review"));
      onOpenChange(false);
      await invalidateFormulaLibrary(queryClient);
    },
    onError: (error) =>
      toast.error(t("Could not submit the library"), { description: describeApiError(error) }),
  });

  const ready = readiness.data ? isReadyFor("submit", readiness.data) : null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="xl"
        className="max-h-[88vh] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 p-0"
      >
        <DialogHeader className="border-b px-5 pt-5 pb-4">
          <DialogTitle>{t("Submit {0} for review", libraryPath(library.name))}</DialogTitle>
          <DialogDescription>
            {t(
              "The reviewer approves exactly what is saved. Once approved, new template versions pin this version when they import the library.",
            )}
          </DialogDescription>
        </DialogHeader>
        <ScrollArea maskVariant="card" className="min-h-0">
          <div className="grid grid-cols-1 gap-5 px-5 py-4 md:grid-cols-2">
            <div className="flex flex-col gap-2">
              <Label>{t("Readiness")}</Label>
              <ReadinessView
                data={readiness.data}
                isLoading={readiness.isPending}
                isError={readiness.isError}
                step="submit"
                liveChecks={liveChecks}
                link={link}
              />
              <Label>{t("Reviewer")}</Label>
              {reviewers.isPending ? (
                <div className="flex flex-col gap-1.5">
                  <Skeleton className="h-11 w-full" />
                  <Skeleton className="h-11 w-full" />
                </div>
              ) : (reviewers.data?.length ?? 0) === 0 ? (
                <p className="text-foreground-muted rounded-md border px-3 py-2 text-xs">
                  {t(
                    "Nobody else here can approve formula libraries. Anyone with approve permission will see it in their notifications.",
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
            </div>
            <div className="flex flex-col gap-2">
              <Label>{t("Note for the reviewer")}</Label>
              <Textarea
                value={note}
                onChange={(event) => setNote(event.target.value.slice(0, MAX_NOTE))}
                minRows={4}
                maxRows={10}
                aria-label={t("Note for the reviewer")}
                placeholder={t("What changed, and which templates should pin the new version")}
              />
              <Label>{t("What changes")}</Label>
              <ReviewDiffView data={diff.data} isLoading={diff.isPending} isError={diff.isError} />
            </div>
          </div>
        </ScrollArea>
        <DialogFooter className="mx-0 mb-0 items-center px-5 py-3">
          <span className="text-foreground-muted mr-auto text-xs">
            {ready === false
              ? t("Fix the blocking checks to submit.")
              : t("You'll be notified when it's reviewed.")}
          </span>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            className="gap-1.5"
            disabled={ready !== true}
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

export type LibraryDecision = "approve" | "requestChanges" | "reject";

const DECISIONS: Record<
  LibraryDecision,
  {
    copy: LabelMap<{
      title: string;
      description: string;
      confirm: string;
      done: string;
      failed: string;
      placeholder: string;
    }>;
    icon: typeof CheckIcon;
    destructive: boolean;
  }
> = {
  approve: {
    copy: defineLabels({
      title: "Approve the library",
      description:
        "New template versions that import it pin this version. Template versions already approved keep the version they pinned.",
      confirm: "Approve",
      done: "Library approved",
      failed: "Could not approve the library",
      placeholder: "A note for the record (optional)",
    }),
    icon: CheckIcon,
    destructive: false,
  },
  requestChanges: {
    copy: defineLabels({
      title: "Request changes",
      description: "The library goes back to its author as a draft, with your comment.",
      confirm: "Request changes",
      done: "Changes requested",
      failed: "Could not request changes",
      placeholder: "What needs to change",
    }),
    icon: MessageAlertSquareIcon,
    destructive: false,
  },
  reject: {
    copy: defineLabels({
      title: "Reject and archive",
      description:
        "The library is archived, and no new template version can import it. Template versions that already import it keep the version they pinned.",
      confirm: "Reject",
      done: "Library rejected",
      failed: "Could not reject the library",
      placeholder: "Why it is rejected",
    }),
    icon: XCloseIcon,
    destructive: true,
  },
};

/**
 * A reviewer's decision on a library in review: the checks Approve enforces, what would change
 * and the conversation so far, then the decision with its comment. Requesting changes and
 * rejecting need a comment; it is the whole point.
 */
export function LibraryDecisionDialog({
  decision,
  onClose,
  library,
}: {
  decision: LibraryDecision | null;
  onClose: () => void;
  library: FormulaLibraryRecord;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [comment, setComment] = useState("");
  const open = decision !== null;
  const config = decision ? DECISIONS[decision] : null;
  const commentRequired = decision === "reject" || decision === "requestChanges";

  const readiness = useQuery({
    ...formulaLibrary.readiness(library.id),
    enabled: decision === "approve",
    staleTime: 0,
  });
  const diff = useQuery({ ...formulaLibrary.reviewDiff(library.id), enabled: open, staleTime: 0 });
  const reviews = useMemo(() => reviewEntriesOf(library), [library]);

  const mutation = useMutation({
    mutationFn: (action: LibraryDecision) => {
      const text = comment.trim();
      if (action === "approve") return apiService.formulaLibraryService.approve(library.id, text);
      if (action === "reject") return apiService.formulaLibraryService.reject(library.id, text);
      return apiService.formulaLibraryService.requestChanges(library.id, text);
    },
    onSuccess: async (_, action) => {
      toast.success(DECISIONS[action].copy.done);
      setComment("");
      onClose();
      await invalidateFormulaLibrary(queryClient);
    },
    onError: (error, action) =>
      toast.error(DECISIONS[action].copy.failed, { description: describeApiError(error) }),
  });

  const blocked =
    decision === "approve"
      ? readiness.data
        ? !isReadyFor("approve", readiness.data)
        : true
      : comment.trim() === "";
  const Icon = config?.icon ?? CheckIcon;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setComment("");
          onClose();
        }
      }}
    >
      <DialogContent
        size="xl"
        className="max-h-[88vh] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 p-0"
      >
        <DialogHeader className="border-b px-5 pt-5 pb-4">
          <DialogTitle>{config?.copy.title}</DialogTitle>
          <DialogDescription>{config?.copy.description}</DialogDescription>
        </DialogHeader>
        <ScrollArea maskVariant="card" className="min-h-0">
          <div className="flex flex-col gap-3 px-5 py-4">
            {decision === "approve" && (
              <ReadinessView
                data={readiness.data}
                isLoading={readiness.isPending}
                isError={readiness.isError}
                step="approve"
              />
            )}
            <ReviewDiffView data={diff.data} isLoading={diff.isPending} isError={diff.isError} />
            <ReviewHistoryView
              reviews={reviews}
              isLoading={false}
              isError={false}
              emptyText={t("This library has never been submitted for review.")}
            />
            <Textarea
              value={comment}
              onChange={(event) => setComment(event.target.value)}
              minRows={3}
              maxLength={2000}
              aria-label={commentRequired ? t("Comment") : t("Comment (optional)")}
              aria-required={commentRequired || undefined}
              placeholder={config?.copy.placeholder}
            />
          </div>
        </ScrollArea>
        <DialogFooter className="mx-0 mb-0 items-center px-5 py-3">
          {commentRequired && comment.trim() === "" && (
            <span className="text-foreground-muted mr-auto text-xs">
              {t("Say what needs to change; the author reads it.")}
            </span>
          )}
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant={config?.destructive ? "destructive" : "default"}
            className="gap-1.5"
            disabled={blocked}
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
