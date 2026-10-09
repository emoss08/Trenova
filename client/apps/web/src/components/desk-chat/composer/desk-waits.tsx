import type { ConversationWaits } from "@/components/assistant/use-conversation-waits";
import type { AgentWait, AgentWaitKind } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { DeskIcon, type DeskIconName } from "../desk-icons";
import { deskSmallIconClass } from "../desk-button-styles";
import { turnTime } from "../turn-time";

const KIND_ICONS: Record<AgentWaitKind, DeskIconName> = {
  Time: "clock",
  StopArrival: "truck",
  StopDeparture: "truck",
  Reply: "inbox",
  AppointmentNear: "clock",
  FreeTimeEnding: "clock",
  HOSDriveBelow: "route",
};

/**
 * What the conversation's agent is waiting on, above the composer: each wait
 * in the agent's words, when it comes due or gives up, and a way to cancel
 * it. Nothing is running while these wait; the agent comes back on its own.
 */
export function DeskWaits({ waits, timezone }: { waits: ConversationWaits; timezone: string }) {
  const t = useT();
  if (waits.open.length === 0) {
    return null;
  }

  return (
    <section className="dk-wq" aria-label={t("What the agent is waiting on")}>
      <div className="dk-wq-hd">
        <DeskIcon name="pause" size={12} />
        <span className="dk-wq-tt">
          {t(
            "{0, plural, one {Waiting on # thing} other {Waiting on # things}}",
            waits.open.length,
          )}
        </span>
      </div>
      <ol className="dk-wq-list">
        {waits.open.map((wait) => (
          <li key={wait.id} className="dk-wq-it dk-wt-it">
            <span className="dk-wq-st">
              <DeskIcon name={KIND_ICONS[wait.kind]} size={12} />
              {kindLabel(wait.kind, t)}
            </span>
            <p className="dk-wq-tx" title={wait.nextStep !== "" ? wait.nextStep : wait.description}>
              {wait.description}
            </p>
            <span className="dk-wq-meta">{whenText(wait, timezone, t)}</span>
            <span className="dk-wq-acts">
              <Button
                variant="quiet"
                size="bare"
                className={deskSmallIconClass}
                title={t("Cancel the wait")}
                aria-label={t("Cancel the wait")}
                onClick={() => void waits.cancel(wait)}
              >
                <DeskIcon name="x" size={13} />
              </Button>
            </span>
          </li>
        ))}
      </ol>
    </section>
  );
}

function kindLabel(kind: AgentWaitKind, t: TranslateFn): string {
  switch (kind) {
    case "Time":
      return t("A time");
    case "StopArrival":
      return t("An arrival");
    case "StopDeparture":
      return t("A departure");
    case "Reply":
      return t("A reply");
    case "AppointmentNear":
      return t("An appointment");
    case "FreeTimeEnding":
      return t("Free time");
    case "HOSDriveBelow":
      return t("Drive time");
  }
}

function whenText(wait: AgentWait, timezone: string, t: TranslateFn): string {
  if (wait.dueAt) {
    return t("Due {0}", turnTime(wait.dueAt, timezone, t));
  }

  return t("Gives up {0}", turnTime(wait.expiresAt, timezone, t));
}
