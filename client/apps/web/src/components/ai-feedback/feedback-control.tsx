import type { AiFeedbackRating, AiFeedbackTarget } from "@/lib/graphql/ai-feedback";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ThumbsDownIcon, ThumbsUpIcon, type LucideIcon } from "lucide-react";
import { useRef, useState, type RefObject } from "react";
import { FeedbackReasonPopover, type FeedbackReasonDraft } from "./feedback-reason-popover";
import { useAiFeedback } from "./use-ai-feedback";

/**
 * A thumbs up and a thumbs down for something an AI wrote.
 *
 * A thumb saves the moment it is pressed, then asks why in a popover the
 * person may ignore. Pressing the chosen thumb again takes the rating back.
 * The thumbs may share a hover fade with the actions around them
 * (`revealClassName`), but a chosen thumb never fades: the rating is a fact
 * about the answer, not an action waiting on the pointer.
 */
export function FeedbackControl({
  target,
  revealClassName,
  className,
}: {
  target: AiFeedbackTarget;
  /** Classes that hide an unchosen thumb until its surroundings are hovered. */
  revealClassName?: string;
  className?: string;
}) {
  const t = useT();
  const { feedback, rating, submit, clear } = useAiFeedback(target);
  const [asking, setAsking] = useState<AiFeedbackRating | null>(null);
  const upRef = useRef<HTMLButtonElement>(null);
  const downRef = useRef<HTMLButtonElement>(null);

  const choose = (value: AiFeedbackRating) => {
    if (rating === value) {
      setAsking(null);
      clear();

      return;
    }
    submit({ rating: value, reasons: [], comment: "" }, { onError: () => setAsking(null) });
    setAsking(value);
  };

  const send = (draft: FeedbackReasonDraft) => {
    if (asking !== null) {
      submit({ rating: asking, reasons: draft.reasons, comment: draft.comment });
    }
    setAsking(null);
  };

  const initial: FeedbackReasonDraft = {
    reasons: feedback?.reasons ?? [],
    comment: feedback?.comment ?? "",
  };

  return (
    <span
      role="group"
      aria-label={t("Rate this")}
      className={cn("inline-flex shrink-0 items-center gap-0.5", className)}
    >
      <Thumb
        buttonRef={upRef}
        icon={ThumbsUpIcon}
        label={t("Helpful")}
        pressed={rating === 1}
        open={asking === 1}
        revealClassName={revealClassName}
        onClick={() => choose(1)}
      />
      <Thumb
        buttonRef={downRef}
        icon={ThumbsDownIcon}
        label={t("Not helpful")}
        pressed={rating === -1}
        open={asking === -1}
        revealClassName={revealClassName}
        onClick={() => choose(-1)}
      />
      <FeedbackReasonPopover
        rating={asking}
        anchor={asking === 1 ? upRef : downRef}
        initial={initial}
        onClose={() => setAsking(null)}
        onSend={send}
      />
    </span>
  );
}

function Thumb({
  buttonRef,
  icon: Icon,
  label,
  pressed,
  open,
  revealClassName,
  onClick,
}: {
  buttonRef: RefObject<HTMLButtonElement | null>;
  icon: LucideIcon;
  label: string;
  pressed: boolean;
  open: boolean;
  revealClassName?: string;
  onClick: () => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            ref={buttonRef}
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={label}
            aria-pressed={pressed}
            aria-expanded={open}
            onClick={onClick}
            className={cn(
              "text-muted-foreground hover:text-foreground transition-[color,opacity]",
              "aria-pressed:text-brand",
              !pressed && !open && revealClassName,
              "focus-visible:opacity-100",
            )}
          />
        }
      >
        <Icon className={cn("size-3", pressed && "fill-current")} aria-hidden />
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
