import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import type { AIControlSegment } from "@/lib/graphql/ai-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type AnimationEvent, type KeyboardEvent, type ReactNode } from "react";
import { novaTarget, type NovaTarget } from "./use-nova-segments";

type NovaSummaryProps = {
  /** The tab the sentence heads, as it is named beside Nova. */
  context: string;
  segments: readonly AIControlSegment[] | undefined;
  /** Nova is still writing the sentence (the summary is pending), so the badge turns. */
  writing?: boolean;
  loading?: boolean;
  onTarget: (target: NovaTarget) => void;
  /** The one control the tab offers beside its sentence, with its note under it. */
  control?: ReactNode;
};

/**
 * The sentence that heads every tab of AI control: what is true now and what needs a
 * person, with links to where to act on it, and one control beside it.
 *
 * Nova's badge turns only while Nova is doing something: writing the sentence
 * (loading, or the summary still pending) or saying it as it arrives. It stops once
 * the sentence has landed; agents at work show that on their own rows.
 */
export function NovaSummary({
  context,
  segments,
  writing = false,
  loading = false,
  onTarget,
  control,
}: NovaSummaryProps) {
  const t = useT();
  const sentence = (segments ?? []).map((segment) => segment.text).join("");
  // The sentence whose entrance has finished; any other is still being said.
  const [said, setSaid] = useState<string | null>(null);
  const saying = !loading && said !== sentence;
  const onAnimationEnd = (event: AnimationEvent<HTMLParagraphElement>) => {
    if (event.target === event.currentTarget) {
      setSaid(sentence);
    }
  };

  return (
    <section className="hero rh">
      <div className="hero-s">
        <span className="who">
          <NovaBadge size="lg" spin={writing || loading || saying ? "always" : "none"} />
          <b>{t("Nova")}</b>
          <span>{context}</span>
        </span>
        {loading ? (
          <div className="mt-2.5 flex flex-col gap-2" aria-busy>
            <Skeleton className="h-6 w-11/12" />
            <Skeleton className="h-6 w-2/3" />
          </div>
        ) : (
          <p key={sentence} className="say" onAnimationEnd={onAnimationEnd}>
            {(segments ?? []).map((segment, index) => (
              <Segment key={index} segment={segment} onTarget={onTarget} />
            ))}
          </p>
        )}
      </div>
      {control && <div className="hc">{control}</div>}
    </section>
  );
}

function Segment({
  segment,
  onTarget,
}: {
  segment: AIControlSegment;
  onTarget: (target: NovaTarget) => void;
}) {
  const target = novaTarget(segment);
  if (target) {
    const open = () => onTarget(target);
    const onKeyDown = (event: KeyboardEvent<HTMLSpanElement>) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        open();
      }
    };
    return (
      <span
        role="link"
        tabIndex={0}
        className={cn("ref", segment.tone === "danger" && "d", segment.tone === "warn" && "w")}
        onClick={open}
        onKeyDown={onKeyDown}
      >
        {segment.text}
      </span>
    );
  }
  if (segment.strong) {
    return (
      <b className={cn(segment.tone === "danger" && "t-d", segment.tone === "warn" && "t-w")}>
        {segment.text}
      </b>
    );
  }
  return <>{segment.text}</>;
}
