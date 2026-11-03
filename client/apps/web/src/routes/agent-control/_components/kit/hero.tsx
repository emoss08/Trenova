import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type AnimationEvent, type KeyboardEvent, type ReactNode } from "react";

type HeroProps = {
  /** The section Nova is speaking about, beside its name. */
  context: string;
  /** Nova is still writing the sentence: what it reads is loading. */
  writing?: boolean;
  /** A key that replays the sentence's entrance when what it says changes. */
  sentenceKey?: string;
  /** The one action the sentence suggests, with a note under it. */
  control?: ReactNode;
  children: ReactNode;
};

/**
 * Nova's sentence at the head of a section, and the one action it suggests.
 *
 * Nova's badge turns only while Nova is doing something: writing the sentence, or
 * saying it as it arrives. It stops when the sentence has landed, whatever the
 * sentence reports is still under way, since agents at work or an index being built
 * show that themselves.
 */
export function Hero({ context, writing = false, sentenceKey, control, children }: HeroProps) {
  const t = useT();
  const key = sentenceKey ?? "";
  // The sentence whose entrance has finished; any other is still being said.
  const [said, setSaid] = useState<string | null>(null);
  const saying = said !== key;
  const onAnimationEnd = (event: AnimationEvent<HTMLParagraphElement>) => {
    if (event.target === event.currentTarget) {
      setSaid(key);
    }
  };

  return (
    <section className="hero rh">
      <div className="hero-s">
        <span className="who">
          <NovaBadge size="lg" spin={writing || saying ? "always" : "none"} />
          <b>{t("Nova")}</b>
          <span>{context}</span>
        </span>
        <p className="say" key={sentenceKey} onAnimationEnd={onAnimationEnd}>
          {children}
        </p>
      </div>
      {control && <div className="hc">{control}</div>}
    </section>
  );
}

type RefProps = {
  /** "d" for something gone wrong, "w" for something waiting. */
  tone?: "d" | "w";
  onOpen: () => void;
  children: ReactNode;
};

/** A phrase in Nova's sentence that opens what it names. */
export function Ref({ tone, onOpen, children }: RefProps) {
  const onKeyDown = (event: KeyboardEvent<HTMLSpanElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onOpen();
    }
  };

  return (
    <span
      role="link"
      tabIndex={0}
      className={cn("ref", tone)}
      onClick={onOpen}
      onKeyDown={onKeyDown}
    >
      {children}
    </span>
  );
}
