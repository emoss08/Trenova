import type { PasswordScore } from "@/lib/auth-validation";
import { defineLabels } from "@trenova/shared/i18n/labels";

const STRENGTH_LABELS = defineLabels({
  0: "Too short",
  1: "Weak",
  2: "Fair",
  3: "Good",
  4: "Strong",
});

const LABEL_TONE: Record<PasswordScore, string> = {
  0: "text-muted-foreground",
  1: "text-danger",
  2: "text-muted-foreground",
  3: "text-muted-foreground",
  4: "text-foreground",
};

const SEGMENT_FILL: Record<PasswordScore, string> = {
  0: "bg-border",
  1: "bg-danger",
  2: "bg-accent-amber",
  3: "bg-brand",
  4: "bg-accent-teal",
};

const SEGMENTS = [1, 2, 3, 4] as const;

/** The word for a score, for the password field's label row; nothing until typing starts. */
export function PasswordStrengthLabel({ score, active }: { score: PasswordScore; active: boolean }) {
  return (
    <span aria-live="polite" className={`text-sm ${LABEL_TONE[score]}`}>
      {active ? <span className="auth-enter-quick">{STRENGTH_LABELS[score]}</span> : null}
    </span>
  );
}

/** Four segments under a new password, filled to its score in the score's colour. */
export function PasswordStrengthMeter({ score, active }: { score: PasswordScore; active: boolean }) {
  return (
    <div aria-hidden="true" className="-mt-2 grid grid-cols-4 gap-1">
      {SEGMENTS.map((segment) => (
        <i
          key={segment}
          className={`block h-[3px] rounded-[2px] transition-colors duration-[240ms] ${
            active && segment <= score ? SEGMENT_FILL[score] : "bg-border"
          }`}
        />
      ))}
    </div>
  );
}
