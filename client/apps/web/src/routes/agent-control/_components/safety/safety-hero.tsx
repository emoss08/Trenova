import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { Hero, Ref } from "../kit/hero";
import type { SafetyFacts } from "./safety-model";

type SafetyHeroProps = {
  facts: SafetyFacts;
  onShowRunning: () => void;
  onReviewOpen: () => void;
};

/** Nova's sentence on what agents can do without a person, and the review it suggests. */
export function SafetyHero({ facts, onShowRunning, onReviewOpen }: SafetyHeroProps) {
  const t = useT();
  const rt = useRichT();

  return (
    <Hero
      context={t("Safety")}
      control={
        facts.open.length > 0 && (
          <>
            <button type="button" className="btn ink lg" onClick={onReviewOpen}>
              {t("Review open agents")}
            </button>
            <span>{t("Compare what they can do")}</span>
          </>
        )
      }
    >
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
    </Hero>
  );
}
