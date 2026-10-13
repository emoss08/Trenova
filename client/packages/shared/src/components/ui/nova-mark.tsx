import { cn } from "@trenova/shared/lib/utils";

/**
 * Nova's mark: a ring painted with a sweep of three categorical accents
 * (`nova-ring` in tokens.css), so it reads as Nova's own and never as a tone.
 *
 * It appears in Triage — the Briefing item in its rail, "Drafted by Nova" on a
 * reply, and Nova's card in the Briefing deck — in the Formula Studio, where
 * Nova is the formula assistant: its tab, its hand-off buttons and its
 * answers — and in the command palette, wherever a question goes to Nova: the
 * Ask row and action, the record being asked about, and the answer. Machine
 * suggestions anywhere else keep `AssistMark`, and the Desk never wears it.
 *
 * `spin` runs the colours round the ring, which itself stays still: "hover"
 * while the pointer is on the nearest `group` ancestor (the rail item),
 * "always" while Nova is working. Reduced motion keeps it still.
 */
export function NovaMark({
  className,
  spin = "none",
  label,
}: {
  /** Size it with a class (`size-3.5`); the ring keeps its proportions. */
  className?: string;
  spin?: "none" | "hover" | "always";
  /** Names the mark for assistive technology; omitted, the mark is decoration. */
  label?: string;
}) {
  return (
    <span
      role={label === undefined ? undefined : "img"}
      aria-label={label}
      aria-hidden={label === undefined ? true : undefined}
      className={cn(
        "inline-block size-4 shrink-0 nova-ring",
        spin === "hover" && "group-hover:animate-nova-spin",
        spin === "always" && "animate-nova-spin",
        className,
      )}
    />
  );
}
