import { waitIdOfNote } from "@/components/assistant/use-conversation-waits";
import type { AgentWait } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { DeskIcon } from "../desk-icons";
import { turnTime } from "../turn-time";

/**
 * Where the agent picked up work it had parked on a wait: what it waited
 * for and what came of it. The note the agent read is its own; the person
 * sees the wait it records.
 */
export function DeskWaitNote({
  content,
  waits,
  timezone,
}: {
  content: string;
  waits: ReadonlyMap<string, AgentWait>;
  timezone: string;
}) {
  const t = useT();
  const id = waitIdOfNote(content);
  const wait = id ? waits.get(id) : undefined;

  return (
    <div className="dk-world" role="note">
      <span className="dk-world-hd">
        <DeskIcon name="play" size={12} />
        {wait?.status === "TimedOut"
          ? t("Picked up after the wait ran out")
          : t("Picked up after a wait")}
      </span>
      {wait ? (
        <>
          <span>
            <b>{wait.description}</b>
            {wait.resolvedAt ? <> · {turnTime(wait.resolvedAt, timezone, t)}</> : null}
          </span>
          {wait.outcome !== "" && <span>{wait.outcome}</span>}
        </>
      ) : (
        <span>{t("The agent came back to work it had set aside.")}</span>
      )}
    </div>
  );
}
