import { humanizeToolName } from "@/components/assistant/proposal-state";
import type { AgentChoice, AgentRecommendation } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { translateLabel } from "@trenova/shared/i18n/labels";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { DeskAgentTile } from "../desk-agent-tile";

/**
 * The agent that holds the tools a question needs, offered under the
 * composer while the question would go to another: which agent, the tools of
 * its that matched, and a choice to ask it or keep the one chosen. Until the
 * person chooses, it says that sending will ask the suggested agent, since
 * that is what Enter does.
 */
export function DeskAgentRecommendation({
  recommendation,
  agent,
  sendsToIt,
  onUse,
  onKeep,
}: {
  recommendation: AgentRecommendation;
  /** The agent the question would go to now. */
  agent: AgentChoice;
  /** Sending now asks the suggested agent, because nobody has chosen yet. */
  sendsToIt: boolean;
  onUse: () => void;
  onKeep: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const suggested = recommendation.agent;

  return (
    <div className="dk-arec" role="status" aria-live="polite" data-slot="agent-recommendation">
      <DeskAgentTile agent={suggested} size="xs" />
      <div className="dk-arec-body">
        <span className="dk-arec-t">
          {sendsToIt
            ? rt(
                "<b>{0}</b> has the tools for this, so sending asks it",
                { b: (chunk) => <b>{chunk}</b> },
                suggested.name,
              )
            : rt(
                "<b>{0}</b> has the tools for this",
                { b: (chunk) => <b>{chunk}</b> },
                suggested.name,
              )}
        </span>
        {recommendation.tools.length > 0 && (
          <span className="dk-arec-tools" aria-label={t("Matching tools")}>
            {recommendation.tools.map((tool) => (
              <span key={tool} className="dk-arec-tool">
                {translateLabel(humanizeToolName(tool))}
              </span>
            ))}
          </span>
        )}
      </div>
      <span className="dk-arec-acts">
        <Button variant="quiet" size="xs" className="dk-arec-btn" onClick={onKeep}>
          {t("Keep {0}", agent.name)}
        </Button>
        <Button variant="secondary" size="xs" className="dk-arec-btn" onClick={onUse}>
          {t("Ask {0}", suggested.name)}
        </Button>
      </span>
    </div>
  );
}
