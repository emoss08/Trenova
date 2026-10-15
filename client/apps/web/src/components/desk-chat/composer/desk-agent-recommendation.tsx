import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";
import type { OfferedMatch } from "../agent-recommendation-state";
import { DeskAgentTile } from "../desk-agent-tile";
import { deskLinkClass, deskSmallIconClass } from "../desk-button-styles";
import { DeskIcon } from "../desk-icons";

/**
 * A small note floating just above the composer, where the terms reminder
 * sits, naming the agent that holds the tools a question needs while the
 * question would go to another. The name and sentence open why: the matched
 * tools and that the agent chosen holds none of them. Switch (Hand off in a
 * conversation) takes the agent, the cross keeps the one chosen. The name
 * stays whole and the sentence gives way on a narrow screen.
 */
export function DeskAgentRecommendation({
  match,
  agent,
  action = "switch",
  disabled = false,
  onUse,
  onKeep,
}: {
  match: OfferedMatch;
  /** The agent the question would go to now. */
  agent: AgentChoice;
  /** Switch before a conversation starts; hand the conversation off within one. */
  action?: "switch" | "handoff";
  /** A question is being sent: the choice it was sent with stands. */
  disabled?: boolean;
  onUse: () => void;
  onKeep: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const suggested = match.agent;

  return (
    <div className="dk-arec-w">
      <div className="dk-arec" data-slot="agent-recommendation">
        <Popover>
          <PopoverTrigger
            render={<Button variant="bare" size="bare" className="dk-arec-why" />}
            aria-label={t("Why {0}", suggested.name)}
          >
            <DeskAgentTile agent={suggested} size="xs" />
            <span className="dk-arec-t">
              {rt(
                "<b>{0}</b> <u>has the tools for this</u>",
                {
                  b: (chunk) => <b>{chunk}</b>,
                  u: (chunk) => <span className="dk-arec-u">{chunk}</span>,
                },
                suggested.name,
              )}
            </span>
          </PopoverTrigger>
          <PopoverContent
            side="top"
            align="start"
            sideOffset={10}
            className="w-80 gap-0 rounded-lg bg-dsk-raised p-0 text-dsk-fg ui-lift-float"
          >
            <div className="dk-arec-pop" data-slot="agent-recommendation-why">
              <div className="dk-arec-pop-h">{t("Why {0}", suggested.name)}</div>
              <ul>
                {match.tools.map((tool) => (
                  <li key={tool.name}>
                    <DeskIcon name="check" size={12} />
                    <span>
                      <b>{describeToolCall(tool.name, null).title}</b>
                      {tool.summary !== "" && <span>{tool.summary}</span>}
                    </span>
                  </li>
                ))}
              </ul>
              <div className="dk-arec-pop-f">{t("{0} holds none of these.", agent.name)}</div>
            </div>
          </PopoverContent>
        </Popover>
        <Button
          variant="bare"
          size="bare"
          className={deskLinkClass}
          disabled={disabled}
          onClick={onUse}
        >
          {action === "handoff" ? t("Hand off") : t("Switch")}
        </Button>
        <Button
          variant="bare"
          size="bare"
          className={deskSmallIconClass}
          disabled={disabled}
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

/**
 * Said in the same place when tools match the question but none of the
 * person's agents holds them. Someone who manages agents is pointed to where
 * tools are added; anyone else is told who can add them.
 */
export function DeskNoAgentFits({
  canManage,
  onDismiss,
}: {
  canManage: boolean;
  onDismiss: () => void;
}) {
  const t = useT();

  return (
    <div className="dk-arec-w">
      <div className="dk-arec dk-arec-none" data-slot="agent-recommendation-none">
        <DeskIcon name="info" size={13} />
        <span className="dk-arec-t">
          {canManage
            ? t("None of your agents has the tools for this.")
            : t("None of your agents has the tools for this. An administrator can add them.")}
        </span>
        {canManage && (
          <Link className={deskLinkClass} to="/admin/agent-control?tab=agents">
            {t("Add tools")}
          </Link>
        )}
        <Button
          variant="bare"
          size="bare"
          className={deskSmallIconClass}
          aria-label={t("Dismiss")}
          title={t("Dismiss")}
          onClick={onDismiss}
        >
          <DeskIcon name="x" size={12} />
        </Button>
      </div>
    </div>
  );
}
