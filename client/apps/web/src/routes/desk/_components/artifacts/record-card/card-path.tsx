import { cn } from "@trenova/shared/lib/utils";
import { CardIcon } from "./record-card";

/** One step of a record's way: taken, where it stands, or still to come. */
export type PathStep = {
  key: string;
  label: string;
  /** What the step says under its name: when and by whom, or what it waits on. */
  line: string;
  state: "done" | "now" | "todo";
  /** How a closing step reads: a hold that can lift, or a close that cannot. */
  tone?: "held" | "closed" | null;
  /** Someone's words left on the step, such as a reviewer's comment or why it was voided. */
  note?: string;
};

/**
 * A record's way down a rail, a node per step: a step taken is ticked, the
 * one it stands at is ringed, a hold is paused, a close is crossed, and a note
 * sits on the step it was left at.
 */
export function CardPath({ steps, label }: { steps: readonly PathStep[]; label: string }) {
  return (
    <ol className="dk-rec-path" aria-label={label}>
      {steps.map((step) => (
        <li key={step.key} className={cn(`dk-${step.state}`, step.tone && `dk-${step.tone}`)}>
          <span className="dk-rec-node" aria-hidden>
            {step.tone === "held" ? (
              <CardIcon name="pause" size={10} stroke={2.6} />
            ) : step.tone === "closed" ? (
              <CardIcon name="x" size={9} stroke={2.8} />
            ) : step.state === "done" ? (
              <CardIcon name="check" size={10} stroke={3} />
            ) : null}
          </span>
          <div className="dk-rec-pb">
            <b>{step.label}</b>
            <span>{step.line}</span>
            {step.note?.trim() && (
              <blockquote className="dk-rec-review">{step.note.trim()}</blockquote>
            )}
          </div>
        </li>
      ))}
    </ol>
  );
}
