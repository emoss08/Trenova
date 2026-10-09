import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { conversationPath } from "@/lib/conversation-path";
import type { AssistantHandoff } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useNavigate } from "react-router";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { carriedChips } from "./handoff-state";

/**
 * Where a conversation went, or where it came from. In the conversation
 * handed off it says which agent took it and what went with it, and leads
 * there; in the conversation it started it says where it came from and leads
 * back.
 */
export function DeskHandoffCard({
  handoff,
  incoming,
  time,
  agentsById,
}: {
  handoff: AssistantHandoff;
  /** The card opens the conversation the hand-off started, rather than the one it left. */
  incoming: boolean;
  time: string;
  agentsById: ReadonlyMap<string, AgentChoice>;
}) {
  const t = useT();
  const navigate = useNavigate();
  const otherId = incoming ? handoff.fromAgentId : handoff.toAgentId;
  const otherName = incoming ? handoff.fromAgentName : handoff.toAgentName;
  const other = agentsById.get(otherId) ?? { id: otherId, name: otherName };
  const targetThread = incoming ? handoff.fromThreadId : handoff.toThreadId;

  return (
    <div className="dk-ho">
      <div className="dk-ho-h">
        <DeskAgentTile agent={other} size="sm" />
        <span>
          <b>
            {incoming ? t("Handed over from {0}", otherName) : t("Handed off to {0}", otherName)}
          </b>
          <em>
            {incoming
              ? t("{0} · the earlier conversation stays open", time)
              : t("{0} · this conversation stays open here", time)}
          </em>
        </span>
      </div>
      {carriedChips(handoff).length > 0 && (
        <div className="dk-ho-c">
          <span>{t("Carried over")}</span>
          <div>
            {carriedChips(handoff).map((chip) => (
              <span
                key={chip.kind === "artifact" ? `a:${chip.title}` : chip.kind}
                className="dk-ho-chip"
              >
                <DeskIcon name="check" size={10} stroke={2.6} />
                {chip.kind === "summary"
                  ? t("Summary of this conversation")
                  : chip.kind === "facts"
                    ? t("{0, plural, one {# pinned fact} other {# pinned facts}}", chip.count)
                    : chip.title}
              </span>
            ))}
          </div>
        </div>
      )}
      <Button
        className="h-7.5 gap-1.25 self-start rounded-lg px-3 has-[>svg]:px-3"
        onClick={() => void navigate(conversationPath(targetThread))}
      >
        {incoming ? t("Open the earlier conversation") : t("Open their conversation")}
        <DeskIcon name="chevR" size={12} />
      </Button>
    </div>
  );
}
