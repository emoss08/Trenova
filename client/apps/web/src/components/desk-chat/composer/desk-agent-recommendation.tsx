import type { AgentChoice, AgentRecommendation } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { DeskAgentTile } from "../desk-agent-tile";
import { deskLinkClass, deskSmallIconClass } from "../desk-button-styles";
import { DeskIcon } from "../desk-icons";

/**
 * A small note floating just above the composer, where the terms reminder
 * sits, naming the agent that holds the tools a question needs while the
 * question would go to another: Switch to ask it, or the cross to keep the
 * agent chosen. The name stays whole and the sentence gives way on a narrow
 * screen.
 */
export function DeskAgentRecommendation({
  recommendation,
  agent,
  onUse,
  onKeep,
}: {
  recommendation: AgentRecommendation;
  /** The agent the question would go to now. */
  agent: AgentChoice;
  onUse: () => void;
  onKeep: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const suggested = recommendation.agent;

  return (
    <div className="dk-arec-w">
      <div className="dk-arec" data-slot="agent-recommendation">
        <DeskAgentTile agent={suggested} size="xs" />
        <span className="dk-arec-t" role="status">
          {rt(
            "<b>{0}</b> has the tools for this",
            { b: (chunk) => <b>{chunk}</b> },
            suggested.name,
          )}
        </span>
        <Button variant="bare" size="bare" className={deskLinkClass} onClick={onUse}>
          {t("Switch")}
        </Button>
        <Button
          variant="bare"
          size="bare"
          className={deskSmallIconClass}
          aria-label={t("Keep {0}", agent.name)}
          title={t("Keep {0}", agent.name)}
          onClick={onKeep}
        >
          <DeskIcon name="x" size={12} />
        </Button>
      </div>
    </div>
  );
}
