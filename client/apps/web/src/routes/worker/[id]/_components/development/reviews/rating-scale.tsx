import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { REVIEW_SCORE_LABELS } from "@trenova/shared/types/performance-review";

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

/**
 * The same scale as a control: each bar sets the score up to it; choosing the score already
 * set clears it. Hovering lightens the bars the click would fill, without moving anything.
 */
export function RatingScaleInput({
  value,
  label,
  onChange,
}: {
  value: number | null;
  label: string;
  onChange: (score: number | null) => void;
}) {
  const t = useT();
  return (
    <span
      role="group"
      aria-label={label}
      className="group/scale inline-grid grid-cols-[repeat(5,1.75rem)] gap-1"
    >
      {STEPS.map((step) => (
        <button
          key={step}
          type="button"
          aria-label={t("{0}: {1}, {2}", label, step, REVIEW_SCORE_LABELS[step])}
          aria-pressed={value === step}
          onClick={() => onChange(value === step ? null : step)}
          className={cn(
            "relative h-2.5 rounded-xs ui-focus-ring transition-colors after:absolute after:-inset-x-0.5 after:-inset-y-2 after:content-['']",
            value != null && step <= value ? "bg-foreground" : "bg-surface-active",
            "group-hover/scale:bg-surface-active hover:bg-foreground/55! has-[~button:hover]:bg-foreground/55!",
            value === step && "animate-confirm motion-reduce:animate-none",
          )}
        />
      ))}
    </span>
  );
}
