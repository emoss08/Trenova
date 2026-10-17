import { cn } from "@trenova/shared/lib/utils";

export type BriefPhraseTone = "warning" | "danger";

const BRIEF_PHRASE_CLASS =
  "ui-focus-ring decoration-border-strong hover:bg-brand/20 rounded-sm underline decoration-1 underline-offset-4 transition-colors";

const BRIEF_PHRASE_TONE_CLASS: Record<BriefPhraseTone, string> = {
  warning: "text-warning decoration-warning/60 hover:bg-warning/20",
  danger: "text-danger decoration-danger/60 hover:bg-danger/20",
};

/**
 * How a phrase of a brief that leads somewhere is drawn: underlined in the sentence, coloured for
 * a warning or a failure, set in the foreground when it carries the figure the eye should land
 * on. Every page brief draws its phrases with this, whether the brief streams in or not.
 */
export function briefPhraseClass(tone?: BriefPhraseTone, emphasis = false): string {
  return cn(
    BRIEF_PHRASE_CLASS,
    emphasis && "text-foreground font-medium",
    tone && BRIEF_PHRASE_TONE_CLASS[tone],
  );
}
