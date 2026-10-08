import { formatCompactAge } from "@trenova/shared/lib/date";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { Hero, Ref } from "../kit/hero";
import type { ActivityFacts } from "./activity-model";

type ActivityHeroProps = {
  facts: ActivityFacts;
  onShowFailed: () => void;
  onShowProposals: () => void;
  onShowExceptions: () => void;
};

/** Nova's sentence on today's runs and what waits on a person, and the decision it suggests. */
export function ActivityHero({
  facts,
  onShowFailed,
  onShowProposals,
  onShowExceptions,
}: ActivityHeroProps) {
  const t = useT();
  const rt = useRichT();
  const pending = facts.pending ?? 0;
  const open = facts.openExceptions ?? 0;

  return (
    <Hero
      context={t("Activity")}
      working={facts.working > 0}
      control={
        pending > 0 && (
          <>
            <button type="button" className="btn ink lg" onClick={onShowProposals}>
              {t("{0, plural, one {Decide # proposal} other {Decide # proposals}}", pending)}
            </button>
            {facts.oldestWaitSeconds !== null && (
              <span>{t("Oldest waiting {0}", formatCompactAge(facts.oldestWaitSeconds))}</span>
            )}
          </>
        )
      }
    >
      {facts.failed > 0
        ? rt(
            "<b>{0, plural, one {# run} other {# runs}}</b> today, <f>{1} failed</f>.",
            {
              b: (children) => <b>{children}</b>,
              f: (children) => (
                <Ref tone="d" onOpen={onShowFailed}>
                  {children}
                </Ref>
              ),
            },
            facts.runs,
            facts.failed,
          )
        : rt(
            "<b>{0, plural, one {# run} other {# runs}}</b> today.",
            { b: (children) => <b>{children}</b> },
            facts.runs,
          )}
      {facts.pending !== null && (
        <>
          {" "}
          {pending > 0
            ? rt(
                "<p>{0, plural, one {# proposal waits} other {# proposals wait}}</p> on a person.",
                {
                  p: (children) => (
                    <Ref tone="w" onOpen={onShowProposals}>
                      {children}
                    </Ref>
                  ),
                },
                pending,
              )
            : t("Nothing waits on a person.")}
        </>
      )}
      {open > 0 && (
        <>
          {" "}
          {rt(
            "<e>{0, plural, one {# exception is} other {# exceptions are}}</e> still open.",
            {
              e: (children) => <Ref onOpen={onShowExceptions}>{children}</Ref>,
            },
            open,
          )}
        </>
      )}
    </Hero>
  );
}
