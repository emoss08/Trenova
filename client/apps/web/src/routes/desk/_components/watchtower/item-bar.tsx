import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { MessageChatSquareIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigate } from "react-router";
import {
  QUEUE_NAV_CLASS,
  QueueBar,
  QueueBarSpacer,
  QueueButton,
  QueueNav,
} from "../queue/queue-bar";
import { WatchtowerHandOff } from "./hand-off-menu";
import type { useWatchtowerActions } from "./use-watchtower-actions";

export type WatchtowerActions = ReturnType<typeof useWatchtowerActions>;

/** A decision is approved where its preview is checked: on Decisions, never in a queue. */
function isDecision(item: WatchtowerItem): boolean {
  return item.sourceKind === "AgentProposal" || item.sourceKind === "AgentPlan";
}

/**
 * The bar under the item in front of the person: walk the queue, ask about
 * the item, and act on it. A decision opens on Decisions, where its dry run
 * is read; a message opens in Triage, where its next step is; everything
 * else is snoozed, done or handed to an agent here.
 */
export function WatchtowerItemBar({
  item,
  actions,
  canAct,
}: {
  item: WatchtowerItem;
  actions: WatchtowerActions;
  canAct: boolean;
}) {
  const t = useT();
  const navigate = useNavigate();
  const decision = isDecision(item);
  const message = item.sourceKind === "InboundMessage";

  return (
    <QueueBar>
      <QueueNav onPrevious={actions.previous} onNext={actions.next} />
      <Button
        variant="quiet"
        size="bare"
        className={cn(QUEUE_NAV_CLASS, "dk-w3-ask ml-1.5 inline-flex h-7.5 items-center")}
        onClick={actions.ask}
        disabled={!actions.canAsk}
        title={t("Ask about this")}
      >
        <MessageChatSquareIcon aria-hidden className="size-3.5" />
        <span>{t("Ask about this")}</span>
      </Button>
      <QueueBarSpacer />
      {decision && item.path !== "" ? (
        <QueueButton ink onClick={() => void navigate(item.path)}>
          {t("Review")}
        </QueueButton>
      ) : canAct ? (
        <>
          {message && item.path !== "" && (
            <QueueButton onClick={() => void navigate(item.path)}>
              {t("Open in Triage")}
            </QueueButton>
          )}
          <QueueButton shortcut="S" onClick={actions.snooze}>
            {t("Snooze")}
          </QueueButton>
          <QueueButton shortcut="E" onClick={actions.done}>
            {t("Done")}
          </QueueButton>
          {item.agentWork === null && item.subjectId !== null && (
            <WatchtowerHandOff
              suggested={item.suggestedAgent}
              open={actions.menuOpen}
              onOpenChange={actions.setMenuOpen}
              onHandOff={actions.handOff}
              disabled={actions.handingOff}
            />
          )}
        </>
      ) : (
        <span className="dk-dc2-qe">{t("You can read these, but not act on them.")}</span>
      )}
    </QueueBar>
  );
}
