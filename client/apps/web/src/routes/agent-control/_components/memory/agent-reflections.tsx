import {
  AGENT_REFLECTIONS_KEY,
  fetchRecentAgentReflections,
  type AgentReflection,
} from "@/lib/graphql/agent-reflections";
import type { AgentReflectionAction } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { Ic } from "../kit/ic";
import { reflectionSignalLabel } from "./reflection-signals";

export const agentReflectionsQueryKey = [AGENT_REFLECTIONS_KEY] as const;

const ACTION_LABEL: Record<AgentReflectionAction, string> = defineLabels({
  Saved: "Kept",
  Suggested: "Offered",
  Refreshed: "Already kept",
  Refused: "Not kept",
});

const ACTION_TONE: Record<AgentReflectionAction, string> = {
  Saved: "k",
  Suggested: "w",
  Refreshed: "",
  Refused: "",
};

/**
 * What agents taught themselves lately: each time one looked back over a conversation
 * that went quiet or a run that settled and found something worth a decision, with what
 * made it look and what it kept, offered or turned down and why. A lesson kept here is
 * an ordinary memory below, which a person can edit or retire like any other.
 */
export function AgentReflections() {
  const t = useT();
  const reflectionsQuery = useQuery({
    queryKey: agentReflectionsQueryKey,
    queryFn: ({ signal }) => fetchRecentAgentReflections({ signal }),
  });

  const reflections = reflectionsQuery.data ?? [];
  if (reflections.length === 0) {
    return null;
  }

  return (
    <section className="sec">
      <header className="sh2">
        <Ic n="brain" s={14} />
        <h3>{t("What agents learned")}</h3>
        <em className="mono">{reflections.length}</em>
        <span className="sp" />
        <span className="sh2-n">{t("Each lesson kept is a memory in the list below")}</span>
      </header>
      {reflections.map((reflection) => (
        <ReflectionRow key={reflection.id} reflection={reflection} />
      ))}
    </section>
  );
}

function ReflectionRow({ reflection }: { reflection: AgentReflection }) {
  const t = useT();
  const where = reflection.subjectType === "Thread" ? t("A conversation") : t("A background run");

  return (
    <div className="sgm">
      <p className="sgm-e">
        {where} · {formatUnixDateTimeMedium(reflection.createdAt)}
        {reflection.tainted && <> · {t("read content written outside the organization")}</>}
      </p>
      {reflection.signals.length > 0 && (
        <ul className="rf-s" aria-label={t("Why it looked")}>
          {reflection.signals.map((signal) => (
            <li key={signal.kind} className="tg">
              {reflectionSignalLabel(signal.kind, t)}
              {signal.detail ? ` · ${signal.detail}` : ""}
            </li>
          ))}
        </ul>
      )}
      {reflection.status === "Failed" ? (
        <p className="sgm-e t-d">
          {t("The look back could not finish: {0}", reflection.errorMessage)}
        </p>
      ) : reflection.changes.length === 0 ? (
        <p className="sgm-c">{reflection.notes || t("Nothing worth keeping.")}</p>
      ) : (
        <>
          <ul className="rf-l" aria-label={t("Lessons")}>
            {reflection.changes.map((change, index) => (
              <li key={change.memoryId ?? `${reflection.id}-${index}`} className="rf-c">
                <span className={cn("tg", ACTION_TONE[change.action])}>
                  {t(ACTION_LABEL[change.action])}
                </span>
                <span className="sgm-c">{change.content}</span>
                {change.action === "Refused" && change.reason && (
                  <span className="sgm-e">{change.reason}</span>
                )}
              </li>
            ))}
          </ul>
          {reflection.notes && <p className="sgm-e">{reflection.notes}</p>}
        </>
      )}
    </div>
  );
}
