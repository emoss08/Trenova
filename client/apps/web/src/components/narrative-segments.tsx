import { cn } from "@trenova/shared/lib/utils";
import type { KeyboardEvent } from "react";

/** One run of a sentence the server wrote: plain, strong, or a reference to what it names. */
export type NarrativeSegment = {
  text: string;
  strong: boolean;
  tone?: string | null;
  target?: string | null;
  targetId?: string | null;
};

function toneClass(tone: string | null | undefined): string {
  if (tone === "danger") return "text-danger-foreground";
  if (tone === "good") return "text-success-foreground";
  return "text-foreground";
}

/**
 * A server-written sentence drawn as one run of text: strong figures in the foreground, and
 * references as links that hand their target to the page. Detention intelligence and the audit
 * overview both speak this way.
 */
export function NarrativeSegments<S extends NarrativeSegment>({
  segments,
  onTarget,
}: {
  segments: readonly S[];
  onTarget: (segment: S, targetId: string) => void;
}) {
  return (
    <>
      {segments.map((segment, index) => {
        const targetId = segment.targetId;
        if (segment.target && targetId) {
          const open = () => onTarget(segment, targetId);
          const onKeyDown = (event: KeyboardEvent<HTMLSpanElement>) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              open();
            }
          };
          return (
            <span
              key={index}
              role="link"
              tabIndex={0}
              onClick={open}
              onKeyDown={onKeyDown}
              className={cn(
                "ui-focus-ring decoration-border-strong hover:decoration-foreground cursor-pointer rounded-sm underline underline-offset-4",
                toneClass(segment.tone),
              )}
            >
              {segment.text}
            </span>
          );
        }
        if (segment.strong) {
          return (
            <span key={index} className={cn("font-medium", toneClass(segment.tone))}>
              {segment.text}
            </span>
          );
        }
        return <span key={index}>{segment.text}</span>;
      })}
    </>
  );
}
