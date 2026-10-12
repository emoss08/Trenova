import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { KeyboardEvent, ReactNode } from "react";

type HeroProps = {
  /** The section Nova is speaking about, beside its name. */
  context: string;
  /** Whether something the sentence describes is still under way. */
  working?: boolean;
  /** A key that replays the sentence's entrance when what it says changes. */
  sentenceKey?: string;
  /** The one action the sentence suggests, with a note under it. */
  control?: ReactNode;
  children: ReactNode;
};

/** Nova's sentence at the head of a section, and the one action it suggests. */
export function Hero({ context, working = false, sentenceKey, control, children }: HeroProps) {
  const t = useT();

  return (
    <section className="hero rh">
      <div className="hero-s">
        <span className="who">
          <span className={cn("dm", working && "spin")} />
          <b>{t("Nova")}</b>
          <span>{context}</span>
        </span>
        <p className="say" key={sentenceKey}>
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
