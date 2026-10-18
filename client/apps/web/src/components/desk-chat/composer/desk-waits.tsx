import { PortalCodeForm } from "@/components/assistant/portal-code-form";
import type { ConversationWaits } from "@/components/assistant/use-conversation-waits";
import type { AgentWait, AgentWaitKind, AgentWaitWebWatch } from "@/types/assistant";
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
  PortalRun: "globe",
  DocumentRead: "file",
  WebWatch: "radar",
};

/**
 * What the conversation's agent is waiting on, above the composer: each wait
 * in the agent's words, when it comes due or gives up, and a way to cancel
 * it. Nothing is running while these wait; the agent comes back on its own.
 * A portal task stopped at a sign-in code the portal sent the person shows
 * where to enter it. A web watch's wait ends only with its watch, so its
 * action stops the watch rather than cancelling the wait.
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
            {wait.signInCode && (
              <PortalCodeForm
                className="dk-wq-ed"
                runId={wait.signInCode.portalRunId}
                prompt={wait.signInCode.prompt}
                deadline={turnTime(wait.signInCode.expiresAt, timezone, t)}
                attempts={wait.signInCode.attempts}
              />
            )}
            <span className="dk-wq-acts">
              {wait.kind === "WebWatch" ? (
                wait.webWatch && (
                  <Button
                    variant="quiet"
                    size="bare"
                    className={deskSmallIconClass}
                    title={t("Stop watching")}
                    aria-label={t("Stop watching")}
                    onClick={() => void waits.stopWatch(wait)}
                  >
                    <DeskIcon name="x" size={13} />
                  </Button>
                )
              ) : (
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
              )}
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
    case "PortalRun":
      return t("A portal task");
    case "DocumentRead":
      return t("A file");
    case "WebWatch":
      return t("A web watch");
  }
}

function whenText(wait: AgentWait, timezone: string, t: TranslateFn): string {
  // A file's wait comes due only to look at the file again; the agent comes
  // back the moment the reading finishes, not at that time.
  if (wait.kind === "DocumentRead") {
    return t("Being read");
  }
  if (wait.signInCode) {
    return t("Waiting for your sign-in code");
  }
  if (wait.webWatch) {
    return webWatchWhen(wait.webWatch, timezone, t);
  }
  if (wait.dueAt) {
    return t("Due {0}", turnTime(wait.dueAt, timezone, t));
  }

  return t("Gives up {0}", turnTime(wait.expiresAt, timezone, t));
}

/** How often a web watch looks, until when. */
function webWatchSchedule(cadence: string, until: string, t: TranslateFn): string {
  switch (cadence) {
    case "SixHours":
      return t("Looks every six hours until {0}", until);
    case "Weekly":
      return t("Looks weekly until {0}", until);
    default:
      return t("Looks daily until {0}", until);
  }
}

/** How often a web watch looks and until when, with what it has found so far. */
function webWatchWhen(watch: AgentWaitWebWatch, timezone: string, t: TranslateFn): string {
  const schedule = webWatchSchedule(watch.cadence, turnTime(watch.endsAt, timezone, t), t);
  if (watch.findingCount === 0) {
    return schedule;
  }

  return [
    schedule,
    t("{0, plural, one {# finding so far} other {# findings so far}}", watch.findingCount),
  ].join(" · ");
}
