import type { AiFeedbackRating, AiFeedbackReason } from "@/lib/graphql/ai-feedback";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
} from "@trenova/shared/components/ui/popover";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState, type RefObject } from "react";
import {
  FEEDBACK_COMMENT_LIMIT,
  reasonsFor,
  reasonsMatching,
  toggleReason,
} from "./feedback-reasons";

export type FeedbackReasonDraft = {
  reasons: AiFeedbackReason[];
  comment: string;
};

/**
 * Why a person rated as they did, asked once they have rated.
 *
 * The rating is already saved when this opens; the reasons and the comment
 * are extra, so closing it keeps the thumb and sending adds to it. It floats
 * from the thumb, so it is inverted like every other popover and built from
 * tokens alone.
 */
export function FeedbackReasonPopover({
  rating,
  anchor,
  initial,
  onClose,
  onSend,
}: {
  /** The thumb just chosen; null keeps the popover closed. */
  rating: AiFeedbackRating | null;
  anchor: RefObject<HTMLElement | null>;
  initial: FeedbackReasonDraft;
  onClose: () => void;
  onSend: (draft: FeedbackReasonDraft) => void;
}) {
  return (
    <Popover
      open={rating !== null}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <PopoverContent anchor={anchor} side="bottom" align="end" className="w-72">
        {rating !== null && (
          <ReasonForm key={rating} rating={rating} initial={initial} onSend={onSend} />
        )}
      </PopoverContent>
    </Popover>
  );
}

function ReasonForm({
  rating,
  initial,
  onSend,
}: {
  rating: AiFeedbackRating;
  initial: FeedbackReasonDraft;
  onSend: (draft: FeedbackReasonDraft) => void;
}) {
  const t = useT();
  const commentId = useId();
  const noticeId = useId();
  const [reasons, setReasons] = useState<AiFeedbackReason[]>(() =>
    reasonsMatching(rating, initial.reasons),
  );
  const [comment, setComment] = useState(initial.comment);
  const options = reasonsFor(rating);

  const send = () => onSend({ reasons: reasonsMatching(rating, reasons), comment: comment.trim() });

  return (
    <form
      className="flex flex-col gap-2.5"
      onSubmit={(event) => {
        event.preventDefault();
        send();
      }}
    >
      <PopoverHeader>
        <PopoverTitle className="text-sm">
          {rating === 1 ? t("What worked?") : t("What went wrong?")}
        </PopoverTitle>
      </PopoverHeader>

      <div role="group" aria-label={t("Reasons")} className="flex flex-wrap gap-1.5">
        {options.map((option) => {
          const chosen = reasons.includes(option.value);

          return (
            <button
              key={option.value}
              type="button"
              aria-pressed={chosen}
              onClick={() => setReasons((current) => toggleReason(current, option.value))}
              className={cn(
                "ui-focus-ring ui-press border-border text-foreground-muted flex h-6 items-center rounded-full border px-2.5 text-xs transition-colors",
                "hover:bg-surface-hover hover:text-foreground",
                "aria-pressed:border-border-strong aria-pressed:bg-surface-selected aria-pressed:text-foreground",
              )}
            >
              {t(option.label)}
            </button>
          );
        })}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor={commentId} className="sr-only">
          {t("Comment")}
        </label>
        <Textarea
          id={commentId}
          value={comment}
          onChange={(event) => setComment(event.target.value.slice(0, FEEDBACK_COMMENT_LIMIT))}
          maxLength={FEEDBACK_COMMENT_LIMIT}
          minRows={2}
          maxRows={6}
          placeholder={t("Anything else? (optional)")}
          aria-describedby={noticeId}
          onKeyDown={(event) => {
            if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
              event.preventDefault();
              send();
            }
          }}
        />
        <span className="text-foreground-subtle self-end text-2xs tabular-nums">
          {comment.length}/{FEEDBACK_COMMENT_LIMIT}
        </span>
      </div>

      <p id={noticeId} className="text-foreground-muted text-xs leading-snug">
        {t("Administrators can see the question, the answer and your rating.")}
      </p>

      <div className="flex justify-end">
        <Button type="submit" size="sm">
          {t("Send")}
        </Button>
      </div>
    </form>
  );
}
