import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { REVIEW_SCORE_LABELS } from "@trenova/shared/types/performance-review";
import type { KeyboardEvent } from "react";

const STEPS = [1, 2, 3, 4, 5] as const;

/** The word for a score on the 1 to 5 scale; null while it is not rated. */
export function scoreWord(score: number | null | undefined): string | null {
  if (score == null) return null;
  return REVIEW_SCORE_LABELS[Math.min(5, Math.max(1, Math.round(score)))] ?? null;
}

/** A score drawn as five short ruled bars, filled in ink up to the score. */
export function RatingScale({ score, label }: { score: number | null | undefined; label: string }) {
  const t = useT();
  const word = scoreWord(score);
  return (
    <span
      role="img"
      aria-label={
        word && score != null
          ? t("{0}: {1} of 5, {2}", label, score, word)
          : t("{0}: not rated", label)
      }
      className="inline-grid grid-cols-[repeat(5,1rem)] items-center gap-0.75"
    >
      {STEPS.map((step) => (
        <span
          key={step}
          className={cn(
            "h-1.5 rounded-[1px]",
            score != null && step <= score ? "bg-foreground" : "bg-surface-active",
          )}
        />
      ))}
    </span>
  );
}

/** The step an arrow, Home or End key moves to from `from`; null for any other key. */
function stepFor(key: string, from: number | null): number | null {
  const at = from ?? 0;
  if (key === "ArrowRight" || key === "ArrowUp") return Math.min(5, at + 1);
  if (key === "ArrowLeft" || key === "ArrowDown") return Math.max(1, at === 0 ? 1 : at - 1);
  if (key === "Home") return 1;
  if (key === "End") return 5;
  return null;
}

/**
 * The same scale as a control: a radio group of five bars. A click sets the score up to the bar;
 * choosing the score already set clears it, as Delete or Backspace does. The arrows, Home and End
 * move the score, and Tab reaches the group once, on the bar that is set (the first when none
 * is). Hovering lightens the bars the click would fill, without moving anything.
 */
export function RatingScaleInput({
  value,
  label,
  labelledBy,
  describedBy,
  invalid = false,
  disabled = false,
  onChange,
  onBlur,
}: {
  value: number | null;
  /** Names each bar ("Safety: 4, Strong"). */
  label: string;
  /** The id of the visible label that names the group. */
  labelledBy?: string;
  describedBy?: string;
  invalid?: boolean;
  disabled?: boolean;
  onChange: (score: number | null) => void;
  onBlur?: () => void;
}) {
  const t = useT();
  const tabStop = value ?? 1;

  const onKeyDown = (event: KeyboardEvent<HTMLSpanElement>) => {
    if (disabled) return;
    if (event.key === "Delete" || event.key === "Backspace") {
      event.preventDefault();
      onChange(null);
      return;
    }
    const next = stepFor(event.key, value);
    if (next === null) return;
    event.preventDefault();
    onChange(next);
    event.currentTarget.querySelector<HTMLButtonElement>(`[data-step="${next}"]`)?.focus();
  };

  return (
    <span
      role="radiogroup"
      aria-label={labelledBy ? undefined : label}
      aria-labelledby={labelledBy}
      aria-describedby={describedBy}
      aria-invalid={invalid || undefined}
      onKeyDown={onKeyDown}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) onBlur?.();
      }}
      className="group/scale inline-grid grid-cols-[repeat(5,1.75rem)] gap-1"
    >
      {STEPS.map((step) => (
        <button
          key={step}
          type="button"
          role="radio"
          data-step={step}
          aria-label={t("{0}: {1}, {2}", label, step, REVIEW_SCORE_LABELS[step])}
          aria-checked={value === step}
          tabIndex={step === tabStop ? 0 : -1}
          disabled={disabled}
          onClick={() => onChange(value === step ? null : step)}
          className={cn(
            "relative h-2.5 rounded-xs ui-focus-ring transition-colors after:absolute after:-inset-x-0.5 after:-inset-y-2 after:content-[''] disabled:cursor-not-allowed",
            value != null && step <= value ? "bg-foreground" : "bg-surface-active",
            "group-hover/scale:bg-surface-active hover:bg-foreground/55! has-[~button:hover]:bg-foreground/55!",
            value === step && "animate-confirm motion-reduce:animate-none",
          )}
        />
      ))}
    </span>
  );
}
