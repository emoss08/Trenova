import type { AgentQualityOverview } from "@/lib/graphql/agent-quality";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { Hero, Ref } from "../kit/hero";
import { formatShare, qualityHeroFacts } from "./quality-model";
import { hourLabel } from "./sweep-settings-model";

type QualityHeroProps = {
  overview: AgentQualityOverview;
  /** Opens the agent that regressed furthest. */
  onOpenAgent: (agentId: string) => void;
};

/** Nova's sentence on how the agents score and what people think, and the agent to look at. */
export function QualityHero({ overview, onOpenAgent }: QualityHeroProps) {
  const t = useT();
  const rt = useRichT();
  const facts = qualityHeroFacts(overview);
  const strong = (children: ReactNode) => <b>{children}</b>;
  const worst = facts.worst;
  const sweep = overview.sweepEnabled
    ? t("Nightly sweep · {0}", hourLabel(overview.nextSweepHourLocal))
    : t("The nightly sweep is off");

  return (
    <Hero
      context={t("Quality")}
      control={
        worst && (
          <>
            <button type="button" className="btn ink lg" onClick={() => onOpenAgent(worst.agentId)}>
              {t("Look at {0}", worst.agentName)}
            </button>
            <span>{sweep}</span>
          </>
        )
      }
    >
      {facts.score === null
        ? facts.noCases
          ? t("No agent has a golden set yet, so nothing is scored.")
          : t("No agent has been scored against its golden set yet.")
        : facts.liked === null
          ? rt(
              "Agents score <b>{0}</b> against their golden sets.",
              { b: strong },
              formatShare(facts.score),
            )
          : rt(
              "Agents score <b>{0}</b> against their golden sets, and people liked <b>{1}</b> of the answers they rated.",
              { b: strong },
              formatShare(facts.score),
              formatShare(facts.liked),
            )}
      {facts.score === null && facts.liked !== null && (
        <>
          {" "}
          {rt(
            "People liked <b>{0}</b> of the answers they rated.",
            { b: strong },
            formatShare(facts.liked),
          )}
        </>
      )}
      {worst && (
        <>
          {" "}
          {worst.points !== null && worst.points < 0
            ? rt(
                "<a>{0}</a> dropped {1, plural, one {# point} other {# points}} after its last change.",
                {
                  a: (children) => (
                    <Ref tone="d" onOpen={() => onOpenAgent(worst.agentId)}>
                      {children}
                    </Ref>
                  ),
                },
                worst.agentName,
                Math.abs(worst.points),
              )
            : rt(
                "<a>{0}</a> regressed after its last change.",
                {
                  a: (children) => (
                    <Ref tone="d" onOpen={() => onOpenAgent(worst.agentId)}>
                      {children}
                    </Ref>
                  ),
                },
                worst.agentName,
              )}
        </>
      )}
    </Hero>
  );
}
