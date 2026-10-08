import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import type { KeyboardEvent, ReactNode } from "react";
import type { SafetyFacts } from "./safety-model";

type SafetyHeroProps = {
  facts: SafetyFacts;
  onShowRunning: () => void;
  onReviewOpen: () => void;
};

function Ref({
  tone,
  onOpen,
  children,
}: {
  tone?: string;
  onOpen: () => void;
  children: ReactNode;
}) {
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
      className={tone ? `ref ${tone}` : "ref"}
      onClick={onOpen}
      onKeyDown={onKeyDown}
    >
      {children}
    </span>
  );
}

/** Nova's sentence on what agents can do without a person, and the review it suggests. */
export function SafetyHero({ facts, onShowRunning, onReviewOpen }: SafetyHeroProps) {
  const t = useT();
  const rt = useRichT();

  return (
    <section className="hero rh">
      <div className="hero-s">
        <span className="who">
          <span className="dm" />
          <b>{t("Nova")}</b>
          <span>{t("Safety")}</span>
        </span>
        <p className="say">
          {facts.runs > 0
            ? rt(
                "<run>{0, plural, one {# tool runs} other {# tools run}}</run> without a person, all inside the organization.",
                { run: (children) => <Ref onOpen={onShowRunning}>{children}</Ref> },
                facts.runs,
              )
            : t("No tool runs without a person.")}{" "}
          {rt(
            "<b>{0, plural, one {# tool} other {# tools}}</b> can send outside the organization, and every one waits for approval.",
            { b: (children) => <b>{children}</b> },
            facts.leave,
          )}
          {facts.open.length > 0 && (
            <>
              {" "}
              {rt(
                "<open>{0, plural, one {# agent} other {# agents}}</open> everyone can use hold some of them.",
                {
                  open: (children) => (
                    <Ref tone="w" onOpen={onReviewOpen}>
                      {children}
                    </Ref>
                  ),
                },
                facts.open.length,
              )}
            </>
          )}
        </p>
      </div>
      {facts.open.length > 0 && (
        <div className="hc">
          <button type="button" className="btn ink lg" onClick={onReviewOpen}>
            {t("Review open agents")}
          </button>
          <span>{t("Compare what they can do")}</span>
        </div>
      )}
    </section>
  );
}
