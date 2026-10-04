import { SectionPanel } from "@/components/section-panel";
import {
  AGENT_REFLECTIONS_KEY,
  fetchRecentAgentReflections,
  type AgentReflection,
} from "@/lib/graphql/agent-reflections";
import type { AgentReflectionAction } from "@trenova/graphql/generated/graphql";
import { GraduationHat01Icon } from "@trenova/shared/components/icons";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useQuery } from "@tanstack/react-query";
import { reflectionSignalLabel } from "./reflection-signals";

export const agentReflectionsQueryKey = [AGENT_REFLECTIONS_KEY] as const;

const ACTION: Record<AgentReflectionAction, { label: string; variant: BadgeVariant }> = {
  Saved: { label: "Kept", variant: "success" },
  Suggested: { label: "Offered", variant: "warning" },
  Refreshed: { label: "Already kept", variant: "neutral" },
  Refused: { label: "Not kept", variant: "neutral" },
};

/**
 * What agents taught themselves lately: each time one looked back over a
 * conversation that went quiet or a run that settled and found something
 * worth a decision, with what made it look and what it kept, offered or
 * turned down and why. A lesson kept here is an ordinary memory below, which
 * a person can edit, pause or retire like any other.
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
    <SectionPanel
      title={t("What agents learned")}
      icon={<GraduationHat01Icon />}
      count={reflections.length}
      help={t(
        "Once a conversation goes quiet or a background run settles, its agent looks back over it when something went wrong, took several tries or was corrected. Each lesson it keeps is a memory in the list below.",
      )}
    >
      <ul className="divide-border-subtle flex flex-col divide-y">
        {reflections.map((reflection) => (
          <ReflectionRow key={reflection.id} reflection={reflection} />
        ))}
      </ul>
    </SectionPanel>
  );
}

function ReflectionRow({ reflection }: { reflection: AgentReflection }) {
  const t = useT();
  const where = reflection.subjectType === "Thread" ? t("A conversation") : t("A background run");

  return (
    <li className="flex flex-col gap-2 px-3 py-3">
      <p className="text-foreground-muted text-xs">
        {where} · {formatUnixDateTimeMedium(reflection.createdAt)}
        {reflection.tainted ? <> · {t("read content written outside the organization")}</> : null}
      </p>

      {reflection.signals.length > 0 && (
        <ul aria-label={t("Why it looked")} className="flex flex-wrap gap-1">
          {reflection.signals.map((signal) => (
            <li key={signal.kind}>
              <Badge variant="neutral" appearance="outline">
                {reflectionSignalLabel(signal.kind, t)}
                {signal.detail ? ` · ${signal.detail}` : ""}
              </Badge>
            </li>
          ))}
        </ul>
      )}

      {reflection.status === "Failed" ? (
        <p className="text-danger text-xs">
          {t("The look back could not finish: {0}", reflection.errorMessage)}
        </p>
      ) : reflection.changes.length === 0 ? (
        <p className="text-foreground-muted text-sm">
          {reflection.notes || t("Nothing worth keeping.")}
        </p>
      ) : (
        <>
          <ul aria-label={t("Lessons")} className="flex flex-col gap-1.5">
            {reflection.changes.map((change, index) => (
              <li
                key={change.memoryId ?? `${reflection.id}-${index}`}
                className="flex flex-col gap-0.5"
              >
                <span className="flex items-start gap-2">
                  <Badge variant={ACTION[change.action].variant}>
                    {t(ACTION[change.action].label)}
                  </Badge>
                  <span className="text-sm leading-relaxed">{change.content}</span>
                </span>
                {change.action === "Refused" && change.reason ? (
                  <span className="text-foreground-muted pl-1 text-xs">{change.reason}</span>
                ) : null}
              </li>
            ))}
          </ul>
          {reflection.notes ? (
            <p className="text-foreground-muted text-xs">{reflection.notes}</p>
          ) : null}
        </>
      )}
    </li>
  );
}
